// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		l    imap.ListData
		want Folder
	}{
		{imap.ListData{Mailbox: "inbox", Delim: '/'}, Folder{Name: "INBOX", Delim: "/", Special: "inbox", Selectable: true, Subscribed: true}},
		{imap.ListData{Mailbox: "Sent Items", Attrs: []imap.MailboxAttr{imap.MailboxAttrSent, imap.MailboxAttrSubscribed}}, Folder{Name: "Sent Items", Special: "sent", Selectable: true, Subscribed: true}},
		{imap.ListData{Mailbox: "Junk", Attrs: []imap.MailboxAttr{imap.MailboxAttrHasNoChildren, imap.MailboxAttrJunk}}, Folder{Name: "Junk", Special: "junk", Selectable: true}},
		{imap.ListData{Mailbox: "Archive/2025", Delim: '/', Attrs: []imap.MailboxAttr{imap.MailboxAttrNoSelect}}, Folder{Name: "Archive/2025", Delim: "/", Selectable: false}},
	}
	for _, tc := range cases {
		if got := classify(&tc.l); got != tc.want {
			t.Errorf("%s: got %+v want %+v", tc.l.Mailbox, got, tc.want)
		}
	}
}

// seededClient logs into a memory server holding INBOX, Sent, Trash and Work.
func seededClient(t *testing.T, inbox int) *imapclient.Client {
	t.Helper()
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("alice@golub.test", "secret")
	for _, mb := range []struct {
		name string
		use  imap.MailboxAttr
	}{{"Sent", imap.MailboxAttrSent}, {"Trash", imap.MailboxAttrTrash}, {"Work", ""}} {
		var opts *imap.CreateOptions
		if mb.use != "" {
			opts = &imap.CreateOptions{SpecialUse: []imap.MailboxAttr{mb.use}}
		}
		if err := u.Create(mb.name, opts); err != nil {
			t.Fatal(err)
		}
	}
	if err := u.Create("INBOX", nil); err != nil && !strings.Contains(err.Error(), "exist") {
		t.Fatal(err)
	}
	mem.AddUser(u)
	ep := listen(t, mem)
	c, err := (&Dialer{Endpoint: ep}).Login(context.Background(), "alice@golub.test", "secret")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= inbox; i++ {
		msg := fmt.Sprintf("From: =?utf-8?q?Bob_=C5=A0?= <bob@golub.test>\r\nSubject: Message %03d\r\nDate: %s\r\n\r\nbody\r\n",
			i, base.Add(time.Duration(i)*time.Hour).Format(time.RFC1123Z))
		var flags []imap.Flag
		if i%2 == 0 {
			flags = []imap.Flag{imap.FlagSeen}
		}
		cmd := c.Append("INBOX", int64(len(msg)), &imap.AppendOptions{Flags: flags, Time: base.Add(time.Duration(i) * time.Hour)})
		if _, err := cmd.Write([]byte(msg)); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestListFolders(t *testing.T) {
	c := seededClient(t, 5)
	folders, err := ListFolders(c)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range folders {
		got = append(got, fmt.Sprintf("%s:%s:%d/%d", f.Name, f.Special, f.Unread, f.Total))
	}
	want := "INBOX:inbox:3/5 Sent::0/0 Trash::0/0 Work::0/0"
	if strings.Join(got, " ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " "), want)
	}
}

func TestListMessagesPaging(t *testing.T) {
	c := seededClient(t, 7)
	p, err := ListMessages(c, "INBOX", 0, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 7 || len(p.Messages) != 3 {
		t.Fatalf("total=%d len=%d", p.Total, len(p.Messages))
	}
	if p.Messages[0].Subject != "Message 007" || p.Messages[2].Subject != "Message 005" {
		t.Errorf("order: %q .. %q", p.Messages[0].Subject, p.Messages[2].Subject)
	}
	m := p.Messages[0]
	if m.Seen || len(m.From) != 1 || m.From[0].Name != "Bob Š" || m.From[0].Address != "bob@golub.test" || m.UID == 0 {
		t.Errorf("summary: %+v", m)
	}
	if !p.Messages[1].Seen {
		t.Error("message 006 should be seen")
	}

	last, err := ListMessages(c, "INBOX", 6, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Messages) != 1 || last.Messages[0].Subject != "Message 001" {
		t.Errorf("last page: %+v", last.Messages)
	}

	past, err := ListMessages(c, "INBOX", 50, 3, "")
	if err != nil || len(past.Messages) != 0 || past.Total != 7 {
		t.Errorf("past end: %+v %v", past, err)
	}
}

func TestListMessagesEmptyAndMissing(t *testing.T) {
	c := seededClient(t, 0)
	p, err := ListMessages(c, "Work", 0, 50, "")
	if err != nil || p.Total != 0 || len(p.Messages) != 0 {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := ListMessages(c, "Nope", 0, 50, ""); err == nil {
		t.Fatal("expected error for missing folder")
	}
}

func TestDecodeLegacyCharsets(t *testing.T) {
	d := wordDecoder()
	cases := map[string]string{
		"=?windows-1250?Q?Pozdrav_=9Aef=E8e?=": "Pozdrav šefče",
		"=?iso-8859-2?B?yGVzdGl0a2E=?=":        "Čestitka",
		"=?koi8-r?B?8NLJ18XU?=":                "Привет",
	}
	for in, want := range cases {
		got, err := d.DecodeHeader(in)
		if err != nil || got != want {
			t.Errorf("%s: got %q err %v", in, got, err)
		}
	}
}

func TestListMessagesSearch(t *testing.T) {
	c := seededClient(t, 12)
	p, err := ListMessages(c, "INBOX", 0, 50, "Message 01")
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 3 || len(p.Messages) != 3 || p.Messages[0].Subject != "Message 012" {
		t.Fatalf("%+v", p)
	}
	p, _ = ListMessages(c, "INBOX", 0, 2, "Message 01")
	if p.Total != 3 || len(p.Messages) != 2 {
		t.Fatalf("paged search: %+v", p)
	}
	p, _ = ListMessages(c, "INBOX", 0, 50, "bob@golub.test")
	if p.Total != 12 {
		t.Fatalf("search by sender: %d", p.Total)
	}
	p, _ = ListMessages(c, "INBOX", 0, 50, "nothing matches this")
	if p.Total != 0 || len(p.Messages) != 0 {
		t.Fatalf("no match: %+v", p)
	}
}

func TestListMessagesAscending(t *testing.T) {
	c := seededClient(t, 5)
	p, err := ListMessages(c, "INBOX", 0, 2, "", Sort{Key: "arrival", Desc: false})
	if err != nil {
		t.Fatal(err)
	}
	if p.Messages[0].Subject != "Message 001" || p.Messages[1].Subject != "Message 002" {
		t.Fatalf("%+v", p.Messages)
	}
	p, _ = ListMessages(c, "INBOX", 4, 2, "", Sort{Key: "arrival", Desc: false})
	if len(p.Messages) != 1 || p.Messages[0].Subject != "Message 005" {
		t.Fatalf("last page %+v", p.Messages)
	}
}

func TestFolderSizes(t *testing.T) {
	c := seededClient(t, 3)
	folders, err := ListFolders(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range folders {
		if f.Name == "INBOX" && (f.Size == nil || *f.Size <= 0) {
			t.Fatalf("INBOX size missing: %+v", f)
		}
	}
}

func TestPriorityInList(t *testing.T) {
	c := seededClient(t, 0)
	for _, raw := range []string{"Subject: hi\r\nX-Priority: 1\r\n\r\nx\r\n", "Subject: lo\r\nImportance: low\r\n\r\nx\r\n", "Subject: plain\r\n\r\nx\r\n"} {
		if _, err := AppendMessage(c, "Work", []byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	p, err := ListMessages(c, "Work", 0, 10, "", Sort{Key: "arrival"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, m := range p.Messages {
		got[m.Subject] = m.Priority
	}
	if got["hi"] != 1 || got["lo"] != 5 || got["plain"] != 3 {
		t.Fatalf("%v", got)
	}
}

func TestMarkAllRead(t *testing.T) {
	c := seededClient(t, 4)
	if err := MarkAllRead(c, "INBOX"); err != nil {
		t.Fatal(err)
	}
	folders, _ := ListFolders(c)
	if folders[0].Unread != 0 {
		t.Fatalf("%+v", folders[0])
	}
}

func TestSortCacheReusedUntilFolderChanges(t *testing.T) {
	c := seededClient(t, 30)
	if !c.Caps().Has(imap.CapSort) {
		t.Skip("test server has no SORT")
	}
	before := sortRuns.Load()
	p1, err := ListMessages(c, "INBOX", 0, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := ListMessages(c, "INBOX", 10, 10, "")
	if got := sortRuns.Load() - before; got != 1 {
		t.Fatalf("second page ran SORT again: %d runs", got)
	}
	if p1.Messages[9].UID == p2.Messages[0].UID || p2.Total != 30 {
		t.Fatalf("pages overlap or wrong total: %d", p2.Total)
	}

	if _, err := AppendMessage(c, "INBOX", []byte("Subject: new\r\n\r\nx\r\n")); err != nil {
		t.Fatal(err)
	}
	p3, _ := ListMessages(c, "INBOX", 0, 10, "")
	if sortRuns.Load()-before != 2 || p3.Total != 31 {
		t.Fatalf("new mail must invalidate the cache: total %d", p3.Total)
	}

	if _, err := ListMessages(c, "INBOX", 0, 10, "", Sort{Key: "size"}); err != nil {
		t.Fatal(err)
	}
	if sortRuns.Load()-before != 3 {
		t.Fatal("a different sort needs its own SORT")
	}
}
