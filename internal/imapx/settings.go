// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// SettingsFolder holds the settings JSON; it stays unsubscribed so other clients do not show it.
const SettingsFolder = "golub-settings"

func isNonexistent(err error) bool {
	var imapErr *imap.Error
	return errors.As(err, &imapErr) && (imapErr.Code == imap.ResponseCodeNonExistent || imapErr.Type == imap.StatusResponseTypeNo)
}

// ContactsFolder holds the address book JSON, hidden the same way.
const ContactsFolder = "golub-contacts"

// KeysFolder holds the PGP keyring; private keys in it are always passphrase protected by the browser.
const KeysFolder = "golub-pgp"

// IsHidden reports folders Golub keeps for itself and never shows or lets the user touch.
func IsHidden(name string) bool {
	return name == SettingsFolder || name == ContactsFolder || name == KeysFolder
}

// LoadKeys returns the stored keyring JSON, or nil when there is none yet.
func LoadKeys(c *imapclient.Client) ([]byte, error) { return loadDoc(c, KeysFolder) }

// SaveKeys replaces the stored keyring with data.
func SaveKeys(c *imapclient.Client, data []byte) error { return saveDoc(c, KeysFolder, "keys", data) }

// LoadSettings returns the stored settings JSON, or nil when none were saved yet.
func LoadSettings(c *imapclient.Client) ([]byte, error) { return loadDoc(c, SettingsFolder) }

// SaveSettings replaces the stored settings with data.
func SaveSettings(c *imapclient.Client, data []byte) error {
	return saveDoc(c, SettingsFolder, "settings", data)
}

// LoadContacts returns the stored address book JSON, or nil when there is none yet.
func LoadContacts(c *imapclient.Client) ([]byte, error) { return loadDoc(c, ContactsFolder) }

// SaveContacts replaces the stored address book with data.
func SaveContacts(c *imapclient.Client, data []byte) error {
	return saveDoc(c, ContactsFolder, "contacts", data)
}

func loadDoc(c *imapclient.Client, folder string) ([]byte, error) {
	sel, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if isNonexistent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if sel.NumMessages == 0 {
		return nil, nil
	}
	var set imap.SeqSet
	set.AddNum(sel.NumMessages)
	section := &imap.FetchItemBodySection{Specifier: imap.PartSpecifierText, Peek: true}
	msgs, err := c.Fetch(set, &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{section}}).Collect()
	if err != nil || len(msgs) == 0 {
		return nil, err
	}
	return bytes.TrimRight(msgs[0].FindBodySection(section), "\r\n"), nil
}

// saveDoc appends the new version first and only then expunges the old ones, so a failure never loses data.
func saveDoc(c *imapclient.Client, folder, kind string, data []byte) error {
	if err := ensureHidden(c, folder); err != nil {
		return err
	}
	sel, err := c.Select(folder, nil).Wait()
	if err != nil {
		return err
	}
	old := sel.NumMessages

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: Golub <%s@golub.invalid>\r\nSubject: Golub %s\r\nDate: %s\r\n", kind, kind, time.Now().Format(time.RFC1123Z))
	msg.WriteString("X-Golub-Data: " + kind + "\r\nMIME-Version: 1.0\r\nContent-Type: application/json; charset=utf-8\r\n\r\n")
	msg.Write(data)
	msg.WriteString("\r\n")

	cmd := c.Append(folder, int64(msg.Len()), &imap.AppendOptions{Flags: []imap.Flag{imap.FlagSeen}})
	if _, err := cmd.Write(msg.Bytes()); err != nil {
		return err
	}
	if err := cmd.Close(); err != nil {
		return err
	}
	if _, err := cmd.Wait(); err != nil {
		return err
	}
	if old == 0 {
		return nil
	}
	var set imap.SeqSet
	set.AddRange(1, old)
	store := &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}
	if err := c.Store(set, store, nil).Close(); err != nil {
		return err
	}
	return c.Expunge().Close()
}

// ensureHidden creates a folder without subscribing to it.
func ensureHidden(c *imapclient.Client, name string) error {
	err := c.Create(name, nil).Wait()
	var imapErr *imap.Error
	if errors.As(err, &imapErr) && imapErr.Code == imap.ResponseCodeAlreadyExists {
		return nil
	}
	return err
}
