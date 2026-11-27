// SPDX-License-Identifier: GPL-3.0-or-later

// Package mime turns raw messages into the parts the UI shows and downloads.
package mime

import (
	"bytes"
	"errors"
	stdmime "mime"
	"net/mail"
	"strings"
	"time"

	"github.com/jhillyerd/enmime/v2"
	"golang.org/x/net/idna"
)

// Address is a display name and email.
type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// Attachment describes a downloadable part.
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	Inline      bool   `json:"inline"`
}

// Unsubscribe holds the List-Unsubscribe targets (RFC 2369) and whether one-click works (RFC 8058).
type Unsubscribe struct {
	Mailto   string `json:"mailto,omitempty"`
	URL      string `json:"url,omitempty"`
	OneClick bool   `json:"oneClick,omitempty"`
}

// Message is a parsed message ready for display.
type Message struct {
	Subject     string
	From        []Address
	To          []Address
	Cc          []Address
	ReplyTo     []Address
	Date        time.Time
	MessageID   string
	References  string
	Unsubscribe Unsubscribe
	// PGP is "encrypted" or "signed" for PGP/MIME, "inline-encrypted" or "inline-signed" for armored text, or empty
	PGP         string
	Priority    int
	ReceiptTo   string
	Text        string
	HTML        string
	Attachments []Attachment

	parts map[string]*enmime.Part
	cids  map[string]*enmime.Part
}

// MaxParts bounds work on hostile messages with absurd part counts.
const MaxParts = 1000

var parser = enmime.NewParser(enmime.SkipMalformedParts(true), enmime.MaxMIMEParts(MaxParts))

// Parse reads a raw RFC 5322 message.
func Parse(raw []byte) (*Message, error) {
	env, err := parser.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if env.Root == nil {
		return nil, errors.New("mime: empty message")
	}
	m := &Message{
		Subject:    env.GetHeader("Subject"),
		From:       addresses(env, "From"),
		To:         addresses(env, "To"),
		Cc:         addresses(env, "Cc"),
		ReplyTo:    addresses(env, "Reply-To"),
		MessageID:  strings.TrimSpace(env.GetHeader("Message-ID")),
		References: strings.TrimSpace(env.GetHeader("References")),
		Text:       env.Text,
		HTML:       env.HTML,
		parts:      map[string]*enmime.Part{},
		cids:       map[string]*enmime.Part{},
	}
	if d, err := env.Date(); err == nil {
		m.Date = d
	}
	m.PGP = detectPGP(env)
	m.Unsubscribe = parseUnsubscribe(env.GetHeader("List-Unsubscribe"), env.GetHeader("List-Unsubscribe-Post"))
	m.Priority = ParsePriority(env.GetHeader("X-Priority"), env.GetHeader("Importance"))
	if a, err := mail.ParseAddress(env.GetHeader("Disposition-Notification-To")); err == nil {
		m.ReceiptTo = a.Address
	}
	if m.Text == "" && m.HTML == "" && len(env.Attachments)+len(env.Inlines) == 0 {
		m.Text = rawBody(raw)
	}

	referenced := map[string]bool{}
	for _, p := range env.Inlines {
		if p.ContentID != "" {
			referenced[p.PartID] = strings.Contains(m.HTML, "cid:"+p.ContentID)
		}
	}
	for _, list := range [][]*enmime.Part{env.Attachments, env.Inlines, env.OtherParts} {
		for _, p := range list {
			if p.PartID == "" || m.parts[p.PartID] != nil || strings.HasPrefix(p.ContentType, "multipart/") {
				continue
			}
			m.parts[p.PartID] = p
			if p.ContentID != "" {
				m.cids[p.ContentID] = p
			}
			// Inline images the HTML actually shows are not offered again as attachments.
			if referenced[p.PartID] || (p.ContentID != "" && strings.Contains(m.HTML, "cid:"+p.ContentID)) {
				continue
			}
			m.Attachments = append(m.Attachments, Attachment{
				ID:          p.PartID,
				Filename:    filename(p),
				ContentType: p.ContentType,
				Size:        len(p.Content),
				Inline:      strings.EqualFold(p.Disposition, "inline"),
			})
		}
	}
	return m, nil
}

// Part returns a downloadable part by ID.
func (m *Message) Part(id string) *enmime.Part {
	return m.parts[id]
}

// CID returns the part with the given Content-ID.
func (m *Message) CID(id string) *enmime.Part {
	return m.cids[strings.Trim(id, "<>")]
}

func parseUnsubscribe(header, post string) Unsubscribe {
	var u Unsubscribe
	for _, part := range strings.Split(header, ",") {
		v := strings.Trim(strings.TrimSpace(part), "<>")
		lower := strings.ToLower(v)
		switch {
		case strings.HasPrefix(lower, "mailto:") && u.Mailto == "":
			u.Mailto = v
		case strings.HasPrefix(lower, "https://") && u.URL == "":
			u.URL = v
		}
	}
	u.OneClick = u.URL != "" && strings.EqualFold(strings.TrimSpace(post), "List-Unsubscribe=One-Click")
	return u
}

// ParsePriority maps X-Priority and Importance to 1 (high), 3 (normal) or 5 (low).
func ParsePriority(xPriority, importance string) int {
	if f := strings.Fields(xPriority); len(f) > 0 {
		switch f[0] {
		case "1", "2":
			return 1
		case "4", "5":
			return 5
		}
	}
	switch strings.ToLower(strings.TrimSpace(importance)) {
	case "high":
		return 1
	case "low":
		return 5
	}
	return 3
}

// rawBody shows something useful when the structure is too broken to parse, e.g. a wrong boundary.
func rawBody(raw []byte) string {
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if _, body, ok := bytes.Cut(raw, []byte(sep)); ok {
			return strings.ToValidUTF8(string(body), "\uFFFD")
		}
	}
	return ""
}

func filename(p *enmime.Part) string {
	if p.FileName != "" {
		return p.FileName
	}
	if p.ContentType == "message/rfc822" {
		return "message.eml"
	}
	return "part-" + p.PartID
}

func addresses(env *enmime.Envelope, key string) []Address {
	list, err := env.AddressList(key)
	out := make([]Address, 0, len(list))
	if err != nil {
		return out
	}
	for _, a := range list {
		addr := a.Address
		if local, domain, ok := strings.Cut(addr, "@"); ok && strings.Contains(strings.ToLower(domain), "xn--") {
			if u, err := idna.Display.ToUnicode(domain); err == nil {
				addr = local + "@" + u
			}
		}
		out = append(out, Address{Name: a.Name, Address: addr})
	}
	return out
}

func detectPGP(env *enmime.Envelope) string {
	// enmime leaves the parameters of a multipart root empty, so read them from the header
	if mt, params, err := stdmime.ParseMediaType(env.GetHeader("Content-Type")); err == nil {
		proto := strings.ToLower(params["protocol"])
		switch {
		case mt == "multipart/encrypted" && proto == "application/pgp-encrypted":
			return "encrypted"
		case mt == "multipart/signed" && proto == "application/pgp-signature":
			return "signed"
		}
	}
	// armor only counts at the start of a line, not mentioned in a sentence
	text := "\n" + strings.ReplaceAll(env.Text, "\r\n", "\n")
	switch {
	case strings.Contains(text, "\n-----BEGIN PGP MESSAGE-----\n"):
		return "inline-encrypted"
	case strings.Contains(text, "\n-----BEGIN PGP SIGNED MESSAGE-----\n"):
		return "inline-signed"
	}
	return ""
}
