// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/imaptest"
	"github.com/stefanpejcic/golub/internal/imapx"
)

func TestEventsStreamReportsNewMail(t *testing.T) {
	mem := imapmemserver.New()
	u := imaptest.User(t, mem, "alice@golub.test", "secret")
	d := &imapx.Dialer{Endpoint: imaptest.Listen(t, mem)}
	pool := imapx.NewPool(d, time.Minute)
	t.Cleanup(pool.Close)
	h, _ := newAuthServer(t, func(o *Options) {
		o.Mail = pool
		o.Pool = pool
		o.Watcher = d
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := login(t, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events?folder=INBOX", nil)
	req.AddCookie(c)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("%d %v", resp.StatusCode, resp.Header)
	}

	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	waitFor := func(want string) {
		t.Helper()
		deadline := time.After(5 * time.Second)
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					t.Fatalf("stream closed before %q", want)
				}
				if strings.HasPrefix(l, want) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %q", want)
			}
		}
	}
	waitFor("event: ready")
	if _, err := u.Append("INBOX", literal("Subject: hi\r\n\r\nx\r\n"), &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
	}
	waitFor("event: change")
}

func TestEventsLimits(t *testing.T) {
	mem := imapmemserver.New()
	imaptest.User(t, mem, "alice@golub.test", "secret")
	d := &imapx.Dialer{Endpoint: imaptest.Listen(t, mem)}
	pool := imapx.NewPool(d, time.Minute)
	t.Cleanup(pool.Close)
	h, _ := newAuthServer(t, func(o *Options) { o.Mail = pool; o.Pool = pool; o.Watcher = d })
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := login(t, h)

	if rec := serve(h, httptest.NewRequest("GET", "/api/events?folder=INBOX", nil)); rec.Code != 401 {
		t.Fatalf("anonymous: %d", rec.Code)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var open []*http.Response
	for i := 0; i < maxStreamsPerUser+1; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events?folder=INBOX", nil)
		req.AddCookie(c)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		open = append(open, resp)
	}
	last := open[len(open)-1]
	if last.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("stream over the limit: %d", last.StatusCode)
	}
	for _, r := range open {
		r.Body.Close()
	}
}
