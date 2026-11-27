// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// CreateFolder creates and subscribes a folder.
func CreateFolder(c *imapclient.Client, name string) error {
	if err := c.Create(name, nil).Wait(); err != nil {
		return err
	}
	return c.Subscribe(name).Wait()
}

// RenameFolder renames a folder and moves its subscription along.
func RenameFolder(c *imapclient.Client, from, to string) error {
	if err := c.Rename(from, to, nil).Wait(); err != nil {
		return err
	}
	_ = c.Unsubscribe(from).Wait()
	return c.Subscribe(to).Wait()
}

// DeleteFolder removes a folder and its messages.
func DeleteFolder(c *imapclient.Client, name string) error {
	// Servers refuse to delete the selected folder; not having one selected is fine.
	_ = c.Unselect().Wait()
	_ = c.Unsubscribe(name).Wait()
	return c.Delete(name).Wait()
}

// EmptyFolder permanently removes every message in a folder.
func EmptyFolder(c *imapclient.Client, name string) error {
	sel, err := c.Select(name, nil).Wait()
	if err != nil || sel.NumMessages == 0 {
		return err
	}
	var all imap.SeqSet
	all.AddRange(1, 0)
	store := &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}
	if err := c.Store(all, store, nil).Close(); err != nil {
		return err
	}
	return c.Expunge().Close()
}

// SetSubscribed subscribes or unsubscribes a folder.
func SetSubscribed(c *imapclient.Client, name string, on bool) error {
	if on {
		return c.Subscribe(name).Wait()
	}
	return c.Unsubscribe(name).Wait()
}

// MarkAllRead sets \Seen on every message in folder.
func MarkAllRead(c *imapclient.Client, name string) error {
	sel, err := c.Select(name, nil).Wait()
	if err != nil || sel.NumMessages == 0 {
		return err
	}
	var all imap.SeqSet
	all.AddRange(1, 0)
	return c.Store(all, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen}}, nil).Close()
}
