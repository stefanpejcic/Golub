// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	s := Defaults("alice@golub.test")
	if s.PageSize != 50 || s.MarkRead != "immediately" || s.RemoteImages != "ask" || s.DeleteMode != "trash" || !s.PrettyDates {
		t.Fatalf("%+v", s)
	}
	if len(s.Identities) != 1 || s.Identities[0].Email != "alice@golub.test" || !s.Identities[0].Default {
		t.Fatalf("identities %+v", s.Identities)
	}
}

func TestDecodeMergesOntoDefaults(t *testing.T) {
	s, err := Decode([]byte(`{"pageSize":100,"preferPlain":true}`), "alice@golub.test")
	if err != nil {
		t.Fatal(err)
	}
	if s.PageSize != 100 || !s.PreferPlain || s.MarkRead != "immediately" || len(s.Identities) != 1 {
		t.Fatalf("%+v", s)
	}
}

func TestDecodeToleratesBrokenStore(t *testing.T) {
	s, err := Decode([]byte(`not json`), "alice@golub.test")
	if err == nil || s.PageSize != 50 {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestValidate(t *testing.T) {
	ok := Defaults("alice@golub.test")
	cases := map[string]func(*Settings){
		"page size":       func(s *Settings) { s.PageSize = 7 },
		"mark read":       func(s *Settings) { s.MarkRead = "sometimes" },
		"mark read delay": func(s *Settings) { s.MarkRead = "delay"; s.MarkReadDelay = 0 },
		"remote images":   func(s *Settings) { s.RemoteImages = "maybe" },
		"delete mode":     func(s *Settings) { s.DeleteMode = "shred" },
		"reply position":  func(s *Settings) { s.ReplyPosition = "middle" },
		"forward mode":    func(s *Settings) { s.ForwardMode = "fax" },
		"autosave":        func(s *Settings) { s.DraftAutosave = 7 },
		"refresh":         func(s *Settings) { s.RefreshInterval = 5 },
		"special folder":  func(s *Settings) { s.SpecialFolders = map[string]string{"bogus": "X"} },
		"folder name":     func(s *Settings) { s.SpecialFolders = map[string]string{"trash": "a\r\nb"} },
		"no identity":     func(s *Settings) { s.Identities = nil },
		"bad email":       func(s *Settings) { s.Identities[0].Email = "nope" },
		"two defaults": func(s *Settings) {
			s.Identities = append(s.Identities, Identity{ID: "b", Name: "B", Email: "b@golub.test", Default: true})
		},
		"no default": func(s *Settings) { s.Identities[0].Default = false },
		"dup id": func(s *Settings) {
			s.Identities = append(s.Identities, Identity{ID: s.Identities[0].ID, Email: "b@golub.test"})
		},
		"huge signature":   func(s *Settings) { s.Identities[0].Signature = strings.Repeat("x", MaxSignature+1) },
		"header injection": func(s *Settings) { s.Identities[0].Name = "Eve\r\nBcc: x@y.z" },
		"response name":    func(s *Settings) { s.Responses = []Response{{ID: "r", Name: "", Text: "x"}} },
		"response size": func(s *Settings) {
			s.Responses = []Response{{ID: "r", Name: "n", Text: strings.Repeat("x", MaxResponse+1)}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := Defaults("alice@golub.test")
			s.Identities = append([]Identity(nil), ok.Identities...)
			mutate(&s)
			if err := s.Validate(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
}

func TestEncodeIsCompactJSON(t *testing.T) {
	s := Defaults("alice@golub.test")
	b, err := Encode(s)
	if err != nil || !json.Valid(b) || len(b) > 4096 {
		t.Fatalf("%d %v", len(b), err)
	}
}

func TestSpellCheckDefaultsOn(t *testing.T) {
	s, err := Decode([]byte(`{"pageSize":25}`), "alice@golub.test")
	if err != nil || !s.SpellCheck {
		t.Fatalf("%v %+v", err, s)
	}
	s, _ = Decode([]byte(`{"spellCheck":false}`), "alice@golub.test")
	if s.SpellCheck {
		t.Fatal("explicit false must stick")
	}
}
