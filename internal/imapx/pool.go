// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Pool keeps one IMAP connection per user and credential, in memory only.
type Pool struct {
	login func(ctx context.Context, user, pass string) (*imapclient.Client, error)
	check func(ctx context.Context, user, pass string) error
	idle  time.Duration
	now   func() time.Time

	mu    sync.Mutex
	conns map[[32]byte]*pooled
	stop  chan struct{}
	once  sync.Once
}

type pooled struct {
	mu   sync.Mutex
	c    *imapclient.Client
	last time.Time
}

// NewPool starts a pool that closes connections idle for longer than idle.
func NewPool(d *Dialer, idle time.Duration) *Pool {
	p := &Pool{
		login: d.Login,
		check: d.Check,
		idle:  idle,
		now:   time.Now,
		conns: map[[32]byte]*pooled{},
		stop:  make(chan struct{}),
	}
	go p.reapLoop()
	return p
}

// Check verifies credentials without pooling the connection.
func (p *Pool) Check(ctx context.Context, user, pass string) error {
	return p.check(ctx, user, pass)
}

// The password is part of the key so a wrong password can never ride on a pooled login.
func key(user, pass string) [32]byte {
	return sha256.Sum256([]byte(user + "\x00" + pass))
}

// Do runs fn on the user's connection, serialized per user, reconnecting once if the connection dropped.
func (p *Pool) Do(ctx context.Context, user, pass string, fn func(*imapclient.Client) error) error {
	k := key(user, pass)
	p.mu.Lock()
	e := p.conns[k]
	if e == nil {
		e = &pooled{}
		p.conns[k] = e
	}
	p.mu.Unlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	for attempt := 0; ; attempt++ {
		if e.c == nil || closed(e.c) {
			c, err := p.login(ctx, user, pass)
			if err != nil {
				return err
			}
			e.c = c
		}
		e.last = p.now()
		err := fn(e.c)
		if err == nil || attempt > 0 || !closed(e.c) || ctx.Err() != nil {
			return err
		}
		e.c = nil
	}
}

func closed(c *imapclient.Client) bool {
	select {
	case <-c.Closed():
		return true
	default:
		return c.State() == imap.ConnStateLogout
	}
}

func (p *Pool) reapLoop() {
	t := time.NewTicker(p.idle / 4)
	defer t.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-t.C:
			p.reap()
		}
	}
}

func (p *Pool) reap() {
	p.mu.Lock()
	defer p.mu.Unlock()
	cutoff := p.now().Add(-p.idle)
	for k, e := range p.conns {
		if !e.mu.TryLock() {
			continue
		}
		if e.last.Before(cutoff) {
			if e.c != nil {
				_ = e.c.Close()
			}
			delete(p.conns, k)
		}
		e.mu.Unlock()
	}
}

func (p *Pool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}

// Close shuts down every pooled connection.
func (p *Pool) Close() {
	p.once.Do(func() { close(p.stop) })
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, e := range p.conns {
		e.mu.Lock()
		if e.c != nil {
			_ = e.c.Close()
		}
		e.mu.Unlock()
		delete(p.conns, k)
	}
}
