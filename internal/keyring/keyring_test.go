// SPDX-License-Identifier: GPL-3.0-or-later

package keyring

import (
	"strings"
	"testing"
)

const pub = "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nxjMEZ\n-----END PGP PUBLIC KEY BLOCK-----\n"
const priv = "-----BEGIN PGP PRIVATE KEY BLOCK-----\n\nxYYEZ\n-----END PGP PRIVATE KEY BLOCK-----\n"

func TestValidate(t *testing.T) {
	r, err := Validate(Ring{Keys: []Key{
		{Fingerprint: "ABCDEF0123456789ABCDEF0123456789ABCDEF01", Emails: []string{"Ana@Golub.Test"}, Private: true, Armored: priv},
		{Fingerprint: "abcdef0123456789abcdef0123456789abcdef01", Emails: []string{"ana@golub.test"}, Armored: pub},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Keys) != 1 || !r.Keys[0].Private || r.Keys[0].Fingerprint != "abcdef0123456789abcdef0123456789abcdef01" || r.Keys[0].Emails[0] != "ana@golub.test" {
		t.Fatalf("the same key twice collapses, private wins, fingerprint and emails lowercased: %+v", r.Keys)
	}

	for name, bad := range map[string]Key{
		"fingerprint":  {Fingerprint: "xyz", Armored: pub},
		"armor":        {Fingerprint: strings.Repeat("a", 40), Armored: "hello"},
		"private flag": {Fingerprint: strings.Repeat("a", 40), Armored: priv, Private: false},
		"public flag":  {Fingerprint: strings.Repeat("a", 40), Armored: pub, Private: true},
		"email":        {Fingerprint: strings.Repeat("a", 40), Armored: pub, Emails: []string{"no at sign"}},
	} {
		if _, err := Validate(Ring{Keys: []Key{bad}}); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := Validate(Ring{Keys: make([]Key, maxKeys+1)}); err == nil {
		t.Error("too many keys")
	}
}
