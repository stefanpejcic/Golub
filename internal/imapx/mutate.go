// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func uidSetOf(uids []uint32) imap.UIDSet {
	var s imap.UIDSet
	for _, u := range uids {
		s.AddNum(imap.UID(u))
	}
	return s
}

// SetFlags adds or removes a flag on several messages.
func SetFlags(c *imapclient.Client, folder string, uids []uint32, flag imap.Flag, on bool) error {
	if _, err := c.Select(folder, nil).Wait(); err != nil {
		return err
	}
	op := imap.StoreFlagsDel
	if on {
		op = imap.StoreFlagsAdd
	}
	return c.Store(uidSetOf(uids), &imap.StoreFlags{Op: op, Silent: true, Flags: []imap.Flag{flag}}, nil).Close()
}

// Move moves messages to dest, emulating MOVE with COPY and EXPUNGE on old servers.
func Move(c *imapclient.Client, folder string, uids []uint32, dest string) error {
	if _, err := c.Select(folder, nil).Wait(); err != nil {
		return err
	}
	set := uidSetOf(uids)
	if c.Caps().Has(imap.CapMove) {
		_, err := c.Move(set, dest).Wait()
		return err
	}
	if _, err := c.Copy(set, dest).Wait(); err != nil {
		return err
	}
	return expungeSelected(c, set)
}

// Expunge permanently removes messages.
func Expunge(c *imapclient.Client, folder string, uids []uint32) error {
	if _, err := c.Select(folder, nil).Wait(); err != nil {
		return err
	}
	return expungeSelected(c, uidSetOf(uids))
}

func expungeSelected(c *imapclient.Client, set imap.UIDSet) error {
	store := &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}
	if err := c.Store(set, store, nil).Close(); err != nil {
		return err
	}
	// Without UIDPLUS a plain EXPUNGE would also remove mail another client marked \Deleted.
	if c.Caps().Has(imap.CapUIDPlus) || c.Caps().Has(imap.CapIMAP4rev2) {
		return c.UIDExpunge(set).Close()
	}
	return c.Expunge().Close()
}

// SpecialFolder finds a folder by special-use role, falling back to a conventional name.
func SpecialFolder(folders []Folder, special, fallback string) string {
	for _, f := range folders {
		if f.Special == special && f.Selectable {
			return f.Name
		}
	}
	for _, f := range folders {
		if strings.EqualFold(f.Name, fallback) && f.Selectable {
			return f.Name
		}
	}
	return ""
}

var specialToAttr = map[string]imap.MailboxAttr{
	"archive": imap.MailboxAttrArchive, "drafts": imap.MailboxAttrDrafts, "sent": imap.MailboxAttrSent,
	"junk": imap.MailboxAttrJunk, "trash": imap.MailboxAttrTrash,
}

// EnsureFolder creates a folder if it does not exist yet, tagging its special use when the server allows it.
func EnsureFolder(c *imapclient.Client, name, special string) error {
	var opts *imap.CreateOptions
	if attr, ok := specialToAttr[special]; ok && c.Caps().Has(imap.CapCreateSpecialUse) {
		opts = &imap.CreateOptions{SpecialUse: []imap.MailboxAttr{attr}}
	}
	err := c.Create(name, opts).Wait()
	var imapErr *imap.Error
	if errors.As(err, &imapErr) && imapErr.Code == imap.ResponseCodeAlreadyExists {
		return nil
	}
	if err != nil {
		return err
	}
	_ = c.Subscribe(name).Wait()
	return nil
}

// Flags set on originals after replying or forwarding.
const (
	FlagAnswered  = imap.FlagAnswered
	FlagDraft     = imap.FlagDraft
	FlagForwarded = imap.Flag("$Forwarded")
)

// AppendMessage stores raw in folder and returns its UID when the server reports one.
func AppendMessage(c *imapclient.Client, folder string, raw []byte, flags ...imap.Flag) (uint32, error) {
	cmd := c.Append(folder, int64(len(raw)), &imap.AppendOptions{Flags: flags})
	if _, err := cmd.Write(raw); err != nil {
		_ = cmd.Close()
		return 0, err
	}
	if err := cmd.Close(); err != nil {
		return 0, err
	}
	data, err := cmd.Wait()
	if err != nil {
		return 0, err
	}
	return uint32(data.UID), nil
}
