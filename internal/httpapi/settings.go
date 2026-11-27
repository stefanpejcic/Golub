// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/settings"
)

// loadSettings reads stored settings; a broken store falls back to defaults rather than failing.
func loadSettings(c *imapclient.Client, email string) (settings.Settings, error) {
	data, err := imapx.LoadSettings(c)
	if err != nil {
		return settings.Settings{}, err
	}
	s, err := settings.Decode(data, email)
	if err != nil {
		slog.Warn("stored settings unreadable, using defaults", "err", err)
	}
	return s, nil
}

func (a *authAPI) getSettings(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var st settings.Settings
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		st, err = loadSettings(c, s.Email)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (a *authAPI) putSettings(w http.ResponseWriter, r *http.Request, s auth.Session) {
	st := settings.Defaults(s.Email)
	if !decodeJSON(w, r, &st) {
		return
	}
	if st.Responses == nil {
		st.Responses = []settings.Response{}
	}
	if err := st.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "settings.invalid", "field": err.Error()})
		return
	}
	data, err := settings.Encode(st)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server.error")
		return
	}
	if err := a.do(r, s, func(c *imapclient.Client) error { return imapx.SaveSettings(c, data) }); err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// specialFolder resolves a role, honoring the user's override before SPECIAL-USE and names.
func specialFolder(st settings.Settings, folders []imapx.Folder, role, fallback string) string {
	if name := st.SpecialFolders[role]; name != "" {
		for _, f := range folders {
			if f.Name == name && f.Selectable {
				return name
			}
		}
	}
	return imapx.SpecialFolder(folders, role, fallback)
}

type folderRequest struct {
	Name  string `json:"name"`
	To    string `json:"to"`
	Value bool   `json:"value"`
}

func (a *authAPI) folderOp(op string) func(http.ResponseWriter, *http.Request, auth.Session) {
	return func(w http.ResponseWriter, r *http.Request, s auth.Session) {
		var req folderRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		protected := strings.EqualFold(req.Name, "INBOX") || req.Name == imapx.SettingsFolder
		switch {
		case !validFolder(req.Name), req.Name == imapx.SettingsFolder:
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		case (op == "rename" || op == "delete") && protected:
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		case op == "rename" && (!validFolder(req.To) || req.To == imapx.SettingsFolder):
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		}
		err := a.do(r, s, func(c *imapclient.Client) error {
			switch op {
			case "create":
				return imapx.CreateFolder(c, req.Name)
			case "rename":
				return imapx.RenameFolder(c, req.Name, req.To)
			case "delete":
				return imapx.DeleteFolder(c, req.Name)
			case "empty":
				return imapx.EmptyFolder(c, req.Name)
			default:
				return imapx.SetSubscribed(c, req.Name, req.Value)
			}
		})
		if err != nil {
			a.mailError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *authAPI) quota(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var q *imapx.QuotaInfo
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		q, err = imapx.Quota(c)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	if q == nil {
		writeJSON(w, http.StatusOK, map[string]bool{"supported": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"supported": true, "used": q.Used, "limit": q.Limit})
}
