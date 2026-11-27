// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	stdmime "mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2/imapclient"

	"golang.org/x/text/encoding/htmlindex"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/mime"
	"github.com/stefanpejcic/golub/internal/sanitize"
)

// ImageProxy serves and signs remote image URLs and performs one-click unsubscribes.
type ImageProxy interface {
	http.Handler
	URL(target string) string
	Unsubscribe(ctx context.Context, target string) error
}

// MaxMessageSize caps what is downloaded and parsed for one message.
const MaxMessageSize = 50 << 20

// Cap on inlined cid images so one message cannot balloon the iframe document.
const maxInlineBytes = 10 << 20

type messageView struct {
	UID           uint32            `json:"uid"`
	Folder        string            `json:"folder"`
	Subject       string            `json:"subject"`
	From          []mime.Address    `json:"from"`
	To            []mime.Address    `json:"to"`
	Cc            []mime.Address    `json:"cc"`
	ReplyTo       []mime.Address    `json:"replyTo"`
	Date          time.Time         `json:"date"`
	MessageID     string            `json:"messageId"`
	Seen          bool              `json:"seen"`
	Flagged       bool              `json:"flagged"`
	Text          string            `json:"text"`
	HTML          bool              `json:"html"`
	BlockedImages int               `json:"blockedImages"`
	Trackers      int               `json:"trackers"`
	Priority      int               `json:"priority"`
	Unsubscribe   mime.Unsubscribe  `json:"unsubscribe"`
	ReceiptTo     string            `json:"receiptTo,omitempty"`
	Attachments   []mime.Attachment `json:"attachments"`
}

type messageRef struct {
	folder string
	uid    uint32
}

func parseRef(r *http.Request) (messageRef, bool) {
	q := r.URL.Query()
	uid, err := strconv.ParseUint(q.Get("uid"), 10, 32)
	folder := q.Get("folder")
	if err != nil || uid == 0 || !validFolder(folder) {
		return messageRef{}, false
	}
	return messageRef{folder: folder, uid: uint32(uid)}, true
}

// load fetches and parses a message; markSeen sets \Seen when it was unread.
func (a *authAPI) load(w http.ResponseWriter, r *http.Request, s auth.Session, ref messageRef, markSeen bool) (*mime.Message, imapx.Raw, bool) {
	var raw imapx.Raw
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		if raw, err = imapx.FetchRaw(c, ref.folder, ref.uid, MaxMessageSize); err != nil {
			return err
		}
		if markSeen && !raw.Seen {
			if err := imapx.SetFlag(c, ref.folder, ref.uid, imapx.FlagSeen, true); err != nil {
				return err
			}
			raw.Seen = true
		}
		return nil
	})
	switch {
	case errors.Is(err, imapx.ErrNotFound):
		writeError(w, http.StatusNotFound, "message.not_found")
		return nil, raw, false
	case errors.Is(err, imapx.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "message.too_large")
		return nil, raw, false
	case err != nil:
		a.mailError(w, r, err)
		return nil, raw, false
	}
	m, err := mime.Parse(raw.Data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "message.unreadable")
		return nil, raw, false
	}
	return m, raw, true
}

func (a *authAPI) message(w http.ResponseWriter, r *http.Request, s auth.Session) {
	ref, ok := parseRef(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	m, raw, ok := a.load(w, r, s, ref, r.URL.Query().Get("seen") != "0")
	if !ok {
		return
	}
	v := messageView{
		UID: ref.uid, Folder: ref.folder, Subject: m.Subject,
		From: m.From, To: m.To, Cc: m.Cc, ReplyTo: m.ReplyTo, Date: m.Date, MessageID: m.MessageID,
		Seen: raw.Seen, Flagged: raw.Flagged, Text: m.Text, HTML: m.HTML != "",
		Attachments: m.Attachments, Priority: m.Priority, Unsubscribe: m.Unsubscribe,
	}
	if !raw.MDNSent {
		v.ReceiptTo = m.ReceiptTo
	}
	if v.Attachments == nil {
		v.Attachments = []mime.Attachment{}
	}
	if v.HTML {
		res := sanitize.HTML(m.HTML, sanitize.Options{})
		v.BlockedImages, v.Trackers = res.BlockedImages, res.Trackers
	}
	writeJSON(w, http.StatusOK, v)
}

func (a *authAPI) messageBody(w http.ResponseWriter, r *http.Request, s auth.Session) {
	ref, ok := parseRef(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	m, _, ok := a.load(w, r, s, ref, false)
	if !ok {
		return
	}
	src := m.HTML
	if src == "" {
		src = "<pre>" + htmlEscape(m.Text) + "</pre>"
	}
	inlined := 0
	opts := sanitize.Options{
		RemoteImages: r.URL.Query().Get("images") == "1" && a.o.Proxy != nil,
		CID: func(id string) (string, bool) {
			p := m.CID(id)
			if p == nil || inlined+len(p.Content) > maxInlineBytes {
				return "", false
			}
			inlined += len(p.Content)
			uri := sanitize.DataURI(p.ContentType, p.Content)
			return uri, uri != ""
		},
	}
	if a.o.Proxy != nil {
		opts.Proxy = a.o.Proxy.URL
	}
	res := sanitize.HTML(src, opts)

	origin := "http://" + r.Host
	if isSecure(r, a.o.HTTPS, a.o.TrustProxy) {
		origin = "https://" + r.Host
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", fmt.Sprintf("default-src 'none'; img-src data: %s/proxy; style-src 'unsafe-inline'; "+
		"base-uri 'none'; form-action 'none'; frame-ancestors 'self'", origin))
	h.Set("X-Frame-Options", "SAMEORIGIN")
	_, _ = w.Write([]byte(sanitize.Document(res.HTML)))
}

func (a *authAPI) messagePart(w http.ResponseWriter, r *http.Request, s auth.Session) {
	ref, ok := parseRef(r)
	id := r.URL.Query().Get("part")
	if !ok || id == "" || len(id) > 64 {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	m, _, ok := a.load(w, r, s, ref, false)
	if !ok {
		return
	}
	p := m.Part(id)
	if p == nil {
		writeError(w, http.StatusNotFound, "message.part_not_found")
		return
	}
	name := p.FileName
	if name == "" {
		name = "attachment"
	}
	h := w.Header()
	content := p.Content
	h.Set("Content-Type", downloadType(p.ContentType))
	h.Set("Content-Disposition", contentDisposition(name))
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if r.URL.Query().Get("inline") == "1" && previewable(p.ContentType) {
		h.Set("Content-Disposition", "inline;"+strings.TrimPrefix(contentDisposition(name), "attachment;"))
		mt, params, _ := stdmime.ParseMediaType(p.ContentType)
		switch {
		case mt == "application/pdf":
			// Chrome will not render a PDF under a sandbox CSP.
			h.Set("Content-Security-Policy", "default-src 'none'; object-src 'self'; style-src 'unsafe-inline'; frame-ancestors 'self'")
		case strings.HasPrefix(mt, "text/"):
			content = toUTF8(content, params["charset"])
			h.Set("Content-Type", "text/plain; charset=utf-8")
		}
	}
	h.Set("Content-Length", strconv.Itoa(len(content)))
	_, _ = w.Write(content) //nolint:gosec // nosniff, a strict CSP and only safe types are ever served inline
}

var previewTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true, "image/avif": true, "image/bmp": true,
	"application/pdf": true, "text/plain": true, "text/csv": true, "text/markdown": true,
}

// previewable lists the attachment types the browser may show inline.
func previewable(ct string) bool {
	mt, _, err := stdmime.ParseMediaType(ct)
	return err == nil && previewTypes[mt]
}

func toUTF8(b []byte, charset string) []byte {
	if charset == "" || utf8.Valid(b) {
		return b
	}
	enc, err := htmlindex.Get(charset)
	if err != nil {
		return b
	}
	out, err := enc.NewDecoder().Bytes(b)
	if err != nil {
		return b
	}
	return out
}

var riskyTypes = map[string]bool{
	"text/html": true, "application/xhtml+xml": true, "image/svg+xml": true, "text/xml": true,
	"application/xml": true, "text/javascript": true, "application/javascript": true, "application/x-javascript": true,
}

// downloadType keeps harmless types so the browser can preview them, everything else is opaque.
func downloadType(ct string) string {
	mt, _, err := stdmime.ParseMediaType(ct)
	if err != nil || mt == "" || riskyTypes[mt] {
		return "application/octet-stream"
	}
	return mt
}

// contentDisposition builds an RFC 6266 header with an ASCII fallback and UTF-8 filename*.
func contentDisposition(name string) string {
	var ascii, enc strings.Builder
	for _, r := range name {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			ascii.WriteByte('_')
		} else {
			ascii.WriteRune(r)
		}
	}
	for _, b := range []byte(name) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || strings.IndexByte("!#$&+-.^_`|~", b) >= 0 {
			enc.WriteByte(b)
		} else {
			fmt.Fprintf(&enc, "%%%02X", b)
		}
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii.String(), enc.String())
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// messageRaw serves the original message as a download, as source text, or just its header block.
func (a *authAPI) messageRaw(w http.ResponseWriter, r *http.Request, s auth.Session) {
	ref, ok := parseRef(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var raw imapx.Raw
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		raw, err = imapx.FetchRaw(c, ref.folder, ref.uid, MaxMessageSize)
		return err
	})
	switch {
	case errors.Is(err, imapx.ErrNotFound):
		writeError(w, http.StatusNotFound, "message.not_found")
		return
	case errors.Is(err, imapx.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "message.too_large")
		return
	case err != nil:
		a.mailError(w, r, err)
		return
	}
	data := raw.Data
	q := r.URL.Query()
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if q.Get("headers") == "1" {
		if end := bytes.Index(data, []byte("\r\n\r\n")); end >= 0 {
			data = data[:end]
		} else if end := bytes.Index(data, []byte("\n\n")); end >= 0 {
			data = data[:end]
		}
	}
	if q.Get("download") == "1" {
		name := "message"
		if m, err := mime.Parse(raw.Data); err == nil && strings.TrimSpace(m.Subject) != "" {
			name = strings.Map(func(r rune) rune {
				if strings.ContainsRune(`/\:*?"<>|`, r) {
					return '_'
				}
				return r
			}, strings.TrimSpace(m.Subject))
		}
		h.Set("Content-Type", "message/rfc822")
		h.Set("Content-Disposition", contentDisposition(name+".eml"))
	} else {
		h.Set("Content-Type", "text/plain; charset=utf-8")
		h.Set("Content-Disposition", "inline")
	}
	_, _ = w.Write(data) //nolint:gosec // sandboxed, nosniff and never served as HTML
}
