// SPDX-License-Identifier: GPL-3.0-or-later

// Package compose builds outgoing messages.
package compose

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	stdmime "mime"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/jhillyerd/enmime/v2"
)

// Attachment is a file attached to an outgoing message.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Draft is everything needed to build a message.
type Draft struct {
	From         mail.Address
	To           []mail.Address
	Cc           []mail.Address
	Bcc          []mail.Address
	ReplyTo      []mail.Address
	Organization string
	Subject      string
	Text         string
	HTML         string
	InReplyTo    string
	References   []string
	Attachments  []Attachment
	Date         time.Time
	// Priority is 1 (high), 3 (normal) or 5 (low).
	Priority       int
	RequestReceipt bool
}

var errHeader = errors.New("compose: control characters in a header")

func clean(s string) bool { return !strings.ContainsFunc(s, unicode.IsControl) }

func newMessageID(from string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	domain := "golub.invalid"
	if _, d, ok := strings.Cut(from, "@"); ok && d != "" {
		domain = d
	}
	return "<" + hex.EncodeToString(b) + "@" + domain + ">"
}

// Build renders the draft and returns its Message-ID; Bcc and the client name never reach the headers.
func Build(d Draft) ([]byte, string, error) {
	for _, h := range append([]string{d.Subject, d.InReplyTo, d.Organization, d.From.Name}, d.References...) {
		if !clean(h) {
			return nil, "", errHeader
		}
	}
	d.To, d.Cc, d.Bcc, d.ReplyTo = asciiList(d.To), asciiList(d.Cc), asciiList(d.Bcc), asciiList(d.ReplyTo)
	d.From = asciiList([]mail.Address{d.From})[0]
	id := newMessageID(d.From.Address)
	bcc := d.Bcc
	// Drafts may have no recipients yet; the builder insists on one, and Bcc never reaches the headers.
	if len(d.To)+len(d.Cc)+len(d.Bcc) == 0 {
		bcc = []mail.Address{d.From}
	}
	b := enmime.Builder().
		From(d.From.Name, d.From.Address).
		Subject(d.Subject).
		ToAddrs(d.To).
		CCAddrs(d.Cc).
		BCCAddrs(bcc).
		Header("Message-Id", id)
	if len(d.ReplyTo) > 0 {
		b = b.ReplyToAddrs(d.ReplyTo)
	}
	if !d.Date.IsZero() {
		b = b.Date(d.Date)
	}
	if d.Organization != "" {
		b = b.Header("Organization", d.Organization)
	}
	if d.InReplyTo != "" {
		b = b.Header("In-Reply-To", d.InReplyTo)
	}
	if len(d.References) > 0 {
		b = b.Header("References", strings.Join(d.References, " "))
	}
	switch d.Priority {
	case 1:
		b = b.Header("X-Priority", "1 (Highest)").Header("Importance", "high")
	case 5:
		b = b.Header("X-Priority", "5 (Lowest)").Header("Importance", "low")
	}
	if d.RequestReceipt {
		b = b.Header("Disposition-Notification-To", (&mail.Address{Name: d.From.Name, Address: d.From.Address}).String())
	}
	b = b.Text([]byte(d.Text))
	if d.HTML != "" {
		b = b.HTML([]byte(d.HTML))
	}
	for _, a := range d.Attachments {
		ct := a.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		b = b.AddAttachment(a.Data, ct, a.Filename)
	}
	part, err := b.Build()
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	if err := part.Encode(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), id, nil
}

// Recipients returns every envelope recipient once, lowercased, with domains in IDNA form.
func Recipients(d Draft) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]mail.Address{d.To, d.Cc, d.Bcc} {
		for _, a := range asciiList(list) {
			addr := strings.ToLower(strings.TrimSpace(a.Address))
			if addr != "" && !seen[addr] {
				seen[addr] = true
				out = append(out, addr)
			}
		}
	}
	return out
}

func prefixed(subject, prefix string) string {
	if len(subject) >= len(prefix) && strings.EqualFold(subject[:len(prefix)], prefix) {
		return subject
	}
	return prefix + subject
}

// ReplySubject adds "Re: " unless it is already there.
func ReplySubject(s string) string { return prefixed(s, "Re: ") }

// ForwardSubject adds "Fwd: " unless it is already there.
func ForwardSubject(s string) string { return prefixed(s, "Fwd: ") }

// MDN is a read receipt for a message that asked for one.
type MDN struct {
	From              mail.Address
	To                mail.Address
	OriginalSubject   string
	OriginalMessageID string
	Text              string
	Date              time.Time
}

// BuildMDN renders a message disposition notification (RFC 8098) for a manually sent receipt.
func BuildMDN(r MDN) ([]byte, error) {
	for _, h := range []string{r.OriginalSubject, r.OriginalMessageID, r.From.Name} {
		if !clean(h) {
			return nil, errHeader
		}
	}
	date := r.Date
	if date.IsZero() {
		date = time.Now()
	}
	boundary := strings.TrimSuffix(strings.TrimPrefix(newMessageID(r.From.Address), "<"), ">")
	boundary = "mdn-" + strings.SplitN(boundary, "@", 2)[0]
	subject := stdmime.QEncoding.Encode("utf-8", "Read: "+r.OriginalSubject)
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-Id: %s\r\nMIME-Version: 1.0\r\n",
		(&mail.Address{Name: r.From.Name, Address: r.From.Address}).String(), (&mail.Address{Address: r.To.Address}).String(),
		subject, date.Format(time.RFC1123Z), newMessageID(r.From.Address))
	fmt.Fprintf(&b, "Content-Type: multipart/report; report-type=disposition-notification; boundary=\"%s\"\r\n\r\n", boundary)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n", boundary, r.Text)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: message/disposition-notification\r\n\r\n", boundary)
	fmt.Fprintf(&b, "Final-Recipient: rfc822; %s\r\n", r.From.Address)
	if r.OriginalMessageID != "" {
		fmt.Fprintf(&b, "Original-Message-ID: %s\r\n", r.OriginalMessageID)
	}
	fmt.Fprintf(&b, "Disposition: manual-action/MDN-sent-manually; displayed\r\n\r\n--%s--\r\n", boundary)
	return b.Bytes(), nil
}
