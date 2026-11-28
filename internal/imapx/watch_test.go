// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

func TestWatchReportsNewMail(t *testing.T) {
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("alice@golub.test", "secret")
	if err := u.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	mem.AddUser(u)
	d := &Dialer{Endpoint: listen(t, mem)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan struct{}, 8)
	ready := make(chan [2]uint32, 1)
	done := make(chan error, 1)
	go func() {
		done <- d.Watch(ctx, "alice@golub.test", "secret", "INBOX", func() { changes <- struct{}{} }, func(total, uidNext uint32) { ready <- [2]uint32{total, uidNext} })
	}()
	select {
	case got := <-ready:
		if got != [2]uint32{0, 1} {
			t.Fatalf("ready reported total and uidnext %v, want an empty INBOX", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watcher never became ready")
	}
	raw := "Subject: new\r\n\r\nhello\r\n"
	// the in-memory server confirms IDLE before it listens, so mail appended in that gap goes unreported; try again
	for try := 0; ; try++ {
		if _, err := u.Append("INBOX", lit(raw), &imap.AppendOptions{}); err != nil {
			t.Fatal(err)
		}
		select {
		case <-changes:
		case <-time.After(time.Second):
			if try == 4 {
				t.Fatal("no change reported for new mail")
			}
			continue
		}
		break
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("watch ended with %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not stop on cancel")
	}
}

func TestWatchBadCredentials(t *testing.T) {
	mem := imapmemserver.New()
	mem.AddUser(imapmemserver.NewUser("alice@golub.test", "secret"))
	d := &Dialer{Endpoint: listen(t, mem)}
	err := d.Watch(context.Background(), "alice@golub.test", "nope", "INBOX", func() {}, nil)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v", err)
	}
}
