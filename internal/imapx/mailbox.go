// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"fmt"
	"io"
	"mime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"golang.org/x/text/encoding/htmlindex"
)

// Folder is a mailbox with its counters and special-use role.
type Folder struct {
	Name       string `json:"name"`
	Delim      string `json:"delim"`
	Special    string `json:"special,omitempty"`
	Unread     uint32 `json:"unread"`
	Total      uint32 `json:"total"`
	Selectable bool   `json:"selectable"`
}

// Address is a display name and email.
type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// Summary is one row of the message list.
type Summary struct {
	UID         uint32    `json:"uid"`
	Subject     string    `json:"subject"`
	From        []Address `json:"from"`
	To          []Address `json:"to"`
	Date        time.Time `json:"date"`
	Seen        bool      `json:"seen"`
	Flagged     bool      `json:"flagged"`
	Answered    bool      `json:"answered"`
	Attachments bool      `json:"attachments"`
	Size        int64     `json:"size"`
}

// Page is a slice of a sorted folder.
type Page struct {
	Total    int       `json:"total"`
	Messages []Summary `json:"messages"`
}

var specialOrder = []string{"inbox", "drafts", "sent", "archive", "junk", "trash"}

var specialAttrs = map[imap.MailboxAttr]string{
	imap.MailboxAttrDrafts:  "drafts",
	imap.MailboxAttrSent:    "sent",
	imap.MailboxAttrArchive: "archive",
	imap.MailboxAttrJunk:    "junk",
	imap.MailboxAttrTrash:   "trash",
}

// wordDecoder decodes RFC 2047 headers in any charset the htmlindex knows.
func wordDecoder() *mime.WordDecoder {
	return &mime.WordDecoder{CharsetReader: func(charset string, r io.Reader) (io.Reader, error) {
		enc, err := htmlindex.Get(charset)
		if err != nil {
			return nil, fmt.Errorf("unknown charset %q", charset)
		}
		return enc.NewDecoder().Reader(r), nil
	}}
}

// ListFolders returns all folders, special ones first, with message counts.
func ListFolders(c *imapclient.Client) ([]Folder, error) {
	caps := c.Caps()
	status := &imap.StatusOptions{NumMessages: true, NumUnseen: true}
	opts := &imap.ListOptions{ReturnSpecialUse: caps.Has(imap.CapSpecialUse)}
	listStatus := caps.Has(imap.CapListStatus) || caps.Has(imap.CapIMAP4rev2)
	if listStatus {
		opts.ReturnStatus = status
	}
	list, err := c.List("", "*", opts).Collect()
	if err != nil {
		return nil, err
	}

	folders := make([]Folder, 0, len(list))
	for _, l := range list {
		f := classify(l)
		st := l.Status
		if st == nil && f.Selectable {
			if st, err = c.Status(l.Mailbox, status).Wait(); err != nil {
				return nil, err
			}
		}
		if st != nil {
			f.Total = deref(st.NumMessages)
			f.Unread = deref(st.NumUnseen)
		}
		folders = append(folders, f)
	}

	rank := func(f Folder) int {
		if i := slices.Index(specialOrder, f.Special); i >= 0 {
			return i
		}
		return len(specialOrder)
	}
	sort.SliceStable(folders, func(i, j int) bool {
		ri, rj := rank(folders[i]), rank(folders[j])
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})
	return folders, nil
}

// classify derives name, role and selectability from a LIST response.
func classify(l *imap.ListData) Folder {
	f := Folder{Name: l.Mailbox, Selectable: true}
	if l.Delim != 0 {
		f.Delim = string(l.Delim)
	}
	if strings.EqualFold(l.Mailbox, "INBOX") {
		f.Name, f.Special = "INBOX", "inbox"
	}
	for _, a := range l.Attrs {
		if s, ok := specialAttrs[a]; ok && f.Special == "" {
			f.Special = s
		}
		if a == imap.MailboxAttrNoSelect || a == imap.MailboxAttrNonExistent {
			f.Selectable = false
		}
	}
	return f
}

func deref(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}

// searchCriteria matches query against subject, sender and recipients; empty matches all.
func searchCriteria(query string) *imap.SearchCriteria {
	if query == "" {
		return &imap.SearchCriteria{}
	}
	h := func(key string) imap.SearchCriteria {
		return imap.SearchCriteria{Header: []imap.SearchCriteriaHeaderField{{Key: key, Value: query}}}
	}
	return &imap.SearchCriteria{Or: [][2]imap.SearchCriteria{{
		h("Subject"),
		{Or: [][2]imap.SearchCriteria{{h("From"), {Or: [][2]imap.SearchCriteria{{h("To"), h("Cc")}}}}}},
	}}}
}

// ListMessages returns messages newest first, using server-side SORT when available.
func ListMessages(c *imapclient.Client, folder string, offset, limit int, query string) (Page, error) {
	sel, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return Page{}, err
	}
	total := int(sel.NumMessages)
	page := Page{Total: total, Messages: []Summary{}}
	if total == 0 || offset >= total || limit <= 0 {
		return page, nil
	}
	end := min(offset+limit, total)

	fetch := &imap.FetchOptions{UID: true, Envelope: true, Flags: true, InternalDate: true, RFC822Size: true, BodyStructure: &imap.FetchItemBodyStructure{}}
	var msgs []*imapclient.FetchMessageBuffer
	var order []uint32

	if c.Caps().Has(imap.CapSort) {
		uids, err := c.UIDSort(&imapclient.SortOptions{
			SearchCriteria: searchCriteria(query),
			SortCriteria:   []imapclient.SortCriterion{{Key: imapclient.SortKeyDate, Reverse: true}},
		}).Wait()
		if err != nil {
			return Page{}, err
		}
		page.Total = len(uids)
		if offset >= len(uids) {
			return page, nil
		}
		order = uids[offset:min(end, len(uids))]
		var set imap.UIDSet
		for _, u := range order {
			set.AddNum(imap.UID(u))
		}
		if msgs, err = c.Fetch(set, fetch).Collect(); err != nil {
			return Page{}, err
		}
	} else if query != "" {
		res, err := c.UIDSearch(searchCriteria(query), nil).Wait()
		if err != nil {
			return Page{}, err
		}
		all := res.AllUIDs()
		slices.Reverse(all)
		page.Total = len(all)
		if offset >= len(all) {
			return page, nil
		}
		for _, u := range all[offset:min(offset+limit, len(all))] {
			order = append(order, uint32(u))
		}
		var set imap.UIDSet
		for _, u := range order {
			set.AddNum(imap.UID(u))
		}
		if msgs, err = c.Fetch(set, fetch).Collect(); err != nil {
			return Page{}, err
		}
	} else {
		// Without SORT, arrival order is the best cheap approximation of date order.
		var set imap.SeqSet
		set.AddRange(uint32(total-end+1), uint32(total-offset)) //nolint:gosec // 0 <= offset < end <= total, which came from a uint32
		if msgs, err = c.Fetch(set, fetch).Collect(); err != nil {
			return Page{}, err
		}
		sort.Slice(msgs, func(i, j int) bool { return msgs[i].SeqNum > msgs[j].SeqNum })
	}

	byUID := make(map[uint32]Summary, len(msgs))
	for _, m := range msgs {
		s := summarize(m)
		if order == nil {
			page.Messages = append(page.Messages, s)
		} else {
			byUID[s.UID] = s
		}
	}
	for _, u := range order {
		if s, ok := byUID[u]; ok {
			page.Messages = append(page.Messages, s)
		}
	}
	return page, nil
}

func summarize(m *imapclient.FetchMessageBuffer) Summary {
	s := Summary{UID: uint32(m.UID), Size: m.RFC822Size, Date: m.InternalDate, From: []Address{}, To: []Address{}}
	if e := m.Envelope; e != nil {
		s.Subject = e.Subject
		s.From = addresses(e.From)
		s.To = addresses(e.To)
		if !e.Date.IsZero() {
			s.Date = e.Date
		}
	}
	for _, f := range m.Flags {
		switch f {
		case imap.FlagSeen:
			s.Seen = true
		case imap.FlagFlagged:
			s.Flagged = true
		case imap.FlagAnswered:
			s.Answered = true
		}
	}
	s.Attachments = hasAttachment(m.BodyStructure)
	return s
}

func addresses(in []imap.Address) []Address {
	out := make([]Address, 0, len(in))
	for _, a := range in {
		if a.IsGroupStart() || a.IsGroupEnd() {
			continue
		}
		out = append(out, Address{Name: a.Name, Address: a.Addr()})
	}
	return out
}

func hasAttachment(bs imap.BodyStructure) bool {
	found := false
	if bs == nil {
		return false
	}
	bs.Walk(func(_ []int, part imap.BodyStructure) bool {
		if sp, ok := part.(*imap.BodyStructureSinglePart); ok {
			if d := sp.Disposition(); d != nil && strings.EqualFold(d.Value, "attachment") {
				found = true
			}
			if sp.Filename() != "" && !strings.EqualFold(sp.Type, "text") && sp.Disposition() == nil {
				found = true
			}
		}
		return !found
	})
	return found
}
