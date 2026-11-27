// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"slices"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// SearchAll runs query in every folder and returns the newest limit matches, each tagged with its folder.
func SearchAll(c *imapclient.Client, query Query, limit int) (Page, error) {
	folders, err := ListFolders(c)
	if err != nil {
		return Page{}, err
	}
	page := Page{Messages: []Summary{}}
	for _, f := range folders {
		if !f.Selectable {
			continue
		}
		uids, err := newestMatches(c, f.Name, query)
		if err != nil {
			return Page{}, err
		}
		page.Total += len(uids)
		uids = uids[:min(len(uids), limit)]
		if len(uids) == 0 {
			continue
		}
		var set imap.UIDSet
		for _, u := range uids {
			set.AddNum(imap.UID(u))
		}
		msgs, err := c.Fetch(set, summaryFetch).Collect()
		if err != nil {
			return Page{}, err
		}
		for _, m := range msgs {
			s := summarize(m)
			s.Folder = f.Name
			page.Messages = append(page.Messages, s)
		}
	}
	slices.SortStableFunc(page.Messages, func(a, b Summary) int { return b.Date.Compare(a.Date) })
	page.Messages = page.Messages[:min(len(page.Messages), limit)]
	return page, nil
}

// newestMatches returns matching UIDs newest first.
func newestMatches(c *imapclient.Client, folder string, query Query) ([]uint32, error) {
	sel, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil || sel.NumMessages == 0 {
		return nil, err
	}
	if c.Caps().Has(imap.CapSort) {
		// One pass over every folder would push the user's list views out of the small per-connection cache.
		st, _ := stateOf(sel)
		return sortedUIDs(c, folder, query, Sort{Key: "date", Desc: true}, st, false)
	}
	res, err := c.UIDSearch(query.criteria(), nil).Wait()
	if err != nil {
		return nil, err
	}
	var out []uint32
	for _, u := range res.AllUIDs() {
		out = append(out, uint32(u))
	}
	slices.Reverse(out)
	return out, nil
}
