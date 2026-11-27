// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"runtime"
	"slices"
	"testing"
	"time"
	"weak"

	"github.com/emersion/go-imap/v2/imapclient"
)

func TestSortCacheHitsAndInvalidation(t *testing.T) {
	sc := sortCache{entries: map[weak.Pointer[imapclient.Client]][]sortEntry{}}
	c := new(imapclient.Client)
	k := sortKey{folder: "INBOX", sort: "date", desc: true}
	st := sortState{validity: 1, next: 10, count: 9}
	sc.put(c, k, st, []uint32{9, 8, 7})

	if got, ok := sc.get(c, k, st); !ok || !slices.Equal(got, []uint32{9, 8, 7}) {
		t.Fatalf("miss: %v", got)
	}
	for name, other := range map[string]sortState{
		"new mail":    {validity: 1, next: 11, count: 10},
		"expunge":     {validity: 1, next: 10, count: 8},
		"uidvalidity": {validity: 2, next: 10, count: 9},
	} {
		if _, ok := sc.get(c, k, other); ok {
			t.Errorf("%s must invalidate", name)
		}
	}
	if _, ok := sc.get(c, sortKey{folder: "INBOX", sort: "size", desc: true}, st); ok {
		t.Error("other sort must miss")
	}
	if _, ok := sc.get(new(imapclient.Client), k, st); ok {
		t.Error("other connection must miss")
	}
	runtime.KeepAlive(c)
}

func TestSortCacheEvictsOldest(t *testing.T) {
	sc := sortCache{entries: map[weak.Pointer[imapclient.Client]][]sortEntry{}}
	c := new(imapclient.Client)
	st := sortState{validity: 1, next: 2, count: 1}
	for _, f := range []string{"a", "b", "c", "d", "e"} {
		sc.put(c, sortKey{folder: f}, st, []uint32{1})
	}
	if _, ok := sc.get(c, sortKey{folder: "a"}, st); ok {
		t.Error("oldest entry should be gone")
	}
	if _, ok := sc.get(c, sortKey{folder: "e"}, st); !ok {
		t.Error("newest entry missing")
	}
	runtime.KeepAlive(c)
}

func TestSortCacheFreedWithConnection(t *testing.T) {
	sc := sortCache{entries: map[weak.Pointer[imapclient.Client]][]sortEntry{}}
	func() {
		c := new(imapclient.Client)
		sc.put(c, sortKey{folder: "INBOX"}, sortState{validity: 1, next: 2, count: 1}, []uint32{1})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		sc.mu.Lock()
		n := len(sc.entries)
		sc.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("entry outlived its connection")
}
