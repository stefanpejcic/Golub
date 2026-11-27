// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"runtime"
	"sync"
	"sync/atomic"
	"weak"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Each entry can hold a whole 50k folder, so only the most recent few views are kept.
const sortCacheEntries = 3

// sortRuns counts real UID SORT commands, for tests.
var sortRuns atomic.Int64

// Flag changes do not touch sortState, so searches that filter on flags must not be cached.
type sortKey struct {
	folder, query, sort string
	desc                bool
}

// sortState is what must stay the same for a cached order to still be right:
// any arrival bumps UIDNEXT and any expunge lowers the count.
type sortState struct {
	validity, next, count uint32
}

type sortEntry struct {
	key   sortKey
	state sortState
	uids  []uint32
}

type sortCache struct {
	mu      sync.Mutex
	entries map[weak.Pointer[imapclient.Client]][]sortEntry
}

var sorted = sortCache{entries: map[weak.Pointer[imapclient.Client]][]sortEntry{}}

func stateOf(sel *imap.SelectData) (sortState, bool) {
	s := sortState{validity: sel.UIDValidity, next: uint32(sel.UIDNext), count: sel.NumMessages}
	return s, s.validity != 0 && s.next != 0
}

func (sc *sortCache) get(c *imapclient.Client, k sortKey, st sortState) ([]uint32, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for _, e := range sc.entries[weak.Make(c)] {
		if e.key == k && e.state == st {
			return e.uids, true
		}
	}
	return nil, false
}

func (sc *sortCache) put(c *imapclient.Client, k sortKey, st sortState, uids []uint32) {
	wp := weak.Make(c)
	sc.mu.Lock()
	defer sc.mu.Unlock()
	list, known := sc.entries[wp]
	if !known {
		runtime.AddCleanup(c, func(p weak.Pointer[imapclient.Client]) {
			sc.mu.Lock()
			delete(sc.entries, p)
			sc.mu.Unlock()
		}, wp)
	}
	kept := []sortEntry{{key: k, state: st, uids: uids}}
	for _, e := range list {
		if e.key != k && len(kept) < sortCacheEntries {
			kept = append(kept, e)
		}
	}
	sc.entries[wp] = kept
}
