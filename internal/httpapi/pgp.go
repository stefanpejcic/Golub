// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/keyring"
)

// 500 keys of up to 64 KB each still fit.
const maxKeyringBytes = 32 << 20

func (a *authAPI) getKeyring(w http.ResponseWriter, r *http.Request, s auth.Session) {
	ring := keyring.Ring{Keys: []keyring.Key{}}
	err := a.do(r, s, func(c *imapclient.Client) error {
		data, err := imapx.LoadKeys(c)
		if err != nil || len(data) == 0 {
			return err
		}
		var stored keyring.Ring
		if err := json.Unmarshal(data, &stored); err != nil {
			slog.Warn("stored keyring is unreadable, starting empty", "err", err)
			return nil
		}
		if v, err := keyring.Validate(stored); err == nil {
			ring = v
		}
		return nil
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ring)
}

func (a *authAPI) putKeyring(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var in keyring.Ring
	if !decodeJSONLimit(w, r, &in, maxKeyringBytes) {
		return
	}
	ring, err := keyring.Validate(in)
	if err != nil {
		writeError(w, http.StatusBadRequest, "pgp.invalid_key")
		return
	}
	data, err := json.Marshal(ring)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server.error")
		return
	}
	if err := a.do(r, s, func(c *imapclient.Client) error { return imapx.SaveKeys(c, data) }); err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ring)
}
