// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/contacts"
	"github.com/stefanpejcic/golub/internal/imapx"
)

// A full 5000-contact book is well under this.
const maxBookBytes = 4 << 20

func loadBook(c *imapclient.Client) (contacts.Book, error) {
	data, err := imapx.LoadContacts(c)
	if err != nil {
		return contacts.Book{}, err
	}
	var b contacts.Book
	if len(data) > 0 {
		if err := json.Unmarshal(data, &b); err != nil {
			slog.Warn("stored contacts are unreadable, starting empty", "err", err)
			b = contacts.Book{}
		}
	}
	if b, err = contacts.Validate(b); err != nil {
		slog.Warn("stored contacts failed validation, starting empty", "err", err)
		b, _ = contacts.Validate(contacts.Book{})
	}
	return b, nil
}

func saveBook(c *imapclient.Client, b contacts.Book) error {
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return imapx.SaveContacts(c, data)
}

func (a *authAPI) getContacts(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var b contacts.Book
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		b, err = loadBook(c)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (a *authAPI) putContacts(w http.ResponseWriter, r *http.Request, s auth.Session) {
	var in contacts.Book
	if !decodeJSONLimit(w, r, &in, maxBookBytes) {
		return
	}
	b, err := contacts.Validate(in)
	if err != nil {
		writeError(w, http.StatusBadRequest, "contacts.invalid")
		return
	}
	if err := a.do(r, s, func(c *imapclient.Client) error { return saveBook(c, b) }); err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (a *authAPI) importContacts(w http.ResponseWriter, r *http.Request, s auth.Session) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBookBytes))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "request.too_large")
		return
	}
	cards, err := contacts.ParseVCards(bytes.NewReader(data))
	if err != nil || len(cards) == 0 {
		writeError(w, http.StatusBadRequest, "contacts.bad_vcard")
		return
	}
	var b contacts.Book
	var added, updated int
	err = a.do(r, s, func(c *imapclient.Client) error {
		cur, err := loadBook(c)
		if err != nil {
			return err
		}
		merged, ad, up := contacts.Merge(cur, cards)
		if b, err = contacts.Validate(merged); err != nil {
			return err
		}
		added, updated = ad, up
		return saveBook(c, b)
	})
	if errors.Is(err, contacts.ErrInvalid) {
		writeError(w, http.StatusBadRequest, "contacts.bad_vcard")
		return
	}
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added, "updated": updated, "book": b})
}

func (a *authAPI) exportContacts(w http.ResponseWriter, r *http.Request, s auth.Session) {
	group := r.URL.Query().Get("group")
	var b contacts.Book
	err := a.do(r, s, func(c *imapclient.Client) error {
		var err error
		b, err = loadBook(c)
		return err
	})
	if err != nil {
		a.mailError(w, r, err)
		return
	}
	list := b.Contacts
	if group != "" {
		list = nil
		for _, c := range b.Contacts {
			for _, g := range c.Groups {
				if strings.EqualFold(g, group) {
					list = append(list, c)
					break
				}
			}
		}
	}
	var out bytes.Buffer
	if err := contacts.WriteVCards(&out, list); err != nil {
		a.mailError(w, r, err)
		return
	}
	name := "contacts.vcf"
	if group != "" {
		name = group + ".vcf"
	}
	h := w.Header()
	h.Set("Content-Type", "text/vcard; charset=utf-8")
	h.Set("Content-Disposition", contentDisposition(name))
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	_, _ = w.Write(out.Bytes())
}

// collectRecipients remembers who the user wrote to, for autocomplete.
func collectRecipients(c *imapclient.Client, lists ...[]mail.Address) error {
	var rs []contacts.Recipient
	for _, l := range lists {
		for _, a := range l {
			rs = append(rs, contacts.Recipient{Name: a.Name, Email: a.Address})
		}
	}
	if len(rs) == 0 {
		return nil
	}
	b, err := loadBook(c)
	if err != nil {
		return err
	}
	return saveBook(c, contacts.Collect(b, rs))
}
