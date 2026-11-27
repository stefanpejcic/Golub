// SPDX-License-Identifier: GPL-3.0-or-later

// Package keyring checks the PGP keyring the browser keeps in the user's mailbox; only the browser reads the keys.
package keyring

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

const (
	maxKeys   = 500
	maxArmor  = 64 << 10
	maxEmails = 20
)

// Key is one public key, or a key pair when Private is set.
type Key struct {
	Fingerprint string   `json:"fingerprint"`
	UserIDs     []string `json:"userIds,omitempty"`
	Emails      []string `json:"emails"`
	Private     bool     `json:"private"`
	Created     string   `json:"created,omitempty"`
	Armored     string   `json:"armored"`
}

// Ring is the stored keyring.
type Ring struct {
	Keys []Key `json:"keys"`
}

// ErrInvalid wraps every validation failure.
var ErrInvalid = errors.New("keyring: invalid")

var fingerprintRe = regexp.MustCompile(`^[0-9a-f]{40}$|^[0-9a-f]{64}$`)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalid}, args...)...)
}

// Validate checks every key and merges duplicates, keeping the private one.
func Validate(r Ring) (Ring, error) {
	if len(r.Keys) > maxKeys {
		return Ring{}, invalid("more than %d keys", maxKeys)
	}
	out := Ring{Keys: []Key{}}
	index := map[string]int{}
	for _, k := range r.Keys {
		k.Fingerprint = strings.ToLower(strings.TrimSpace(k.Fingerprint))
		if !fingerprintRe.MatchString(k.Fingerprint) {
			return Ring{}, invalid("bad fingerprint %q", k.Fingerprint)
		}
		if len(k.Armored) > maxArmor {
			return Ring{}, invalid("key %s is too large", k.Fingerprint)
		}
		isPriv := strings.HasPrefix(k.Armored, "-----BEGIN PGP PRIVATE KEY BLOCK-----")
		isPub := strings.HasPrefix(k.Armored, "-----BEGIN PGP PUBLIC KEY BLOCK-----")
		if (!isPriv && !isPub) || isPriv != k.Private {
			return Ring{}, invalid("key %s is not an armored key of the stated kind", k.Fingerprint)
		}
		if len(k.Emails) > maxEmails || len(k.UserIDs) > maxEmails {
			return Ring{}, invalid("key %s has too many user ids", k.Fingerprint)
		}
		emails := []string{}
		for _, e := range k.Emails {
			a, err := mail.ParseAddress(e)
			if err != nil || a.Address != e {
				return Ring{}, invalid("bad address %q", e)
			}
			emails = append(emails, strings.ToLower(e))
		}
		k.Emails = emails
		if i, seen := index[k.Fingerprint]; seen {
			if k.Private {
				out.Keys[i] = k
			}
			continue
		}
		index[k.Fingerprint] = len(out.Keys)
		out.Keys = append(out.Keys, k)
	}
	return out, nil
}
