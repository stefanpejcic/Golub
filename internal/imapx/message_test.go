// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"errors"
	"strings"
	"testing"
)

func TestFetchAndMarkSeen(t *testing.T) {
	c := seededClient(t, 3)
	p, err := ListMessages(c, "INBOX", 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	uid := p.Messages[0].UID
	raw, err := FetchRaw(c, "INBOX", uid, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw.Data), "Subject: Message 003") || raw.Seen {
		t.Fatalf("raw = %+v", raw)
	}
	if err := SetFlag(c, "INBOX", uid, FlagSeen, true); err != nil {
		t.Fatal(err)
	}
	raw, _ = FetchRaw(c, "INBOX", uid, 1<<20)
	if !raw.Seen {
		t.Error("not marked seen")
	}
	if err := SetFlag(c, "INBOX", uid, FlagSeen, false); err != nil {
		t.Fatal(err)
	}
	if raw, _ = FetchRaw(c, "INBOX", uid, 1<<20); raw.Seen {
		t.Error("not marked unseen")
	}
}

func TestFetchErrors(t *testing.T) {
	c := seededClient(t, 1)
	if _, err := FetchRaw(c, "INBOX", 9999, 1<<20); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing uid: %v", err)
	}
	p, _ := ListMessages(c, "INBOX", 0, 1)
	if _, err := FetchRaw(c, "INBOX", p.Messages[0].UID, 10); !errors.Is(err, ErrTooLarge) {
		t.Errorf("size cap: %v", err)
	}
}
