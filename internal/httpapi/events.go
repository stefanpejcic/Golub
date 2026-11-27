// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
)

// Watcher pushes mailbox changes over a dedicated IMAP connection.
type Watcher interface {
	Watch(ctx context.Context, user, pass, folder string, notify func(), ready func(total, uidNext uint32)) error
}

// Each stream holds an IMAP connection, so a user gets a few, not one per stray tab.
const maxStreamsPerUser = 3

const pingEvery = 25 * time.Second

type streamCounter struct {
	mu sync.Mutex
	n  map[string]int
}

func (s *streamCounter) acquire(user string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n[user] >= maxStreamsPerUser {
		return false
	}
	s.n[user]++
	return true
}

func (s *streamCounter) release(user string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n[user]--; s.n[user] <= 0 {
		delete(s.n, user)
	}
}

// events streams "change" events for one folder as Server-Sent Events.
func (a *authAPI) events(w http.ResponseWriter, r *http.Request, s auth.Session) {
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "INBOX"
	}
	if !validFolder(folder) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if !a.streams.acquire(s.Email) {
		writeError(w, http.StatusTooManyRequests, "events.too_many")
		return
	}
	defer a.streams.release(s.Email)

	rc := http.NewResponseController(w)
	// The server's write timeout is meant for normal requests, not a stream kept open for hours.
	_ = rc.SetWriteDeadline(time.Time{})
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(event string, data any) error {
		b, _ := json.Marshal(data)
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b); err != nil {
			return err
		}
		return rc.Flush()
	}
	// Headers go out now so the client knows the stream is open; "ready" waits for IDLE.
	if rc.Flush() != nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	changed := make(chan struct{}, 1)
	notify := func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	}
	// buffered, so the watcher never blocks on it
	ready := make(chan map[string]any, 1)
	user, pass := a.imapCreds(s)
	done := make(chan error, 1)
	go func() {
		done <- a.o.Watcher.Watch(ctx, user, pass, folder, notify, func(total, uidNext uint32) {
			ready <- map[string]any{"folder": folder, "total": total, "uidNext": uidNext}
		})
	}()

	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-done:
			if ctx.Err() == nil {
				code := "events.unavailable"
				if errors.Is(err, imapx.ErrNoIdle) {
					code = "events.unsupported"
				} else {
					slog.Warn("imap watch ended", "err", err)
				}
				_ = send("error", map[string]string{"error": code})
			}
			return
		case state := <-ready:
			ready = nil
			if send("ready", state) != nil {
				return
			}
		case <-changed:
			// Coalesce bursts, e.g. a bulk move, into one refresh.
			time.Sleep(300 * time.Millisecond)
			select {
			case state := <-ready:
				ready = nil
				if send("ready", state) != nil {
					return
				}
			case <-changed:
			default:
			}
			if send("change", map[string]string{"folder": folder}) != nil {
				return
			}
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		}
	}
}
