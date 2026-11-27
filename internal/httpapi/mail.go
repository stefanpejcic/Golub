// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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
	query := strings.TrimSpace(q.Get("q"))
	order := imapx.Sort{Key: q.Get("sort"), Desc: q.Get("order") != "asc"}
	if order.Key == "" {
		order.Key = "date"
	}
	page, okPage := intParam(q.Get("page"), 1)
	size, okSize := intParam(q.Get("size"), defaultPageSize)
	if !validFolder(folder) || !okPage || !okSize || page < 1 || size < 1 || size > maxPageSize || !validQuery(query) || !imapx.ValidSort(order.Key) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var p imapx.Page
	threaded := false
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		if q.Get("threads") == "1" && imapx.CanThread(c) {
			threaded = true
			p, err = imapx.ListThreads(c, folder, (page-1)*size, size, query, order)
			return err
		}
		p, err = imapx.ListMessages(c, folder, (page-1)*size, size, query, order)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": p.Total, "page": page, "size": size, "messages": p.Messages, "threaded": threaded})
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
