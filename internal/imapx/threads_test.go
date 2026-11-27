// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"fmt"
	"testing"

	"github.com/emersion/go-imap/v2/imapclient"
)

func show(threads [][]member) string {
	var s string
	for i, t := range threads {
		if i > 0 {
			s += " | "
		}
		for j, m := range t {
			if j > 0 {
				s += " "
			}
			s += fmt.Sprintf("%d:%d", m.uid, m.depth)
		}
	}
	return s
}

func TestFlattenThreads(t *testing.T) {
	// (2)(3 6 (4 23)(44 7 96)) from RFC 5256, plus a thread whose root is missing.
	trees := []imapclient.ThreadData{
		{Chain: []uint32{2}},
		{Chain: []uint32{3, 6}, SubThreads: []imapclient.ThreadData{{Chain: []uint32{4, 23}}, {Chain: []uint32{44, 7, 96}}}},
		{SubThreads: []imapclient.ThreadData{{Chain: []uint32{10}}, {Chain: []uint32{11}}}},
	}
	got := flattenThreads(trees)
	want := "2:0 | 3:0 6:1 4:2 23:3 44:2 7:3 96:4 | 10:0 11:0"
	if show(got) != want {
		t.Fatalf("\n got %s\nwant %s", show(got), want)
	}
}

func TestOrderThreadsByNewestMember(t *testing.T) {
	threads := [][]member{{{uid: 1}}, {{uid: 2}, {uid: 5, depth: 1}}, {{uid: 3}}}
	// Sorted newest first: 5, 3, 1, 2.
	rank := map[uint32]int{5: 0, 3: 1, 1: 2, 2: 3}
	got := orderThreads(threads, rank)
	if show(got) != "2:0 5:1 | 3:0 | 1:0" {
		t.Fatalf("got %s", show(got))
	}
}
