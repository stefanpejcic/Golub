// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/mbox"
)

// maxImport bounds one upload; bigger archives can be split or imported in parts.
const maxImport = 200 << 20

// transferTime replaces the server's short timeouts for a big upload or download.
const transferTime = time.Hour

var errEmptyImport = errors.New("import: no messages")

func (a *authAPI) importFolder(w http.ResponseWriter, r *http.Request, s auth.Session) {
	folder := r.URL.Query().Get("folder")
	if !validFolder(folder) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(transferTime))
	body := http.MaxBytesReader(w, r.Body, maxImport)
	imported := 0
	err := a.do(r, s, func(c *imapclient.Client) error {
		err := mbox.Read(body, func(msg []byte) error {
			if len(bytes.TrimSpace(msg)) == 0 {
				return nil
			}
			if err := imapx.ImportMessage(c, folder, msg); err != nil {
				return err
			}
			imported++
			return nil
		})
		if err == nil && imported == 0 {
			return errEmptyImport
		}
		return err
	})
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, mbox.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "import.too_large")
	case errors.Is(err, errEmptyImport):
		writeError(w, http.StatusBadRequest, "import.empty")
	case err != nil:
		// Messages before the failure stay imported, so say how many made it.
		slog.Warn("import stopped", "imported", imported, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "import.partial", "imported": imported})
	default:
		writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
	}
}

func (a *authAPI) exportFolder(w http.ResponseWriter, r *http.Request, s auth.Session) {
	folder := r.URL.Query().Get("folder")
	if !validFolder(folder) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(transferTime))
	started := false
	err := a.do(r, s, func(c *imapclient.Client) error {
		mw := mbox.NewWriter(w)
		return imapx.ExportFolder(c, folder, func(raw []byte, from string, date time.Time) error {
			if !started {
				h := w.Header()
				h.Set("Content-Type", "application/mbox")
				h.Set("Content-Disposition", contentDisposition(folder+".mbox"))
				h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
				started = true
			}
			return mw.Write(raw, from, date)
		})
	})
	switch {
	case err != nil && started:
		// Headers are gone; all that is left is to stop and log.
		slog.Warn("export stopped", "err", err)
	case err != nil:
		a.mailError(w, r, err)
	case !started:
		w.Header().Set("Content-Type", "application/mbox")
		w.Header().Set("Content-Disposition", contentDisposition(folder+".mbox"))
		w.WriteHeader(http.StatusOK)
	}
}
