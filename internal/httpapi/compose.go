// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	stdmime "mime"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/compose"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/mime"
	"github.com/stefanpejcic/golub/internal/sanitize"
	"github.com/stefanpejcic/golub/internal/settings"
	"github.com/stefanpejcic/golub/internal/smtpx"
)

// MailSender submits a finished message.
type MailSender interface {
	Send(ctx context.Context, user, pass, from string, to []string, msg []byte) error
}

// Limits for outgoing mail.
const (
	MaxOutgoing      = 25 << 20
	maxRecipients    = 100
	maxComposeFields = 1 << 20
)

type partRef struct {
	Folder string `json:"folder"`
	UID    uint32 `json:"uid"`
	Part   string `json:"part"`
}

type answeredRef struct {
	Folder    string `json:"folder"`
	UID       uint32 `json:"uid"`
	Forwarded bool   `json:"forwarded"`
}

type composeRequest struct {
	Identity   string       `json:"identity"`
	To         []string     `json:"to"`
	Cc         []string     `json:"cc"`
	Bcc        []string     `json:"bcc"`
	Subject    string       `json:"subject"`
	Text       string       `json:"text"`
	HTML       string       `json:"html"`
	InReplyTo  string       `json:"inReplyTo"`
	References []string     `json:"references"`
	Attach     []partRef    `json:"attach"`
	Answered   *answeredRef `json:"answered"`
	DraftUID   uint32       `json:"draftUid"`
}

type composeError struct {
	status  int
	code    string
	address string
}

func (e *composeError) Error() string { return e.code }

func badRequest(code, address string) *composeError {
	return &composeError{status: http.StatusBadRequest, code: code, address: address}
}

func (a *authAPI) writeComposeError(w http.ResponseWriter, r *http.Request, err error) {
	var ce *composeError
	var rej *smtpx.RecipientError
	switch {
	case errors.As(err, &ce):
		body := map[string]string{"error": ce.code}
		if ce.address != "" {
			body["address"] = ce.address
		}
		writeJSON(w, ce.status, body)
	case errors.As(err, &rej):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "compose.recipient_rejected", "address": rej.Address})
	case errors.Is(err, smtpx.ErrAuth), errors.Is(err, smtpx.ErrUnavailable):
		slog.Error("smtp submission failed", "err", err)
		writeError(w, http.StatusBadGateway, "compose.send_failed")
	default:
		a.mailError(w, r, err)
	}
}

// readCompose accepts JSON, or multipart with a "message" JSON field plus "attachment" files.
func readCompose(w http.ResponseWriter, r *http.Request) (composeRequest, []compose.Attachment, bool) {
	var req composeRequest
	mt, _, _ := stdmime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt != "multipart/form-data" {
		return req, nil, decodeJSON(w, r, &req)
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxOutgoing+maxComposeFields)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return req, nil, false
	}
	var files []compose.Attachment
	total := 0
	gotMeta := false
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "compose.too_large")
			return req, nil, false
		}
		switch p.FormName() {
		case "message":
			b, err := io.ReadAll(io.LimitReader(p, maxComposeFields))
			if err != nil || json.Unmarshal(b, &req) != nil {
				writeError(w, http.StatusBadRequest, "request.invalid")
				return req, nil, false
			}
			gotMeta = true
		case "attachment":
			b, err := io.ReadAll(io.LimitReader(p, int64(MaxOutgoing-total+1)))
			if err != nil {
				writeError(w, http.StatusBadRequest, "request.invalid")
				return req, nil, false
			}
			total += len(b)
			if total > MaxOutgoing {
				writeError(w, http.StatusRequestEntityTooLarge, "compose.too_large")
				return req, nil, false
			}
			files = append(files, compose.Attachment{Filename: p.FileName(), ContentType: p.Header.Get("Content-Type"), Data: b})
		}
	}
	if !gotMeta {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return req, nil, false
	}
	return req, files, true
}

func parseAddresses(list []string) ([]mail.Address, error) {
	out := make([]mail.Address, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		a, err := mail.ParseAddress(s)
		if err != nil || !auth.ValidEmail(a.Address) {
			return nil, badRequest("compose.invalid_recipient", s)
		}
		out = append(out, *a)
	}
	return out, nil
}

// draftFrom turns a request into a message, resolving the identity and attachments by reference.
func (a *authAPI) draftFrom(c *imapclient.Client, req composeRequest, files []compose.Attachment, st settings.Settings) (compose.Draft, error) {
	var d compose.Draft
	var id *settings.Identity
	for i := range st.Identities {
		if st.Identities[i].ID == req.Identity {
			id = &st.Identities[i]
		}
	}
	if id == nil {
		return d, badRequest("compose.unknown_identity", "")
	}
	to, err := parseAddresses(req.To)
	if err != nil {
		return d, err
	}
	cc, err := parseAddresses(req.Cc)
	if err != nil {
		return d, err
	}
	bcc, err := parseAddresses(req.Bcc)
	if err != nil {
		return d, err
	}
	if id.Bcc != "" {
		if extra, err := mail.ParseAddress(id.Bcc); err == nil {
			bcc = append(bcc, *extra)
		}
	}
	d = compose.Draft{
		From:         mail.Address{Name: id.Name, Address: id.Email},
		To:           to,
		Cc:           cc,
		Bcc:          bcc,
		Organization: id.Organization,
		Subject:      req.Subject,
		Text:         req.Text,
		InReplyTo:    req.InReplyTo,
		References:   req.References,
		Attachments:  files,
		Date:         time.Now(),
	}
	if id.ReplyTo != "" {
		if rt, err := mail.ParseAddress(id.ReplyTo); err == nil {
			d.ReplyTo = []mail.Address{*rt}
		}
	}
	if req.HTML != "" {
		keep := func(u string) string { return u }
		d.HTML = sanitize.HTML(req.HTML, sanitize.Options{RemoteImages: true, Proxy: keep}).HTML
	}

	size := 0
	for _, f := range files {
		size += len(f.Data)
	}
	parsed := map[string]*mime.Message{}
	for _, ref := range req.Attach {
		key := ref.Folder + "\x00" + strconv.FormatUint(uint64(ref.UID), 10)
		m := parsed[key]
		if m == nil {
			raw, err := imapx.FetchRaw(c, ref.Folder, ref.UID, MaxMessageSize)
			if err != nil {
				return d, err
			}
			if m, err = mime.Parse(raw.Data); err != nil {
				return d, badRequest("message.unreadable", "")
			}
			parsed[key] = m
		}
		p := m.Part(ref.Part)
		if p == nil {
			return d, &composeError{status: http.StatusNotFound, code: "message.part_not_found"}
		}
		size += len(p.Content)
		if size > MaxOutgoing {
			return d, &composeError{status: http.StatusRequestEntityTooLarge, code: "compose.too_large"}
		}
		d.Attachments = append(d.Attachments, compose.Attachment{Filename: p.FileName, ContentType: p.ContentType, Data: p.Content})
	}
	return d, nil
}

func (a *authAPI) composeSend(w http.ResponseWriter, r *http.Request, s auth.Session) {
	req, files, ok := readCompose(w, r)
	if !ok {
		return
	}
	var raw []byte
	var d compose.Draft
	var st settings.Settings
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		if st, err = loadSettings(c, s.Email); err != nil {
			return err
		}
		if d, err = a.draftFrom(c, req, files, st); err != nil {
			return err
		}
		rcpts := compose.Recipients(d)
		if len(rcpts) == 0 {
			return badRequest("compose.no_recipients", "")
		}
		if len(rcpts) > maxRecipients {
			return badRequest("compose.too_many_recipients", "")
		}
		raw, _, err = compose.Build(d)
		if err != nil {
			return badRequest("request.invalid", "")
		}
		return nil
	})
	if err != nil {
		a.writeComposeError(w, r, err)
		return
	}

	user, pass := a.imapCreds(s)
	if err := a.o.SMTP.Send(r.Context(), user, pass, d.From.Address, compose.Recipients(d), raw); err != nil {
		a.writeComposeError(w, r, err)
		return
	}

	// The mail is out; failures below only cost the Sent copy or a flag, so they are reported, not fatal.
	saved := false
	err = a.do(r, s, func(c *imapclient.Client) error {
		folders, err := imapx.ListFolders(c)
		if err != nil {
			return err
		}
		sent := specialFolder(st, folders, "sent", "Sent")
		if sent == "" {
			sent = "Sent"
			if err := imapx.EnsureFolder(c, sent, "sent"); err != nil {
				return err
			}
		}
		if _, err := imapx.AppendMessage(c, sent, raw, imapx.FlagSeen); err != nil {
			return err
		}
		saved = true
		if req.Answered != nil && req.Answered.UID != 0 && validFolder(req.Answered.Folder) {
			flag := imapx.FlagAnswered
			if req.Answered.Forwarded {
				flag = imapx.FlagForwarded
			}
			_ = imapx.SetFlags(c, req.Answered.Folder, []uint32{req.Answered.UID}, flag, true)
		}
		if req.DraftUID != 0 {
			if drafts := specialFolder(st, folders, "drafts", "Drafts"); drafts != "" {
				_ = imapx.Expunge(c, drafts, []uint32{req.DraftUID})
			}
		}
		return nil
	})
	if err != nil {
		slog.Warn("post-send bookkeeping failed", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"sent": true, "savedToSent": saved})
}

func (a *authAPI) composeDraft(w http.ResponseWriter, r *http.Request, s auth.Session) {
	req, files, ok := readCompose(w, r)
	if !ok {
		return
	}
	var uid uint32
	err := a.do(r, s, func(c *imapclient.Client) error {
		st, err := loadSettings(c, s.Email)
		if err != nil {
			return err
		}
		d, err := a.draftFrom(c, req, files, st)
		if err != nil {
			return err
		}
		raw, _, err := compose.Build(d)
		if err != nil {
			return badRequest("request.invalid", "")
		}
		folders, err := imapx.ListFolders(c)
		if err != nil {
			return err
		}
		drafts := specialFolder(st, folders, "drafts", "Drafts")
		if drafts == "" {
			drafts = "Drafts"
			if err := imapx.EnsureFolder(c, drafts, "drafts"); err != nil {
				return err
			}
		}
		if uid, err = imapx.AppendMessage(c, drafts, raw, imapx.FlagSeen, imapx.FlagDraft); err != nil {
			return err
		}
		if req.DraftUID != 0 && req.DraftUID != uid {
			_ = imapx.Expunge(c, drafts, []uint32{req.DraftUID})
		}
		return nil
	})
	if err != nil {
		a.writeComposeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]uint32{"uid": uid})
}

type quote struct {
	From mime.Address `json:"from"`
	Date time.Time    `json:"date"`
	Text string       `json:"text"`
	HTML string       `json:"html"`
}

type composeSource struct {
	Subject     string            `json:"subject"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc"`
	Bcc         []string          `json:"bcc"`
	InReplyTo   string            `json:"inReplyTo,omitempty"`
	References  []string          `json:"references,omitempty"`
	Quote       *quote            `json:"quote,omitempty"`
	Text        string            `json:"text,omitempty"`
	HTML        string            `json:"html,omitempty"`
	Attachments []mime.Attachment `json:"attachments"`
	Identity    string            `json:"identity,omitempty"`
}

func formatAddr(a mime.Address) string {
	return (&mail.Address{Name: a.Name, Address: a.Address}).String()
}

// composeSourceFor prepares reply, reply-all, forward and edit-as-new from an existing message.
func (a *authAPI) composeSourceFor(w http.ResponseWriter, r *http.Request, s auth.Session) {
	mode := r.URL.Query().Get("mode")
	if mode != "reply" && mode != "replyAll" && mode != "forward" && mode != "edit" {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	ref, ok := parseRef(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var st settings.Settings
	if err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		st, err = loadSettings(c, s.Email)
		return err
	}); err != nil {
		a.mailError(w, r, err)
		return
	}
	m, _, ok := a.load(w, r, s, ref, false)
	if !ok {
		return
	}

	mine := map[string]string{strings.ToLower(s.Email): ""}
	for _, id := range st.Identities {
		mine[strings.ToLower(id.Email)] = id.ID
	}
	isMine := func(addr string) bool { _, ok := mine[strings.ToLower(addr)]; return ok }

	out := composeSource{To: []string{}, Cc: []string{}, Bcc: []string{}, Attachments: []mime.Attachment{}}
	// Answer from the identity the original was addressed to.
	for _, list := range [][]mime.Address{m.To, m.Cc} {
		for _, addr := range list {
			if id := mine[strings.ToLower(addr.Address)]; id != "" && out.Identity == "" {
				out.Identity = id
			}
		}
	}

	q := &quote{Date: m.Date, Text: m.Text}
	if len(m.From) > 0 {
		q.From = m.From[0]
	}
	if m.HTML != "" {
		q.HTML = sanitize.HTML(m.HTML, sanitize.Options{CID: func(id string) (string, bool) {
			p := m.CID(id)
			if p == nil || len(p.Content) > 2<<20 {
				return "", false
			}
			uri := sanitize.DataURI(p.ContentType, p.Content)
			return uri, uri != ""
		}}).HTML
	}

	switch mode {
	case "reply", "replyAll":
		out.Subject = compose.ReplySubject(m.Subject)
		out.InReplyTo = m.MessageID
		out.References = append(splitRefs(m.References), m.MessageID)
		target := m.ReplyTo
		if len(target) == 0 {
			target = m.From
		}
		seen := map[string]bool{}
		add := func(list *[]string, addrs []mime.Address) {
			for _, addr := range addrs {
				key := strings.ToLower(addr.Address)
				if key == "" || seen[key] || (isMine(addr.Address) && mode == "replyAll") {
					continue
				}
				seen[key] = true
				*list = append(*list, formatAddr(addr))
			}
		}
		add(&out.To, target)
		if mode == "replyAll" {
			add(&out.To, m.To)
			add(&out.Cc, m.Cc)
		}
		out.Quote = q
	case "forward":
		out.Subject = compose.ForwardSubject(m.Subject)
		out.Quote = q
		out.Attachments = m.Attachments
	case "edit":
		out.Subject = m.Subject
		for _, addr := range m.To {
			out.To = append(out.To, formatAddr(addr))
		}
		for _, addr := range m.Cc {
			out.Cc = append(out.Cc, formatAddr(addr))
		}
		out.Text = m.Text
		out.HTML = q.HTML
		out.Attachments = m.Attachments
	}
	if out.InReplyTo == "" {
		out.References = nil
	}
	if out.Attachments == nil {
		out.Attachments = []mime.Attachment{}
	}
	writeJSON(w, http.StatusOK, out)
}

func splitRefs(s string) []string {
	var out []string
	for _, f := range strings.Fields(s) {
		if strings.HasPrefix(f, "<") && strings.HasSuffix(f, ">") {
			out = append(out, f)
		}
	}
	return out
}
