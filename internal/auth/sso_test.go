// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const ssoSecret = "0123456789abcdef0123456789abcdef"

func TestSSOVerify(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	v := NewSSOVerifier(ssoSecret)
	v.now = func() time.Time { return now }

	tok := SignSSO(ssoSecret, SSOClaims{Email: "alice@golub.test", Expires: now.Add(30 * time.Second).Unix(), Nonce: "nonce-0123456789", Lang: "sr-Latn", Theme: "dark"})
	c, err := v.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Email != "alice@golub.test" || c.Lang != "sr-Latn" || c.Theme != "dark" {
		t.Errorf("claims = %+v", c)
	}
	if _, err := v.Verify(tok); !errors.Is(err, ErrSSOReplay) {
		t.Errorf("replay err = %v", err)
	}
}

func TestSSOOptionalFields(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	v := NewSSOVerifier(ssoSecret)
	v.now = func() time.Time { return now }
	tok := SignSSO(ssoSecret, SSOClaims{Email: "bob@golub.test", Expires: now.Add(10 * time.Second).Unix(), Nonce: "nonce-abcdefghijk"})
	c, err := v.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Lang != "" || c.Theme != "" {
		t.Errorf("claims = %+v", c)
	}
}

func TestSSORejects(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ok := SSOClaims{Email: "alice@golub.test", Expires: now.Add(30 * time.Second).Unix(), Nonce: "nonce-0123456789"}
	mod := func(f func(*SSOClaims)) SSOClaims { c := ok; f(&c); return c }

	cases := map[string]string{
		"wrong secret":    SignSSO(strings.Repeat("x", 32), ok),
		"expired":         SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Expires = now.Add(-time.Second).Unix() })),
		"ttl too long":    SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Expires = now.Add(10 * time.Minute).Unix() })),
		"short nonce":     SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Nonce = "abc" })),
		"bad email":       SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Email = "not-an-email" })),
		"master in email": SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Email = "alice@golub.test*admin" })),
		"bad lang":        SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Lang = "xx" })),
		"bad theme":       SignSSO(ssoSecret, mod(func(c *SSOClaims) { c.Theme = "pink" })),
		"no dot":          "abc",
		"bad base64":      "!!!.???",
		"empty":           "",
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			v := NewSSOVerifier(ssoSecret)
			v.now = func() time.Time { return now }
			if _, err := v.Verify(tok); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSSONonceSweep(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	v := NewSSOVerifier(ssoSecret)
	v.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		tok := SignSSO(ssoSecret, SSOClaims{Email: "a@b.c", Expires: now.Add(5 * time.Second).Unix(), Nonce: "nonce-sweep-" + string(rune('a'+i)) + "xxxx"})
		if _, err := v.Verify(tok); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(2 * time.Minute)
	tok := SignSSO(ssoSecret, SSOClaims{Email: "a@b.c", Expires: now.Add(5 * time.Second).Unix(), Nonce: "nonce-after-sweep"})
	if _, err := v.Verify(tok); err != nil {
		t.Fatal(err)
	}
	if n := v.seenCount(); n != 1 {
		t.Errorf("seen = %d, want 1 after sweep", n)
	}
}
