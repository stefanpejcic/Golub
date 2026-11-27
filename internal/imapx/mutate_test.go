// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"testing"
)

func TestSetFlags(t *testing.T) {
	c := seededClient(t, 4)
	p, _ := ListMessages(c, "INBOX", 0, 4)
	var all []uint32
	for _, m := range p.Messages {
		all = append(all, m.UID)
	}
	if err := SetFlags(c, "INBOX", all, FlagSeen, true); err != nil {
		t.Fatal(err)
	}
	if err := SetFlags(c, "INBOX", all[:1], FlagFlagged, true); err != nil {
		t.Fatal(err)
	}
	p, _ = ListMessages(c, "INBOX", 0, 4)
	for i, m := range p.Messages {
		if !m.Seen || m.Flagged != (i == 0) {
			t.Errorf("%d: %+v", i, m)
		}
	}
}

func TestMoveAndDelete(t *testing.T) {
	c := seededClient(t, 5)
	p, _ := ListMessages(c, "INBOX", 0, 5)
	first, second := p.Messages[0].UID, p.Messages[1].UID

	if err := Move(c, "INBOX", []uint32{first, second}, "Work"); err != nil {
		t.Fatal(err)
	}
	inbox, _ := ListMessages(c, "INBOX", 0, 10)
	work, _ := ListMessages(c, "Work", 0, 10)
	if inbox.Total != 3 || work.Total != 2 {
		t.Fatalf("inbox %d work %d", inbox.Total, work.Total)
	}

	if err := Expunge(c, "Work", []uint32{work.Messages[0].UID}); err != nil {
		t.Fatal(err)
	}
	work, _ = ListMessages(c, "Work", 0, 10)
	if work.Total != 1 {
		t.Fatalf("work after expunge = %d", work.Total)
	}

	if err := Move(c, "INBOX", []uint32{inbox.Messages[0].UID}, "Missing"); err == nil {
		t.Error("move to missing folder should fail")
	}
}

func TestSpecialFolder(t *testing.T) {
	c := seededClient(t, 0)
	folders, _ := ListFolders(c)
	if name := SpecialFolder(folders, "trash", "Trash"); name != "Trash" {
		t.Errorf("by name fallback = %q", name)
	}
	folders = append(folders, Folder{Name: "Deleted Items", Special: "trash", Selectable: true})
	if name := SpecialFolder(folders, "trash", "Trash"); name != "Deleted Items" {
		t.Errorf("by special use = %q", name)
	}
	if name := SpecialFolder(nil, "trash", "Trash"); name != "" {
		t.Errorf("none = %q", name)
	}
}
