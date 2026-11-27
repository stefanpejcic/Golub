// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

func newTestPool(t *testing.T) (*Pool, *int) {
	t.Helper()
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("alice@golub.test", "secret")
	if err := u.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	mem.AddUser(u)
	p := NewPool(&Dialer{Endpoint: listen(t, mem)}, time.Minute)
	t.Cleanup(p.Close)
	logins := 0
	inner := p.login
	p.login = func(ctx context.Context, user, pass string) (*imapclient.Client, error) {
		logins++
		return inner(ctx, user, pass)
	}
	return p, &logins
}

func noop(c *imapclient.Client) error { return c.Noop().Wait() }

func TestPoolReuses(t *testing.T) {
	p, logins := newTestPool(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := p.Do(ctx, "alice@golub.test", "secret", noop); err != nil {
			t.Fatal(err)
		}
	}
	if *logins != 1 {
		t.Errorf("logins = %d", *logins)
	}
}

func TestPoolKeysOnPassword(t *testing.T) {
	p, _ := newTestPool(t)
	ctx := context.Background()
	if err := p.Do(ctx, "alice@golub.test", "secret", noop); err != nil {
		t.Fatal(err)
	}
	if err := p.Do(ctx, "alice@golub.test", "wrong", noop); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password reused pooled connection: %v", err)
	}
}

func TestPoolReconnects(t *testing.T) {
	p, logins := newTestPool(t)
	ctx := context.Background()
	var first *imapclient.Client
	if err := p.Do(ctx, "alice@golub.test", "secret", func(c *imapclient.Client) error { first = c; return nil }); err != nil {
		t.Fatal(err)
	}
	_ = first.Close()
	if err := p.Do(ctx, "alice@golub.test", "secret", noop); err != nil {
		t.Fatal(err)
	}
	if *logins != 2 {
		t.Errorf("logins = %d", *logins)
	}
}

func TestPoolRetriesOnceAfterConnectionDrop(t *testing.T) {
	p, logins := newTestPool(t)
	ctx := context.Background()
	calls := 0
	err := p.Do(ctx, "alice@golub.test", "secret", func(c *imapclient.Client) error {
		calls++
		if calls == 1 {
			_ = c.Close()
			return c.Noop().Wait()
		}
		return noop(c)
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || *logins != 2 {
		t.Errorf("calls=%d logins=%d", calls, *logins)
	}
}

func TestPoolDoesNotRetryCommandErrors(t *testing.T) {
	p, _ := newTestPool(t)
	calls := 0
	err := p.Do(context.Background(), "alice@golub.test", "secret", func(c *imapclient.Client) error {
		calls++
		_, err := c.Select("Missing", nil).Wait()
		return err
	})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestPoolReapsIdle(t *testing.T) {
	p, logins := newTestPool(t)
	now := time.Now()
	p.now = func() time.Time { return now }
	ctx := context.Background()
	if err := p.Do(ctx, "alice@golub.test", "secret", noop); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	p.reap()
	if n := p.size(); n != 0 {
		t.Fatalf("size after reap = %d", n)
	}
	if err := p.Do(ctx, "alice@golub.test", "secret", noop); err != nil {
		t.Fatal(err)
	}
	if *logins != 2 {
		t.Errorf("logins = %d", *logins)
	}
}

func TestPoolConcurrentSameUser(t *testing.T) {
	p, logins := newTestPool(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Do(context.Background(), "alice@golub.test", "secret", func(c *imapclient.Client) error {
				_, err := c.Select("INBOX", nil).Wait()
				return err
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if *logins != 1 {
		t.Errorf("logins = %d", *logins)
	}
}
