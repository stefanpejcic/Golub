// SPDX-License-Identifier: GPL-3.0-or-later

package mime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func load(t *testing.T, name string) *Message {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mail", name))
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(raw)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return m
}

func TestAllFixturesParse(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "mail", "*.eml"))
	if len(files) == 0 {
		t.Fatal("no fixtures")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			m := load(t, filepath.Base(f))
			if m.Subject == "" {
				t.Error("empty subject")
			}
			if m.Text == "" && m.HTML == "" {
				t.Error("no body")
			}
		})
	}
}

func TestHeaders(t *testing.T) {
	m := load(t, "01-plain.eml")
	if m.Subject != "Plain text" || len(m.From) != 1 || m.From[0].Name != "Bob Sender" || m.From[0].Address != "bob@golub.test" {
		t.Fatalf("%+v", m)
	}
	if len(m.To) != 1 || m.To[0].Address != "alice@golub.test" || m.Date.IsZero() || m.MessageID != "<plain-text@golub.test>" {
		t.Errorf("%+v", m)
	}
	if !strings.Contains(m.Text, "čćžšđ") || m.HTML != "" {
		t.Errorf("text = %q html = %q", m.Text, m.HTML)
	}
}

func TestLegacyCharsets(t *testing.T) {
	cases := map[string][2]string{
		"06-windows-1250.eml":    {"Pozdrav šefče", "Šta ima, čika Đorđe? Žuta ćurka."},
		"07-iso-8859-2.eml":      {"Čestitka", "Srećan rođendan, Željko!"},
		"08-koi8-r.eml":          {"Привет", "Привет, как дела?"},
		"09-missing-charset.eml": {"Missing charset", "Body without charset"},
	}
	for f, want := range cases {
		m := load(t, f)
		if m.Subject != want[0] || !strings.Contains(m.Text, want[1]) {
			t.Errorf("%s: subject %q text %q", f, m.Subject, m.Text)
		}
	}
}

func TestAlternativePrefersBoth(t *testing.T) {
	m := load(t, "03-alternative.eml")
	if !strings.Contains(m.HTML, "<i>part</i>") || !strings.Contains(m.Text, "Plain part.") {
		t.Errorf("html %q text %q", m.HTML, m.Text)
	}
}

func TestAttachments(t *testing.T) {
	m := load(t, "05-attachment.eml")
	if len(m.Attachments) != 2 {
		t.Fatalf("attachments = %+v", m.Attachments)
	}
	pdf, txt := m.Attachments[0], m.Attachments[1]
	if pdf.Filename != "report.pdf" || pdf.ContentType != "application/pdf" || pdf.Size == 0 || pdf.Inline {
		t.Errorf("pdf = %+v", pdf)
	}
	if txt.Filename != "članarina šema.txt" {
		t.Errorf("rfc2231 filename = %q", txt.Filename)
	}
	p := m.Part(pdf.ID)
	if p == nil || !strings.HasPrefix(string(p.Content), "%PDF") {
		t.Fatal("part lookup failed")
	}
	if m.Part("9.9.9") != nil {
		t.Error("unknown part found")
	}
}

func TestInlineCID(t *testing.T) {
	m := load(t, "04-inline-cid.eml")
	if len(m.Attachments) != 0 {
		t.Errorf("inline image listed as attachment: %+v", m.Attachments)
	}
	p := m.CID("logo@golub.test")
	if p == nil || p.ContentType != "image/png" || len(p.Content) == 0 {
		t.Fatalf("cid part = %+v", p)
	}
	if m.CID("<logo@golub.test>") == nil {
		t.Error("angle brackets not tolerated")
	}
}

func TestNested(t *testing.T) {
	m := load(t, "11-nested.eml")
	if !strings.Contains(m.HTML, "Nested <b>html</b>") || m.CID("dot@golub.test") == nil {
		t.Errorf("html %q", m.HTML)
	}
	found := false
	for _, a := range m.Attachments {
		if a.ContentType == "message/rfc822" {
			found = true
		}
	}
	if !found {
		t.Errorf("forwarded message not offered: %+v", m.Attachments)
	}
}

func TestTNEFIsDownloadable(t *testing.T) {
	m := load(t, "30-tnef.eml")
	if len(m.Attachments) != 1 || m.Attachments[0].Filename != "winmail.dat" {
		t.Errorf("%+v", m.Attachments)
	}
}

func TestListUnsubscribe(t *testing.T) {
	m := load(t, "40-newsletter.eml")
	u := m.Unsubscribe
	if u.Mailto != "mailto:unsubscribe@lists.golub.test?subject=unsubscribe%20alice" || u.URL != "https://lists.golub.test/u/abc123" || !u.OneClick {
		t.Fatalf("%+v", u)
	}
	if m.Priority != 5 {
		t.Errorf("priority %d", m.Priority)
	}
	if load(t, "01-plain.eml").Unsubscribe.Mailto != "" {
		t.Error("plain message has no unsubscribe")
	}
}

func TestPriorityAndReceipt(t *testing.T) {
	m := load(t, "41-urgent-receipt.eml")
	if m.Priority != 1 || m.ReceiptTo != "bob@golub.test" {
		t.Fatalf("priority %d receipt %q", m.Priority, m.ReceiptTo)
	}
	if p := load(t, "01-plain.eml"); p.Priority != 3 || p.ReceiptTo != "" {
		t.Errorf("defaults: %d %q", p.Priority, p.ReceiptTo)
	}
}

func TestParsePriority(t *testing.T) {
	cases := map[[2]string]int{
		{"1 (Highest)", ""}: 1, {"2", ""}: 1, {"3", ""}: 3, {"4 (Low)", ""}: 5, {"5", ""}: 5,
		{"", "high"}: 1, {"", "Low"}: 5, {"", ""}: 3, {"bogus", ""}: 3,
	}
	for in, want := range cases {
		if got := ParsePriority(in[0], in[1]); got != want {
			t.Errorf("%v -> %d", in, got)
		}
	}
}

func TestPGPDetection(t *testing.T) {
	head := "From: a@golub.test\r\nTo: b@golub.test\r\nSubject: s\r\nMIME-Version: 1.0\r\n"
	cases := map[string]string{
		"encrypted":        head + "Content-Type: multipart/encrypted; protocol=\"application/pgp-encrypted\"; boundary=\"b\"\r\n\r\n--b\r\nContent-Type: application/pgp-encrypted\r\n\r\nVersion: 1\r\n--b\r\nContent-Type: application/octet-stream\r\n\r\n-----BEGIN PGP MESSAGE-----\r\n\r\nwV4D\r\n-----END PGP MESSAGE-----\r\n--b--\r\n",
		"signed":           head + "Content-Type: multipart/signed; micalg=pgp-sha256; protocol=\"application/pgp-signature\"; boundary=\"b\"\r\n\r\n--b\r\nContent-Type: text/plain\r\n\r\nhello\r\n--b\r\nContent-Type: application/pgp-signature\r\n\r\n-----BEGIN PGP SIGNATURE-----\r\n\r\nwnUE\r\n-----END PGP SIGNATURE-----\r\n--b--\r\n",
		"inline-encrypted": head + "Content-Type: text/plain\r\n\r\n-----BEGIN PGP MESSAGE-----\r\n\r\nwV4D\r\n-----END PGP MESSAGE-----\r\n",
		"inline-signed":    head + "Content-Type: text/plain\r\n\r\n-----BEGIN PGP SIGNED MESSAGE-----\r\nHash: SHA256\r\n\r\nhello\r\n-----BEGIN PGP SIGNATURE-----\r\n\r\nwnUE\r\n-----END PGP SIGNATURE-----\r\n",
		"":                 head + "Content-Type: text/plain\r\n\r\nWe talked about -----BEGIN PGP MESSAGE----- blocks today.\r\n",
	}
	for want, raw := range cases {
		m, err := Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if m.PGP != want {
			t.Errorf("want %q, got %q", want, m.PGP)
		}
	}
	if m, _ := Parse([]byte(cases["signed"])); !strings.Contains(m.Text, "hello") {
		t.Error("a signed message still reads normally while the browser checks the signature")
	}
}
