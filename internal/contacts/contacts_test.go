// SPDX-License-Identifier: GPL-3.0-or-later

package contacts

import (
	"strings"
	"testing"
)

func TestValidateCleansAndRejects(t *testing.T) {
	b := Book{
		Groups: []string{"Work", "work", " Family ", ""},
		Contacts: []Contact{
			{Name: " Bob ", Emails: []string{"Bob@Golub.Test", "bob@golub.test", ""}, Groups: []string{"Work", "Nope"}},
			{Name: "No id yet", Emails: []string{"carol@golub.test"}},
		},
	}
	got, err := Validate(b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Groups, ",") != "Work,Family" {
		t.Errorf("groups %v", got.Groups)
	}
	bob := got.Contacts[0]
	if bob.Name != "Bob" || len(bob.Emails) != 1 || bob.Emails[0] != "Bob@golub.test" || strings.Join(bob.Groups, ",") != "Work" {
		t.Errorf("bob %+v", bob)
	}
	if got.Contacts[1].ID == "" || got.Contacts[1].ID == bob.ID {
		t.Error("missing or duplicate ids get a fresh one")
	}

	for name, bad := range map[string]Book{
		"bad email":  {Contacts: []Contact{{Name: "x", Emails: []string{"not an address"}}}},
		"control":    {Contacts: []Contact{{Name: "a\nb", Emails: []string{"a@b.test"}}}},
		"empty":      {Contacts: []Contact{{}}},
		"long notes": {Contacts: []Contact{{Name: "x", Notes: strings.Repeat("x", maxNotes+1)}}},
	} {
		if _, err := Validate(bad); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestCollect(t *testing.T) {
	b := Book{Contacts: []Contact{{ID: "1", Name: "Bob", Emails: []string{"bob@golub.test"}}}}
	b = Collect(b, []Recipient{{Name: "Bob", Email: "BOB@golub.test"}, {Name: "Carol", Email: "carol@golub.test"}})
	b = Collect(b, []Recipient{{Email: "carol@golub.test"}, {Name: "Dave", Email: "dave@golub.test"}})
	if len(b.Collected) != 2 {
		t.Fatalf("%+v", b.Collected)
	}
	if b.Collected[0].Email != "dave@golub.test" || b.Collected[1].Email != "carol@golub.test" || b.Collected[1].Name != "Carol" || b.Collected[1].Count != 2 {
		t.Errorf("most recent first, names kept, counts add up: %+v", b.Collected)
	}
	for i := range maxCollected + 10 {
		b = Collect(b, []Recipient{{Email: strings.Repeat("x", i%7+1) + string(rune('a'+i%26)) + "@n" + strings.Repeat("y", i/26) + ".test"}})
	}
	if len(b.Collected) > maxCollected {
		t.Errorf("collected grows without bound: %d", len(b.Collected))
	}
}

func TestVCardRoundTrip(t *testing.T) {
	b := Book{
		Groups:   []string{"Work"},
		Contacts: []Contact{{ID: "1", Name: "Željko Đurić", Emails: []string{"zeljko@pošta.rs", "z@golub.test"}, Phones: []string{"+381 60 123"}, Org: "Golub", Notes: "line one\nline two", Groups: []string{"Work"}}},
	}
	var out strings.Builder
	if err := WriteVCards(&out, b.Contacts); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"BEGIN:VCARD", "VERSION:3.0", "FN:Željko Đurić", "EMAIL", "zeljko@pošta.rs", "CATEGORIES:Work", "ORG:Golub", "TEL"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in\n%s", want, s)
		}
	}
	got, err := ParseVCards(strings.NewReader(s))
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %+v", err, got)
	}
	c := got[0]
	if c.Name != "Željko Đurić" || len(c.Emails) != 2 || c.Notes != "line one\nline two" || c.Groups[0] != "Work" || c.Phones[0] != "+381 60 123" {
		t.Errorf("%+v", c)
	}
}

func TestParseVCardsFromOtherClients(t *testing.T) {
	src := "BEGIN:VCARD\r\nVERSION:4.0\r\nN:Smith;Anna;;;\r\nEMAIL;TYPE=work:anna@golub.test\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nFN:No Email\r\nTEL:123\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nEMAIL:only@golub.test\r\nEND:VCARD\r\n"
	got, err := ParseVCards(strings.NewReader(src))
	if err != nil || len(got) != 3 {
		t.Fatalf("%v %+v", err, got)
	}
	if got[0].Name != "Anna Smith" || got[2].Name != "" || got[2].Emails[0] != "only@golub.test" {
		t.Errorf("%+v", got)
	}
}

func TestMerge(t *testing.T) {
	b := Book{Contacts: []Contact{{ID: "1", Name: "Bob", Emails: []string{"bob@golub.test"}}}}
	b, added, updated := Merge(b, []Contact{
		{Name: "Robert", Emails: []string{"BOB@golub.test", "rob@golub.test"}, Groups: []string{"Friends"}},
		{Name: "Anna", Emails: []string{"anna@golub.test"}},
	})
	if added != 1 || updated != 1 || len(b.Contacts) != 2 {
		t.Fatalf("added %d updated %d %+v", added, updated, b.Contacts)
	}
	bob := b.Contacts[0]
	if bob.Name != "Bob" || len(bob.Emails) != 2 || bob.Groups[0] != "Friends" || b.Groups[0] != "Friends" {
		t.Errorf("existing name kept, emails and groups added: %+v %v", bob, b.Groups)
	}
}
