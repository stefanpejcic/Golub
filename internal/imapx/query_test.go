// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
)

func TestQueryCriteria(t *testing.T) {
	if c := (Query{}).criteria(); len(c.Or) != 0 || len(c.Body) != 0 || len(c.Flag) != 0 {
		t.Fatalf("empty query must match all: %+v", c)
	}
	day := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	c := Query{Text: "invoice", Body: true, Since: day, Before: day.AddDate(0, 1, 0), Unread: true, Flagged: true, Attachments: true}.criteria()
	if len(c.Or) != 1 || len(c.Or[0][1].Text) != 1 || c.Or[0][1].Text[0] != "invoice" {
		t.Errorf("body search must OR the full text: %+v", c.Or)
	}
	if !c.Since.Equal(day) || !c.Before.Equal(day.AddDate(0, 1, 0)) {
		t.Errorf("dates %v %v", c.Since, c.Before)
	}
	if len(c.NotFlag) != 1 || c.NotFlag[0] != imap.FlagSeen || len(c.Flag) != 1 || c.Flag[0] != imap.FlagFlagged {
		t.Errorf("flags %+v %+v", c.Flag, c.NotFlag)
	}
	if len(c.Header) != 1 || c.Header[0].Key != "Content-Type" {
		t.Errorf("attachments %+v", c.Header)
	}
}

func TestQueryCacheKey(t *testing.T) {
	if k, ok := (Query{Text: "a", Body: true}).cacheKey(); !ok || k == "" {
		t.Error("text and body searches are cacheable")
	}
	a, _ := (Query{Text: "a"}).cacheKey()
	b, _ := (Query{Text: "a", Body: true}).cacheKey()
	if a == b {
		t.Error("body changes the key")
	}
	for _, q := range []Query{{Unread: true}, {Flagged: true}} {
		if _, ok := q.cacheKey(); ok {
			t.Errorf("%+v filters on flags and must not be cached", q)
		}
	}
}

func TestSearchAllFolders(t *testing.T) {
	c := seededClient(t, 0)
	for folder, subjects := range map[string][]string{"INBOX": {"invoice march", "lunch"}, "Work": {"Invoice april"}} {
		for _, s := range subjects {
			if _, err := AppendMessage(c, folder, []byte("Subject: "+s+"\r\n\r\nx\r\n")); err != nil {
				t.Fatal(err)
			}
		}
	}
	res, err := SearchAll(c, Query{Text: "invoice"}, 50)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range res.Messages {
		got[m.Subject] = m.Folder
	}
	if res.Total != 2 || got["invoice march"] != "INBOX" || got["Invoice april"] != "Work" {
		t.Fatalf("%d %v", res.Total, got)
	}
	if res, _ := SearchAll(c, Query{Text: "invoice"}, 1); len(res.Messages) != 1 || res.Total != 2 {
		t.Fatalf("limit: %d of %d", len(res.Messages), res.Total)
	}
}
