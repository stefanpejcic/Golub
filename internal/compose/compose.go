// SPDX-License-Identifier: GPL-3.0-or-later

// Package compose builds outgoing messages.
package compose

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
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

// Build renders the draft as RFC 5322 bytes and returns its Message-ID. Bcc never appears in headers.
func Build(d Draft, userAgent string) ([]byte, string, error) {
	for _, h := range append([]string{d.Subject, d.InReplyTo, d.Organization, d.From.Name}, d.References...) {
		if !clean(h) {
			return nil, "", errHeader
		}
	}
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
		Header("Message-Id", id).
		Header("User-Agent", userAgent)
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

// Recipients returns every envelope recipient once, lowercased.
func Recipients(d Draft) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]mail.Address{d.To, d.Cc, d.Bcc} {
		for _, a := range list {
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
