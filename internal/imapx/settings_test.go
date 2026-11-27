// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	c := seededClient(t, 0)
	data, err := LoadSettings(c)
	if err != nil || data != nil {
		t.Fatalf("fresh account: %q %v", data, err)
	}
	if err := SaveSettings(c, []byte(`{"pageSize":25}`)); err != nil {
		t.Fatal(err)
	}
	if err := SaveSettings(c, []byte(`{"pageSize":100}`)); err != nil {
		t.Fatal(err)
	}
	data, err = LoadSettings(c)
	if err != nil || string(data) != `{"pageSize":100}` {
		t.Fatalf("got %q %v", data, err)
	}
	p, err := ListMessages(c, SettingsFolder, 0, 10, Query{})
	if err != nil || p.Total != 1 {
		t.Fatalf("old settings not cleaned up: %d %v", p.Total, err)
	}
	folders, _ := ListFolders(c)
	for _, f := range folders {
		if f.Name == SettingsFolder {
			t.Fatal("settings folder must not be listed")
		}
	}
}
