// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/compose"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/mime"
	"github.com/stefanpejcic/golub/internal/settings"
)

const maxBatch = 1000

type batch struct {
	Folder string   `json:"folder"`
	UIDs   []uint32 `json:"uids"`
}

func (b batch) valid() bool {
	if !validFolder(b.Folder) || len(b.UIDs) == 0 || len(b.UIDs) > maxBatch {
		return false
	}
	for _, u := range b.UIDs {
		if u == 0 {
			return false
		}
	}
	return true
}

var flagNames = map[string]imap.Flag{"seen": imapx.FlagSeen, "flagged": imapx.FlagFlagged}

func (a *authAPI) setFlag(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var req struct {
		batch
		Flag  string `json:"flag"`
		Value bool   `json:"value"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	flag, ok := flagNames[req.Flag]
	if !req.valid() || !ok {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	err := a.do(r, s, func(c *imapclient.Client) error {
		return imapx.SetFlags(c, req.Folder, req.UIDs, flag, req.Value)
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// moved tells the client where messages went, so it can offer Undo.
type moved struct {
	Folder  string   `json:"folder"`
	UIDs    []uint32 `json:"uids"`
	Trashed bool     `json:"trashed"`
}

func (a *authAPI) copyTo(w http.ResponseWriter, r *http.Request, s auth.Session) {
	a.transfer(w, r, s, false)
}

func (a *authAPI) move(w http.ResponseWriter, r *http.Request, s auth.Session) {
	a.transfer(w, r, s, true)
}

func (a *authAPI) transfer(w http.ResponseWriter, r *http.Request, s auth.Session, isMove bool) {
	var req struct {
		batch
		To string `json:"to"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !req.valid() || !validFolder(req.To) || req.To == req.Folder {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var dest []uint32
	err := a.do(r, s, func(c *imapclient.Client) error {
		if !isMove {
			return imapx.Copy(c, req.Folder, req.UIDs, req.To)
		}
		var err error
		dest, err = imapx.MoveUIDs(c, req.Folder, req.UIDs, req.To)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	if !isMove {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, moved{Folder: req.To, UIDs: orEmpty(dest)})
}

func orEmpty(u []uint32) []uint32 {
	if u == nil {
		return []uint32{}
	}
	return u
}

// remove moves mail to Trash, or deletes it for good when it is already there.
func (a *authAPI) remove(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var req batch
	if !decodeJSON(w, r, &req) {
		return
	}
	if !req.valid() {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	trashed := false
	var trash string
	var dest []uint32
	err := a.do(r, s, func(c *imapclient.Client) error {
		st, err := loadSettings(c, s.Email)
		if err != nil {
			return err
		}
		folders, err := imapx.ListFolders(c)
		if err != nil {
			return err
		}
		trash = specialFolder(st, folders, "trash", "Trash")
		if st.DeleteMode == "permanent" || trash == "" || trash == req.Folder {
			return imapx.Expunge(c, req.Folder, req.UIDs)
		}
		trashed = true
		dest, err = imapx.MoveUIDs(c, req.Folder, req.UIDs, trash)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	if !trashed {
		trash = ""
	}
	writeJSON(w, http.StatusOK, moved{Folder: trash, UIDs: orEmpty(dest), Trashed: trashed})
}

// toSpecial moves mail to a role folder (archive, junk), creating it on first use.
func (a *authAPI) toSpecial(role, fallback string) func(http.ResponseWriter, *http.Request, auth.Session) {
	return func(w http.ResponseWriter, r *http.Request, s auth.Session) {
		var req batch
		if !decodeJSON(w, r, &req) {
			return
		}
		if !req.valid() {
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		}
		var folder string
		var dest []uint32
		err := a.do(r, s, func(c *imapclient.Client) error {
			st, err := loadSettings(c, s.Email)
			if err != nil {
				return err
			}
			folders, err := imapx.ListFolders(c)
			if err != nil {
				return err
			}
			if folder = specialFolder(st, folders, role, fallback); folder == "" {
				folder = fallback
				if err := imapx.EnsureFolder(c, folder, role); err != nil {
					return err
				}
			}
			if folder == req.Folder {
				return errSameFolder
			}
			dest, err = imapx.MoveUIDs(c, req.Folder, req.UIDs, folder)
			return err
		})
		if errors.Is(err, errSameFolder) {
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		}
		if err != nil {
			a.mailError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, moved{Folder: folder, UIDs: orEmpty(dest)})
	}
}

var errSameFolder = errors.New("source and destination are the same folder")

type messageRequest struct {
	Folder string `json:"folder"`
	UID    uint32 `json:"uid"`
	Send   bool   `json:"send"`
}

func (q messageRequest) ref() (messageRef, bool) {
	return messageRef{folder: q.Folder, uid: q.UID}, q.UID != 0 && validFolder(q.Folder)
}

// unsubscribe runs the one-click unsubscribe the message offers; other kinds are handled in the browser.
func (a *authAPI) unsubscribe(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var req messageRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ref, ok := req.ref()
	if !ok || a.o.Proxy == nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	m, _, ok := a.load(w, r, s, ref, false)
	if !ok {
		return
	}
	if !m.Unsubscribe.OneClick {
		writeError(w, http.StatusBadRequest, "unsubscribe.not_one_click")
		return
	}
	if err := a.o.Proxy.Unsubscribe(r.Context(), m.Unsubscribe.URL); err != nil {
		slog.Warn("one-click unsubscribe failed", "err", err)
		writeError(w, http.StatusBadGateway, "unsubscribe.failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// receipt sends or declines a requested read receipt; either way it is not asked again.
func (a *authAPI) receipt(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var req messageRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ref, ok := req.ref()
	if !ok {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	m, _, ok := a.load(w, r, s, ref, false)
	if !ok {
		return
	}
	if m.ReceiptTo == "" {
		writeError(w, http.StatusBadRequest, "receipt.not_requested")
		return
	}
	if req.Send {
		var from mail.Address
		err := a.do(r, s, func(c *imapclient.Client) error {
			st, err := loadSettings(c, s.Email)
			if err != nil {
				return err
			}
			from = receiptIdentity(st, m, s.Email)
			return nil
		})
		if err != nil {
			a.mailError(w, r, err)
			return
		}
		raw, err := compose.BuildMDN(compose.MDN{
			From: from, To: mail.Address{Address: m.ReceiptTo},
			OriginalSubject: m.Subject, OriginalMessageID: m.MessageID,
			Text: fmt.Sprintf("Your message %q was displayed.", m.Subject),
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		}
		user, pass := a.imapCreds(s)
		if err := a.o.SMTP.Send(r.Context(), user, pass, from.Address, []string{m.ReceiptTo}, raw); err != nil {
			a.writeComposeError(w, r, err)
			return
		}
	}
	err := a.do(r, s, func(c *imapclient.Client) error {
		return imapx.SetFlag(c, ref.folder, ref.uid, imapx.FlagMDNSent, true)
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// receiptIdentity answers from the identity the message was addressed to.
func receiptIdentity(st settings.Settings, m *mime.Message, fallback string) mail.Address {
	for _, id := range st.Identities {
		for _, to := range append(append([]mime.Address{}, m.To...), m.Cc...) {
			if strings.EqualFold(id.Email, to.Address) {
				return mail.Address{Name: id.Name, Address: id.Email}
			}
		}
	}
	for _, id := range st.Identities {
		if id.Default {
			return mail.Address{Name: id.Name, Address: id.Email}
		}
	}
	return mail.Address{Address: fallback}
}
