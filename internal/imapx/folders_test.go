// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import "testing"

func TestFolderManagement(t *testing.T) {
	c := seededClient(t, 3)
	find := func(name string) *Folder {
		folders, err := ListFolders(c)
		if err != nil {
			t.Fatal(err)
		}
		for i := range folders {
			if folders[i].Name == name {
				return &folders[i]
			}
		}
		return nil
	}

	if err := CreateFolder(c, "Projects"); err != nil {
		t.Fatal(err)
	}
	if f := find("Projects"); f == nil || !f.Subscribed {
		t.Fatalf("created folder: %+v", f)
	}
	if err := SetSubscribed(c, "Projects", false); err != nil {
		t.Fatal(err)
	}
	if f := find("Projects"); f == nil || f.Subscribed {
		t.Fatalf("unsubscribe: %+v", f)
	}
	if err := RenameFolder(c, "Projects", "Clients"); err != nil {
		t.Fatal(err)
	}
	if find("Projects") != nil || find("Clients") == nil {
		t.Fatal("rename failed")
	}
	if err := DeleteFolder(c, "Clients"); err != nil {
		t.Fatal(err)
	}
	if find("Clients") != nil {
		t.Fatal("delete failed")
	}

	if err := EmptyFolder(c, "INBOX"); err != nil {
		t.Fatal(err)
	}
	if f := find("INBOX"); f.Total != 0 {
		t.Fatalf("empty: %+v", f)
	}
}
