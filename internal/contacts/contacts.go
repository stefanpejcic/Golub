// SPDX-License-Identifier: GPL-3.0-or-later

// Package contacts holds the address book, collected recipients and vCard conversion.
package contacts

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode"
)

const (
	maxContacts  = 5000
	maxCollected = 500
	maxGroups    = 200
	maxField     = 200
	maxNotes     = 2000
	maxPerList   = 20
)

// Contact is one address book entry.
type Contact struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Emails []string `json:"emails"`
	Phones []string `json:"phones,omitempty"`
	Org    string   `json:"org,omitempty"`
	Notes  string   `json:"notes,omitempty"`
	Groups []string `json:"groups,omitempty"`
}

// Collected is an address the user sent mail to that is not in the address book.
type Collected struct {
	Name  string    `json:"name,omitempty"`
	Email string    `json:"email"`
	Count int       `json:"count"`
	Last  time.Time `json:"last"`
}

// Book is everything stored for one user.
type Book struct {
	Contacts  []Contact   `json:"contacts"`
	Groups    []string    `json:"groups"`
	Collected []Collected `json:"collected"`
}

// Recipient is an address taken from a sent message.
type Recipient struct {
	Name  string
	Email string
}

// ErrInvalid wraps every validation failure.
var ErrInvalid = errors.New("contacts: invalid")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalid}, args...)...)
}

func clean(s string, limit int, multiline bool) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) > limit {
		return "", invalid("field longer than %d bytes", limit)
	}
	if strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) && (!multiline || (r != '\n' && r != '\t')) }) {
		return "", invalid("control characters")
	}
	return s, nil
}

// NormalizeEmail validates an address and lowercases its domain.
func NormalizeEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || len(s) > 254 {
		return "", invalid("bad address %q", s)
	}
	local, domain, _ := strings.Cut(s, "@")
	return local + "@" + strings.ToLower(domain), nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func uniqueFold(list []string, limit int) ([]string, error) {
	var out []string
	for _, v := range list {
		v, err := clean(v, maxField, false)
		if err != nil {
			return nil, err
		}
		if v != "" && !slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, v) }) {
			out = append(out, v)
		}
	}
	if len(out) > limit {
		return nil, invalid("more than %d entries", limit)
	}
	return out, nil
}

// Validate trims and dedupes the book, drops references to unknown groups and rejects bad data.
func Validate(b Book) (Book, error) {
	if len(b.Contacts) > maxContacts {
		return Book{}, invalid("more than %d contacts", maxContacts)
	}
	groups, err := uniqueFold(b.Groups, maxGroups)
	if err != nil {
		return Book{}, err
	}
	out := Book{Groups: groups, Contacts: make([]Contact, 0, len(b.Contacts)), Collected: []Collected{}}
	if out.Groups == nil {
		out.Groups = []string{}
	}
	ids := map[string]bool{}
	for _, c := range b.Contacts {
		var n Contact
		if n.Name, err = clean(c.Name, maxField, false); err != nil {
			return Book{}, err
		}
		if n.Org, err = clean(c.Org, maxField, false); err != nil {
			return Book{}, err
		}
		if n.Notes, err = clean(c.Notes, maxNotes, true); err != nil {
			return Book{}, err
		}
		for _, e := range c.Emails {
			if strings.TrimSpace(e) == "" {
				continue
			}
			ne, err := NormalizeEmail(e)
			if err != nil {
				return Book{}, err
			}
			if !slices.ContainsFunc(n.Emails, func(o string) bool { return strings.EqualFold(o, ne) }) {
				n.Emails = append(n.Emails, ne)
			}
		}
		if len(n.Emails) > maxPerList {
			return Book{}, invalid("too many addresses")
		}
		if n.Phones, err = uniqueFold(c.Phones, maxPerList); err != nil {
			return Book{}, err
		}
		for _, g := range c.Groups {
			if i := slices.IndexFunc(groups, func(o string) bool { return strings.EqualFold(o, strings.TrimSpace(g)) }); i >= 0 && !slices.Contains(n.Groups, groups[i]) {
				n.Groups = append(n.Groups, groups[i])
			}
		}
		if n.Name == "" && len(n.Emails) == 0 && len(n.Phones) == 0 {
			return Book{}, invalid("empty contact")
		}
		if n.Emails == nil {
			n.Emails = []string{}
		}
		n.ID = c.ID
		if n.ID == "" || len(n.ID) > 64 || ids[n.ID] {
			n.ID = newID()
		}
		ids[n.ID] = true
		out.Contacts = append(out.Contacts, n)
	}
	for _, c := range b.Collected {
		e, err := NormalizeEmail(c.Email)
		if err != nil {
			continue
		}
		name, err := clean(c.Name, maxField, false)
		if err != nil {
			name = ""
		}
		out.Collected = append(out.Collected, Collected{Name: name, Email: e, Count: max(c.Count, 1), Last: c.Last})
		if len(out.Collected) == maxCollected {
			break
		}
	}
	return out, nil
}

func (b Book) has(email string) bool {
	for _, c := range b.Contacts {
		if slices.ContainsFunc(c.Emails, func(e string) bool { return strings.EqualFold(e, email) }) {
			return true
		}
	}
	return false
}

// Collect records recipients that are not contacts yet, most recent first.
func Collect(b Book, rs []Recipient) Book {
	now := time.Now()
	for _, r := range rs {
		email, err := NormalizeEmail(r.Email)
		if err != nil || b.has(email) {
			continue
		}
		name, _ := clean(r.Name, maxField, false)
		entry := Collected{Name: name, Email: email, Count: 1, Last: now}
		if i := slices.IndexFunc(b.Collected, func(c Collected) bool { return strings.EqualFold(c.Email, email) }); i >= 0 {
			old := b.Collected[i]
			entry.Count += old.Count
			if entry.Name == "" {
				entry.Name = old.Name
			}
			b.Collected = slices.Delete(b.Collected, i, i+1)
		}
		b.Collected = slices.Insert(b.Collected, 0, entry)
	}
	if len(b.Collected) > maxCollected {
		b.Collected = b.Collected[:maxCollected]
	}
	return b
}

// Merge adds imported contacts, folding them into an existing contact that shares an address.
func Merge(b Book, in []Contact) (out Book, added, updated int) {
	addGroup := func(g string) string {
		if i := slices.IndexFunc(b.Groups, func(o string) bool { return strings.EqualFold(o, g) }); i >= 0 {
			return b.Groups[i]
		}
		b.Groups = append(b.Groups, g)
		return g
	}
	for _, c := range in {
		for i, g := range c.Groups {
			c.Groups[i] = addGroup(g)
		}
		i := slices.IndexFunc(b.Contacts, func(old Contact) bool {
			return slices.ContainsFunc(c.Emails, func(e string) bool {
				return slices.ContainsFunc(old.Emails, func(o string) bool { return strings.EqualFold(o, e) })
			})
		})
		if i < 0 {
			c.ID = ""
			b.Contacts = append(b.Contacts, c)
			added++
			continue
		}
		old := &b.Contacts[i]
		old.Emails = appendNew(old.Emails, c.Emails)
		old.Phones = appendNew(old.Phones, c.Phones)
		old.Groups = appendNew(old.Groups, c.Groups)
		if old.Name == "" {
			old.Name = c.Name
		}
		if old.Org == "" {
			old.Org = c.Org
		}
		if old.Notes == "" {
			old.Notes = c.Notes
		}
		updated++
	}
	return b, added, updated
}

func appendNew(dst, src []string) []string {
	for _, v := range src {
		if !slices.ContainsFunc(dst, func(o string) bool { return strings.EqualFold(o, v) }) {
			dst = append(dst, v)
		}
	}
	return dst
}
