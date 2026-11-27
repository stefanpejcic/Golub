// SPDX-License-Identifier: GPL-3.0-or-later

package compose

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

var errAddress = errors.New("compose: invalid address")

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// ASCIIAddress puts the domain into its IDNA (punycode) form and reports whether the local part
// still needs SMTPUTF8.
func ASCIIAddress(addr string) (string, bool, error) {
	local, domain, ok := strings.Cut(strings.TrimSpace(addr), "@")
	if !ok || local == "" || domain == "" {
		return "", false, errAddress
	}
	ascii, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return "", false, err
	}
	return local + "@" + ascii, !isASCII(local), nil
}

// UnicodeAddress shows a punycode domain in its readable form.
func UnicodeAddress(addr string) string {
	local, domain, ok := strings.Cut(addr, "@")
	if !ok || !strings.Contains(strings.ToLower(domain), "xn--") {
		return addr
	}
	u, err := idna.Display.ToUnicode(domain)
	if err != nil {
		return addr
	}
	return local + "@" + u
}

func asciiList(list []mail.Address) []mail.Address {
	out := make([]mail.Address, len(list))
	for i, a := range list {
		out[i] = a
		if conv, _, err := ASCIIAddress(a.Address); err == nil {
			out[i].Address = conv
		}
	}
	return out
}
