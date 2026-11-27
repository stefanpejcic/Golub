// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"net/http"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
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

func (a *authAPI) move(w http.ResponseWriter, r *http.Request, s auth.Session) {
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
	err := a.do(r, s, func(c *imapclient.Client) error {
		return imapx.Move(c, req.Folder, req.UIDs, req.To)
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	err := a.do(r, s, func(c *imapclient.Client) error {
		folders, err := imapx.ListFolders(c)
		if err != nil {
			return err
		}
		trash := imapx.SpecialFolder(folders, "trash", "Trash")
		if trash == "" || trash == req.Folder {
			return imapx.Expunge(c, req.Folder, req.UIDs)
		}
		trashed = true
		return imapx.Move(c, req.Folder, req.UIDs, trash)
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"trashed": trashed})
}
