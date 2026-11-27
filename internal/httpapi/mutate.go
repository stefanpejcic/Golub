// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
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
		st, err := loadSettings(c, s.Email)
		if err != nil {
			return err
		}
		folders, err := imapx.ListFolders(c)
		if err != nil {
			return err
		}
		trash := specialFolder(st, folders, "trash", "Trash")
		if st.DeleteMode == "permanent" || trash == "" || trash == req.Folder {
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

// archive moves mail to the Archive folder, creating it on first use.
func (a *authAPI) archive(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var req batch
	if !decodeJSON(w, r, &req) {
		return
	}
	if !req.valid() {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var dest string
	err := a.do(r, s, func(c *imapclient.Client) error {
		st, err := loadSettings(c, s.Email)
		if err != nil {
			return err
		}
		folders, err := imapx.ListFolders(c)
		if err != nil {
			return err
		}
		if dest = specialFolder(st, folders, "archive", "Archive"); dest == "" {
			dest = "Archive"
			if err := imapx.EnsureFolder(c, dest, "archive"); err != nil {
				return err
			}
		}
		if dest == req.Folder {
			return errSameFolder
		}
		return imapx.Move(c, req.Folder, req.UIDs, dest)
	})
	if errors.Is(err, errSameFolder) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"folder": dest})
}

var errSameFolder = errors.New("source and destination are the same folder")
