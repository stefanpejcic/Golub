// SPDX-License-Identifier: GPL-3.0-or-later

// Package settings defines per-user preferences and their validation.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"unicode"
)

// Size limits keep the stored message small and the UI responsive.
const (
	MaxIdentities = 20
	MaxResponses  = 100
	MaxSignature  = 10 << 10
	MaxResponse   = 20 << 10
	maxShort      = 200
)

// Identity is a sender the user can write as.
type Identity struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Organization  string `json:"organization,omitempty"`
	ReplyTo       string `json:"replyTo,omitempty"`
	Bcc           string `json:"bcc,omitempty"`
	Signature     string `json:"signature,omitempty"`
	SignatureHTML bool   `json:"signatureHtml,omitempty"`
	Default       bool   `json:"default,omitempty"`
}

// Response is a canned reply inserted from the composer.
type Response struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

// Settings are the user's preferences.
type Settings struct {
	PageSize           int               `json:"pageSize"`
	SortBy             string            `json:"sortBy"`
	SortDesc           bool              `json:"sortDesc"`
	Threads            bool              `json:"threads"`
	PrettyDates        bool              `json:"prettyDates"`
	RefreshInterval    int               `json:"refreshInterval"`
	MarkRead           string            `json:"markRead"`
	MarkReadDelay      int               `json:"markReadDelay"`
	PreferPlain        bool              `json:"preferPlain"`
	RemoteImages       string            `json:"remoteImages"`
	ShowAddresses      bool              `json:"showAddresses"`
	ComposeHTML        bool              `json:"composeHtml"`
	ReplyPosition      string            `json:"replyPosition"`
	ForwardMode        string            `json:"forwardMode"`
	SignatureOnReply   bool              `json:"signatureOnReply"`
	DraftAutosave      int               `json:"draftAutosave"`
	SpellCheck         bool              `json:"spellCheck"`
	CollectRecipients  bool              `json:"collectRecipients"`
	DeleteMode         string            `json:"deleteMode"`
	EmptyTrashOnLogout bool              `json:"emptyTrashOnLogout"`
	SpecialFolders     map[string]string `json:"specialFolders,omitempty"`
	Identities         []Identity        `json:"identities"`
	Responses          []Response        `json:"responses"`
}

// Specials lists the folder roles that can be remapped.
var Specials = []string{"drafts", "sent", "junk", "trash", "archive"}

// Defaults returns sensible settings for a new user.
func Defaults(email string) Settings {
	return Settings{
		PageSize:          50,
		SortBy:            "date",
		SortDesc:          true,
		PrettyDates:       true,
		RefreshInterval:   60,
		MarkRead:          "immediately",
		MarkReadDelay:     3,
		RemoteImages:      "ask",
		ShowAddresses:     true,
		ComposeHTML:       true,
		ReplyPosition:     "above",
		ForwardMode:       "inline",
		SpellCheck:        true,
		CollectRecipients: true,
		SignatureOnReply:  true,
		DraftAutosave:     60,
		DeleteMode:        "trash",
		Identities:        []Identity{{ID: "default", Name: "", Email: email, Default: true}},
		Responses:         []Response{},
	}
}

// Decode merges stored JSON onto defaults; on error the defaults are returned with it.
func Decode(data []byte, email string) (Settings, error) {
	s := Defaults(email)
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Defaults(email), err
	}
	if len(s.Identities) == 0 {
		s.Identities = Defaults(email).Identities
	}
	if s.Responses == nil {
		s.Responses = []Response{}
	}
	return s, nil
}

// Encode serializes settings for storage.
func Encode(s Settings) ([]byte, error) {
	return json.Marshal(s)
}

func oneOf[T comparable](v T, set ...T) bool { return slices.Contains(set, v) }

// cleanLine rejects control characters, which would allow header injection.
func cleanLine(s string, limit int) bool {
	if len(s) > limit {
		return false
	}
	return !strings.ContainsFunc(s, unicode.IsControl)
}

func validAddress(s string, optional bool) bool {
	if s == "" {
		return optional
	}
	if !cleanLine(s, 254) {
		return false
	}
	_, err := mail.ParseAddress(s)
	return err == nil
}

// Validate checks every field; the first problem is returned.
func (s Settings) Validate() error {
	switch {
	case !oneOf(s.PageSize, 25, 50, 100, 200):
		return errors.New("pageSize")
	case !oneOf(s.SortBy, "date", "arrival", "subject", "from", "size"):
		return errors.New("sortBy")
	case !oneOf(s.RefreshInterval, 0, 30, 60, 300, 600):
		return errors.New("refreshInterval")
	case !oneOf(s.MarkRead, "immediately", "delay", "never"):
		return errors.New("markRead")
	case s.MarkRead == "delay" && (s.MarkReadDelay < 1 || s.MarkReadDelay > 60):
		return errors.New("markReadDelay")
	case !oneOf(s.RemoteImages, "ask", "always"):
		return errors.New("remoteImages")
	case !oneOf(s.ReplyPosition, "above", "below"):
		return errors.New("replyPosition")
	case !oneOf(s.ForwardMode, "inline", "attachment"):
		return errors.New("forwardMode")
	case !oneOf(s.DraftAutosave, 0, 30, 60, 180, 300):
		return errors.New("draftAutosave")
	case !oneOf(s.DeleteMode, "trash", "permanent"):
		return errors.New("deleteMode")
	}
	for role, name := range s.SpecialFolders {
		if !slices.Contains(Specials, role) || name == "" || !cleanLine(name, 512) {
			return fmt.Errorf("specialFolders.%s", role)
		}
	}

	if len(s.Identities) == 0 || len(s.Identities) > MaxIdentities {
		return errors.New("identities")
	}
	ids := map[string]bool{}
	defaults := 0
	for i, id := range s.Identities {
		switch {
		case id.ID == "" || !cleanLine(id.ID, 64) || ids[id.ID]:
			return fmt.Errorf("identities.%d.id", i)
		case !cleanLine(id.Name, maxShort) || !cleanLine(id.Organization, maxShort):
			return fmt.Errorf("identities.%d.name", i)
		case !validAddress(id.Email, false) || strings.Contains(id.Email, "*"):
			return fmt.Errorf("identities.%d.email", i)
		case !validAddress(id.ReplyTo, true) || !validAddress(id.Bcc, true):
			return fmt.Errorf("identities.%d.address", i)
		case len(id.Signature) > MaxSignature:
			return fmt.Errorf("identities.%d.signature", i)
		}
		ids[id.ID] = true
		if id.Default {
			defaults++
		}
	}
	if defaults != 1 {
		return errors.New("identities.default")
	}

	if len(s.Responses) > MaxResponses {
		return errors.New("responses")
	}
	rids := map[string]bool{}
	for i, r := range s.Responses {
		if r.ID == "" || !cleanLine(r.ID, 64) || rids[r.ID] || r.Name == "" || !cleanLine(r.Name, maxShort) || len(r.Text) > MaxResponse {
			return fmt.Errorf("responses.%d", i)
		}
		rids[r.ID] = true
	}
	return nil
}
