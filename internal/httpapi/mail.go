// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
)

// MailPool runs IMAP work on a user's pooled connection.
type MailPool interface {
	Do(ctx context.Context, user, pass string, fn func(*imapclient.Client) error) error
}

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// withSession rejects requests without a valid session.
func (a *authAPI) withSession(h func(http.ResponseWriter, *http.Request, auth.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		s, ok := a.current(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "auth.required")
			return
		}
		h(w, r, s)
	}
}

// imapCreds returns the IMAP login for a session; SSO sessions go through the master user.
func (a *authAPI) imapCreds(s auth.Session) (string, string) {
	if s.Mode == auth.ModeSSO {
		return s.Email + "*" + a.o.MasterUser, a.o.MasterPassword
	}
	return s.Email, s.Password
}

func (a *authAPI) do(r *http.Request, s auth.Session, fn func(*imapclient.Client) error) error {
	user, pass := a.imapCreds(s)
	return a.o.Pool.Do(r.Context(), user, pass, fn)
}

// mailError maps IMAP failures to stable API error codes.
func (a *authAPI) mailError(w http.ResponseWriter, r *http.Request, err error) {
	var imapErr *imap.Error
	switch {
	case errors.Is(err, imapx.ErrInvalidCredentials):
		a.clearSession(w, r)
		writeError(w, http.StatusUnauthorized, "auth.required")
	case errors.As(err, &imapErr) && imapErr.Type == imap.StatusResponseTypeNo:
		writeError(w, http.StatusNotFound, "folder.not_found")
	default:
		slog.Error("imap request failed", "path", r.URL.Path, "err", err)
		writeError(w, http.StatusBadGateway, "mail.unavailable")
	}
}

func (a *authAPI) folders(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var folders []imapx.Folder
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		folders, err = imapx.ListFolders(c)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": folders})
}

func (a *authAPI) messages(w http.ResponseWriter, r *http.Request, s auth.Session) {
	q := r.URL.Query()
	folder := q.Get("folder")
	order := imapx.Sort{Key: q.Get("sort"), Desc: q.Get("order") != "asc"}
	if order.Key == "" {
		order.Key = "date"
	}
	page, okPage := intParam(q.Get("page"), 1)
	size, okSize := intParam(q.Get("size"), defaultPageSize)
	query, okQuery := parseQuery(q)
	if !validFolder(folder) || !okPage || !okSize || !okQuery || page < 1 || size < 1 || size > maxPageSize || !imapx.ValidSort(order.Key) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var p imapx.Page
	threaded, all := false, q.Get("scope") == "all" && !query.IsZero()
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		switch {
		case all:
			p, err = imapx.SearchAll(c, query, maxPageSize)
		case q.Get("threads") == "1" && imapx.CanThread(c):
			threaded = true
			p, err = imapx.ListThreads(c, folder, (page-1)*size, size, query, order)
		default:
			p, err = imapx.ListMessages(c, folder, (page-1)*size, size, query, order)
		}
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": p.Total, "page": page, "size": size, "messages": p.Messages, "threaded": threaded, "all": all})
}

// parseQuery reads the search fields; dates are whole days and the end date is inclusive.
func parseQuery(q url.Values) (imapx.Query, bool) {
	out := imapx.Query{Text: strings.TrimSpace(q.Get("q"))}
	if !validQuery(out.Text) {
		return out, false
	}
	flags := map[string]*bool{"body": &out.Body, "unread": &out.Unread, "flagged": &out.Flagged, "attachments": &out.Attachments}
	for name, dst := range flags {
		switch q.Get(name) {
		case "", "0":
		case "1":
			*dst = true
		default:
			return out, false
		}
	}
	for name, dst := range map[string]*time.Time{"since": &out.Since, "before": &out.Before} {
		v := q.Get(name)
		if v == "" {
			continue
		}
		d, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return out, false
		}
		*dst = d
	}
	if !out.Before.IsZero() {
		out.Before = out.Before.AddDate(0, 0, 1)
	}
	return out, true
}

func intParam(v string, def int) (int, bool) {
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil
}

func validQuery(q string) bool {
	if len(q) > 256 || !utf8.ValidString(q) {
		return false
	}
	for _, r := range q {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validFolder(name string) bool {
	if name == "" || len(name) > 512 || !utf8.ValidString(name) || imapx.IsHidden(name) {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
