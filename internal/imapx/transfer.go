// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"bytes"
	"net/mail"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Export fetches this many messages per round trip, which keeps memory flat on big folders.
const exportBatch = 50

// ImportMessage appends an imported message as read, dated by its own Date header when it has one.
func ImportMessage(c *imapclient.Client, folder string, raw []byte) error {
	opts := &imap.AppendOptions{Flags: []imap.Flag{imap.FlagSeen}}
	if m, err := mail.ReadMessage(bytes.NewReader(raw)); err == nil {
		if d, err := m.Header.Date(); err == nil {
			opts.Time = d
		}
	}
	cmd := c.Append(folder, int64(len(raw)), opts)
	if _, err := cmd.Write(raw); err != nil {
		_ = cmd.Close()
		return err
	}
	if err := cmd.Close(); err != nil {
		return err
	}
	_, err := cmd.Wait()
	return err
}

// ExportFolder calls fn with every message in folder, oldest first.
func ExportFolder(c *imapclient.Client, folder string, fn func(raw []byte, from string, date time.Time) error) error {
	if _, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return err
	}
	res, err := c.UIDSearch(&imap.SearchCriteria{}, nil).Wait()
	if err != nil {
		return err
	}
	uids := res.AllUIDs()
	body := &imap.FetchItemBodySection{Peek: true}
	opts := &imap.FetchOptions{UID: true, Envelope: true, InternalDate: true, BodySection: []*imap.FetchItemBodySection{body}}
	for start := 0; start < len(uids); start += exportBatch {
		var set imap.UIDSet
		set.AddNum(uids[start:min(start+exportBatch, len(uids))]...)
		msgs, err := c.Fetch(set, opts).Collect()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			from := ""
			if m.Envelope != nil && len(m.Envelope.From) > 0 {
				from = m.Envelope.From[0].Addr()
			}
			if err := fn(m.FindBodySection(body), from, m.InternalDate); err != nil {
				return err
			}
		}
	}
	return nil
}
