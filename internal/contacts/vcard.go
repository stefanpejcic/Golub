// SPDX-License-Identifier: GPL-3.0-or-later

package contacts

import (
	"errors"
	"io"
	"strings"

	"github.com/emersion/go-vcard"
)

// maxImport caps one import so a huge file cannot pin memory.
const maxImport = maxContacts

// ParseVCards reads every card; cards that cannot be used are skipped.
func ParseVCards(r io.Reader) ([]Contact, error) {
	dec := vcard.NewDecoder(r)
	var out []Contact
	for len(out) < maxImport {
		card, err := dec.Decode()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if len(out) == 0 {
				return nil, invalid("not a vCard file")
			}
			break
		}
		c := Contact{Name: strings.TrimSpace(card.PreferredValue(vcard.FieldFormattedName))}
		if c.Name == "" {
			if n := card.Name(); n != nil {
				c.Name = strings.TrimSpace(strings.Join(strings.Fields(n.GivenName+" "+n.AdditionalName+" "+n.FamilyName), " "))
			}
		}
		c.Emails = card.Values(vcard.FieldEmail)
		c.Phones = card.Values(vcard.FieldTelephone)
		c.Org = strings.ReplaceAll(card.PreferredValue(vcard.FieldOrganization), ";", " ")
		c.Notes = card.PreferredValue(vcard.FieldNote)
		for _, cat := range card.Categories() {
			if cat = strings.TrimSpace(cat); cat != "" {
				c.Groups = append(c.Groups, cat)
			}
		}
		if c.Name == "" && len(c.Emails) == 0 && len(c.Phones) == 0 {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// WriteVCards writes vCard 3.0, which every address book still reads.
func WriteVCards(w io.Writer, list []Contact) error {
	enc := vcard.NewEncoder(w)
	for _, c := range list {
		card := vcard.Card{}
		card.SetValue(vcard.FieldVersion, "3.0")
		name := c.Name
		if name == "" && len(c.Emails) > 0 {
			name = c.Emails[0]
		}
		card.SetValue(vcard.FieldFormattedName, name)
		card.SetName(&vcard.Name{FamilyName: name})
		for _, e := range c.Emails {
			card.AddValue(vcard.FieldEmail, e)
		}
		for _, p := range c.Phones {
			card.AddValue(vcard.FieldTelephone, p)
		}
		if c.Org != "" {
			card.SetValue(vcard.FieldOrganization, c.Org)
		}
		if c.Notes != "" {
			card.SetValue(vcard.FieldNote, c.Notes)
		}
		if len(c.Groups) > 0 {
			card.SetCategories(c.Groups)
		}
		if c.ID != "" {
			card.SetValue(vcard.FieldUID, c.ID)
		}
		if err := enc.Encode(card); err != nil {
			return err
		}
	}
	return nil
}
