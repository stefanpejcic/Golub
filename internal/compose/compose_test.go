// SPDX-License-Identifier: GPL-3.0-or-later

package compose

import (
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stefanpejcic/golub/internal/mime"
)

func TestBuildRoundTrip(t *testing.T) {
	d := Draft{
		From:       mail.Address{Name: "Alice Ž", Address: "alice@golub.test"},
		To:         []mail.Address{{Name: "Bob", Address: "bob@golub.test"}},
		Cc:         []mail.Address{{Address: "carol@golub.test"}},
		Bcc:        []mail.Address{{Address: "secret@golub.test"}},
		ReplyTo:    []mail.Address{{Address: "replies@golub.test"}},
		Subject:    "Pozdrav šefče",
		Text:       "Zdravo,\nkako si?",
		HTML:       "<p>Zdravo, <b>kako si?</b></p>",
		InReplyTo:  "<orig@golub.test>",
		References: []string{"<root@golub.test>", "<orig@golub.test>"},
		Attachments: []Attachment{
			{Filename: "članarina.txt", ContentType: "text/plain", Data: []byte("hello")},
		},
		Date: time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC),
	}
	raw, id, err := Build(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, "@golub.test>") {
		t.Errorf("message id = %q", id)
	}
	s := string(raw)
	for _, bad := range []string{"secret@golub.test", "Bcc:"} {
		if strings.Contains(s, bad) {
			t.Errorf("bcc leaked into headers")
		}
	}
	if strings.Contains(s, "User-Agent") || strings.Contains(s, "X-Mailer") {
		t.Error("outgoing mail must not reveal the client")
	}
	for _, want := range []string{"In-Reply-To: <orig@golub.test>", "References: <root@golub.test> <orig@golub.test>", "Message-Id: " + id} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in\n%s", want, s)
		}
	}

	m, err := mime.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "Pozdrav šefče" || m.From[0].Name != "Alice Ž" || len(m.To) != 1 || len(m.Cc) != 1 || m.ReplyTo[0].Address != "replies@golub.test" {
		t.Fatalf("%+v", m)
	}
	if !strings.Contains(m.Text, "kako si?") || !strings.Contains(m.HTML, "<b>kako si?</b>") {
		t.Errorf("text %q html %q", m.Text, m.HTML)
	}
	if len(m.Attachments) != 1 || m.Attachments[0].Filename != "članarina.txt" {
		t.Errorf("attachments %+v", m.Attachments)
	}
}

func TestBuildPlainOnly(t *testing.T) {
	raw, _, err := Build(Draft{From: mail.Address{Address: "a@golub.test"}, To: []mail.Address{{Address: "b@golub.test"}}, Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "text/html") {
		t.Error("plain message must not carry an HTML part")
	}
}

func TestRecipients(t *testing.T) {
	d := Draft{
		To:  []mail.Address{{Address: "b@golub.test"}, {Address: "B@golub.test"}},
		Cc:  []mail.Address{{Address: "c@golub.test"}},
		Bcc: []mail.Address{{Address: "d@golub.test"}, {Address: "c@golub.test"}},
	}
	got := Recipients(d)
	if strings.Join(got, ",") != "b@golub.test,c@golub.test,d@golub.test" {
		t.Fatalf("%v", got)
	}
}

func TestBuildRejectsHeaderInjection(t *testing.T) {
	_, _, err := Build(Draft{From: mail.Address{Address: "a@golub.test"}, To: []mail.Address{{Address: "b@golub.test"}}, Subject: "hi\r\nBcc: evil@x.y"})
	if err == nil {
		t.Fatal("expected error")
	}
	_, _, err = Build(Draft{From: mail.Address{Address: "a@golub.test"}, To: []mail.Address{{Address: "b@golub.test"}}, InReplyTo: "<x>\r\nBcc: e@x.y"})
	if err == nil {
		t.Fatal("expected error for In-Reply-To")
	}
}

func TestReplySubject(t *testing.T) {
	cases := map[string]string{"Hello": "Re: Hello", "Re: Hello": "Re: Hello", "RE: Hello": "RE: Hello", "": "Re: "}
	for in, want := range cases {
		if got := ReplySubject(in); got != want {
			t.Errorf("%q -> %q", in, got)
		}
	}
	if got := ForwardSubject("Fwd: Hi"); got != "Fwd: Hi" {
		t.Error(got)
	}
	if got := ForwardSubject("Hi"); got != "Fwd: Hi" {
		t.Error(got)
	}
}

func TestBuildWithoutRecipients(t *testing.T) {
	raw, _, err := Build(Draft{From: mail.Address{Address: "a@golub.test"}, Subject: "draft", Text: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "To:") || strings.Contains(string(raw), "Bcc:") {
		t.Fatalf("%s", raw)
	}
}

func TestPriorityAndReceiptHeaders(t *testing.T) {
	d := Draft{From: mail.Address{Address: "a@golub.test"}, To: []mail.Address{{Address: "b@golub.test"}}, Subject: "x", Text: "y", Priority: 1, RequestReceipt: true}
	raw, _, err := Build(d)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{"X-Priority: 1 (Highest)", "Importance: high", "Disposition-Notification-To: <a@golub.test>"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	d.Priority, d.RequestReceipt = 3, false
	raw, _, _ = Build(d)
	if strings.Contains(string(raw), "X-Priority") || strings.Contains(string(raw), "Disposition-Notification-To") {
		t.Error("normal priority without receipt adds no headers")
	}
}

func TestBuildMDN(t *testing.T) {
	raw, err := BuildMDN(MDN{
		From:              mail.Address{Name: "Alice", Address: "alice@golub.test"},
		To:                mail.Address{Address: "bob@golub.test"},
		OriginalSubject:   "Urgent with receipt",
		OriginalMessageID: "<urgent-receipt@golub.test>",
		Text:              "Your message was read.",
		Date:              time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{
		"Content-Type: multipart/report; report-type=disposition-notification",
		"Content-Type: message/disposition-notification",
		"Final-Recipient: rfc822; alice@golub.test",
		"Original-Message-ID: <urgent-receipt@golub.test>",
		"Disposition: manual-action/MDN-sent-manually; displayed",
		"Subject: Read: Urgent with receipt",
		"To: <bob@golub.test>",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in\n%s", want, s)
		}
	}
	if m, err := mime.Parse(raw); err != nil || !strings.Contains(m.Text, "Your message was read.") {
		t.Fatalf("parse: %v %+v", err, m)
	}
}
