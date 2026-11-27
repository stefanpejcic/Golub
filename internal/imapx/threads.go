// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"math"
	"slices"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// A huge thread still costs one fetch per member, so very long ones are cut.
const maxThreadMembers = 500

type member struct {
	uid   uint32
	depth int
}

// flattenThreads turns THREAD trees into per-thread lists in conversation order.
func flattenThreads(trees []imapclient.ThreadData) [][]member {
	out := make([][]member, 0, len(trees))
	for _, t := range trees {
		var list []member
		walkThread(t, 0, &list)
		if len(list) > 0 {
			out = append(out, list[:min(len(list), maxThreadMembers)])
		}
	}
	return out
}

func walkThread(t imapclient.ThreadData, depth int, list *[]member) {
	for _, uid := range t.Chain {
		*list = append(*list, member{uid: uid, depth: depth})
		depth++
	}
	for _, sub := range t.SubThreads {
		walkThread(sub, depth, list)
	}
}

// orderThreads sorts threads by their best-ranked member, so with newest-first a thread moves up on every reply.
func orderThreads(threads [][]member, rank map[uint32]int) [][]member {
	best := func(t []member) int {
		r := math.MaxInt
		for _, m := range t {
			if v, ok := rank[m.uid]; ok && v < r {
				r = v
			}
		}
		return r
	}
	type ranked struct {
		t []member
		r int
	}
	list := make([]ranked, len(threads))
	for i, t := range threads {
		list[i] = ranked{t, best(t)}
	}
	slices.SortStableFunc(list, func(a, b ranked) int { return a.r - b.r })
	out := make([][]member, len(list))
	for i, r := range list {
		out[i] = r.t
	}
	return out
}

// CanThread reports whether the server can group messages into conversations.
func CanThread(c *imapclient.Client) bool {
	return c.Caps().Has(imap.CapSort) && slices.Contains(c.Caps().ThreadAlgorithms(), imap.ThreadReferences)
}

// ListThreads pages through conversations; each page holds limit threads with all their messages.
func ListThreads(c *imapclient.Client, folder string, offset, limit int, query Query, sortBy Sort) (Page, error) {
	if !ValidSort(sortBy.Key) {
		sortBy = Sort{Key: "date", Desc: true}
	}
	sel, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return Page{}, err
	}
	page := Page{Messages: []Summary{}}
	if sel.NumMessages == 0 {
		return page, nil
	}
	st, cacheable := stateOf(sel)

	sorted, err := sortedUIDs(c, folder, query, sortBy, st, cacheable)
	if err != nil {
		return Page{}, err
	}
	threads, err := threadsOf(c, folder, query, st, cacheable)
	if err != nil {
		return Page{}, err
	}
	rank := make(map[uint32]int, len(sorted))
	for i, u := range sorted {
		rank[u] = i
	}
	ordered := orderThreads(threads, rank)
	page.Total = len(ordered)
	if offset >= len(ordered) || limit <= 0 {
		return page, nil
	}
	ordered = ordered[offset:min(offset+limit, len(ordered))]

	var set imap.UIDSet
	for _, t := range ordered {
		for _, m := range t {
			set.AddNum(imap.UID(m.uid))
		}
	}
	msgs, err := c.Fetch(set, summaryFetch).Collect()
	if err != nil {
		return Page{}, err
	}
	byUID := make(map[uint32]Summary, len(msgs))
	for _, m := range msgs {
		s := summarize(m)
		byUID[s.UID] = s
	}
	for _, t := range ordered {
		start := len(page.Messages)
		unread := 0
		for _, m := range t {
			s, ok := byUID[m.uid]
			if !ok {
				continue
			}
			s.Depth = m.depth
			s.Root = t[0].uid
			if !s.Seen {
				unread++
			}
			page.Messages = append(page.Messages, s)
		}
		if len(page.Messages) > start {
			page.Messages[start].ThreadSize = len(page.Messages) - start
			page.Messages[start].ThreadUnread = unread
		}
	}
	return page, nil
}

func threadsOf(c *imapclient.Client, folder string, query Query, st sortState, cacheable bool) ([][]member, error) {
	qk, ok := query.cacheKey()
	cacheable = cacheable && ok
	key := sortKey{folder: folder, query: qk, sort: "thread"}
	if e, ok := sorted.getEntry(c, key, st); cacheable && ok {
		return e.threads, nil
	}
	sortRuns.Add(1)
	trees, err := c.UIDThread(&imapclient.ThreadOptions{Algorithm: imap.ThreadReferences, SearchCriteria: query.criteria()}).Wait()
	if err != nil {
		return nil, err
	}
	threads := flattenThreads(trees)
	if cacheable {
		sorted.putEntry(c, sortEntry{key: key, state: st, threads: threads})
	}
	return threads, nil
}
