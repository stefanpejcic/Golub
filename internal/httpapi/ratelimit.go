// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"sync"
	"time"
)

const (
	loginMaxFailures = 10
	loginWindow      = 5 * time.Minute
)

// failLimiter counts failed logins per client IP in a fixed window; memory only.
type failLimiter struct {
	mu    sync.Mutex
	now   func() time.Time
	byKey map[string]*window
}

type window struct {
	start time.Time
	n     int
}

func newFailLimiter() *failLimiter {
	return &failLimiter{now: time.Now, byKey: map[string]*window{}}
}

func (l *failLimiter) blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.byKey[key]
	return w != nil && l.now().Sub(w.start) < loginWindow && w.n >= loginMaxFailures
}

func (l *failLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for k, w := range l.byKey {
		if now.Sub(w.start) >= loginWindow {
			delete(l.byKey, k)
		}
	}
	w := l.byKey[key]
	if w == nil {
		w = &window{start: now}
		l.byKey[key] = w
	}
	w.n++
}
