// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"errors"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Message errors.
var (
	ErrNotFound = errors.New("imap: message not found")
	ErrTooLarge = errors.New("imap: message too large")
)

// Flags the UI can toggle.
const (
	FlagSeen    = imap.FlagSeen
	FlagFlagged = imap.FlagFlagged
)

// Raw is a full message with its state.
type Raw struct {
	UID     uint32
	Data    []byte
	Seen    bool
	Flagged bool
}

// FetchRaw downloads a message without touching its \Seen flag.
func FetchRaw(c *imapclient.Client, folder string, uid uint32, maxSize int64) (Raw, error) {
	if _, err := c.Select(folder, nil).Wait(); err != nil {
		return Raw{}, err
	}
	set := uidSetOf([]uint32{uid})
	meta, err := c.Fetch(set, &imap.FetchOptions{UID: true, Flags: true, RFC822Size: true}).Collect()
	if err != nil {
		return Raw{}, err
	}
	if len(meta) == 0 {
		return Raw{}, ErrNotFound
	}
	if meta[0].RFC822Size > maxSize {
		return Raw{}, fmt.Errorf("%w: %d bytes", ErrTooLarge, meta[0].RFC822Size)
	}
	section := &imap.FetchItemBodySection{Peek: true}
	msgs, err := c.Fetch(set, &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{section}}).Collect()
	if err != nil {
		return Raw{}, err
	}
	if len(msgs) == 0 {
		return Raw{}, ErrNotFound
	}
	r := Raw{UID: uid, Data: msgs[0].FindBodySection(section)}
	for _, f := range meta[0].Flags {
		switch f {
		case imap.FlagSeen:
			r.Seen = true
		case imap.FlagFlagged:
			r.Flagged = true
		}
	}
	return r, nil
}

// SetFlag adds or removes a flag on one message.
func SetFlag(c *imapclient.Client, folder string, uid uint32, flag imap.Flag, on bool) error {
	return SetFlags(c, folder, []uint32{uid}, flag, on)
}
