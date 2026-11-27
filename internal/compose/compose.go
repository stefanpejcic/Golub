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
	"regexp"
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
	// PGP replaces the body with what the browser encrypted or signed; Text, HTML and Attachments are ignored then.
	PGP *PGPBody
}

// PGPBody is a PGP/MIME body made in the browser (RFC 3156): an encrypted message, or a part and its detached signature.
type PGPBody struct {
	Encrypted    string `json:"encrypted,omitempty"`
	SignedEntity string `json:"signedEntity,omitempty"`
	Signature    string `json:"signature,omitempty"`
	MicAlg       string `json:"micalg,omitempty"`
}

var (
	errPGP   = errors.New("compose: invalid PGP body")
	micAlgRe = regexp.MustCompile(`^pgp-(sha256|sha384|sha512|sha224|sha1)$`)
)

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
	if d.PGP != nil {
		return buildPGP(b, d.PGP, id)
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

// buildPGP takes the headers the builder made and puts the PGP/MIME body under them.
func buildPGP(b enmime.MailBuilder, p *PGPBody, id string) ([]byte, string, error) {
	body, ctype, err := pgpBody(p)
	if err != nil {
		return nil, "", err
	}
	part, err := b.Text([]byte("x")).Build()
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	if err := part.Encode(&buf); err != nil {
		return nil, "", err
	}
	head, _, _ := strings.Cut(buf.String(), "\r\n\r\n")
	var out strings.Builder
	out.WriteString(dropHeaders(head, "content-type", "content-transfer-encoding"))
	out.WriteString("\r\nContent-Type: " + ctype + "\r\n\r\n" + body)
	return []byte(out.String()), id, nil
}

func pgpBody(p *PGPBody) (body, ctype string, err error) {
	boundary := "golub-pgp-" + strings.Trim(strings.SplitN(newMessageID("x"), "@", 2)[0], "<")
	crlf := func(s string) string {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
		return strings.TrimRight(s, "\r\n") + "\r\n"
	}
	switch {
	case p.Encrypted != "" && p.SignedEntity == "" && p.Signature == "":
		if !strings.HasPrefix(p.Encrypted, "-----BEGIN PGP MESSAGE-----") {
			return "", "", errPGP
		}
		ctype = `multipart/encrypted; protocol="application/pgp-encrypted"; boundary="` + boundary + `"`
		body = "This is an OpenPGP/MIME encrypted message (RFC 4880 and 3156)\r\n" +
			"--" + boundary + "\r\nContent-Type: application/pgp-encrypted\r\nContent-Description: PGP/MIME version identification\r\n\r\nVersion: 1\r\n\r\n" +
			"--" + boundary + "\r\nContent-Type: application/octet-stream; name=\"encrypted.asc\"\r\nContent-Description: OpenPGP encrypted message\r\nContent-Disposition: inline; filename=\"encrypted.asc\"\r\n\r\n" +
			crlf(p.Encrypted) + "--" + boundary + "--\r\n"
	case p.Encrypted == "" && p.SignedEntity != "" && p.Signature != "":
		head, _, ok := strings.Cut(p.SignedEntity, "\r\n\r\n")
		if !ok || !strings.HasPrefix(strings.ToLower(head), "content-type:") || !micAlgRe.MatchString(p.MicAlg) ||
			!strings.HasPrefix(p.Signature, "-----BEGIN PGP SIGNATURE-----") || strings.Contains(p.SignedEntity, boundary) {
			return "", "", errPGP
		}
		ctype = "multipart/signed; micalg=" + p.MicAlg + `; protocol="application/pgp-signature"; boundary="` + boundary + `"`
		// the signed part goes out exactly as the browser signed it, a single changed byte breaks the signature
		body = "This is an OpenPGP/MIME signed message (RFC 4880 and 3156)\r\n" +
			"--" + boundary + "\r\n" + p.SignedEntity + "\r\n" +
			"--" + boundary + "\r\nContent-Type: application/pgp-signature; name=\"signature.asc\"\r\nContent-Description: OpenPGP digital signature\r\nContent-Disposition: attachment; filename=\"signature.asc\"\r\n\r\n" +
			crlf(p.Signature) + "--" + boundary + "--\r\n"
	default:
		return "", "", errPGP
	}
	return body, ctype, nil
}

// dropHeaders removes the named headers, folded continuation lines included.
func dropHeaders(head string, names ...string) string {
	var out []string
	skip := false
	for _, line := range strings.Split(head, "\r\n") {
		if line != "" && (line[0] == ' ' || line[0] == '\t') {
			if !skip {
				out = append(out, line)
			}
			continue
		}
		name, _, _ := strings.Cut(line, ":")
		skip = false
		for _, n := range names {
			if strings.EqualFold(strings.TrimSpace(name), n) {
				skip = true
			}
		}
		if !skip {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\r\n")
}
