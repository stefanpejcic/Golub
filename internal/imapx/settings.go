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

// SettingsFolder holds one message with the user's settings as JSON. It is created
// unsubscribed so other mail clients do not show it.
const SettingsFolder = "golub-settings"

func isNonexistent(err error) bool {
	var imapErr *imap.Error
	return errors.As(err, &imapErr) && (imapErr.Code == imap.ResponseCodeNonExistent || imapErr.Type == imap.StatusResponseTypeNo)
}

// LoadSettings returns the stored settings JSON, or nil when none were saved yet.
func LoadSettings(c *imapclient.Client) ([]byte, error) {
	sel, err := c.Select(SettingsFolder, &imap.SelectOptions{ReadOnly: true}).Wait()
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

// SaveSettings replaces the stored settings with data.
func SaveSettings(c *imapclient.Client, data []byte) error {
	if err := ensureHidden(c, SettingsFolder); err != nil {
		return err
	}
	sel, err := c.Select(SettingsFolder, nil).Wait()
	if err != nil {
		return err
	}
	old := sel.NumMessages

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: Golub <settings@golub.invalid>\r\nSubject: Golub settings\r\nDate: %s\r\n", time.Now().Format(time.RFC1123Z))
	msg.WriteString("X-Golub-Settings: 1\r\nMIME-Version: 1.0\r\nContent-Type: application/json; charset=utf-8\r\n\r\n")
	msg.Write(data)
	msg.WriteString("\r\n")

	cmd := c.Append(SettingsFolder, int64(msg.Len()), &imap.AppendOptions{Flags: []imap.Flag{imap.FlagSeen}})
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
