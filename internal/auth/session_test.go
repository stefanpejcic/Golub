// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

var key = bytes.Repeat([]byte{7}, 32)

func TestSessionRoundTrip(t *testing.T) {
	c, err := NewCodec(key)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	s := Session{Email: "alice@golub.test", Mode: ModePassword, Password: "p|w", Expires: now.Add(time.Hour).Unix(), Locale: "sr-Latn", Theme: "dark"}
	v, err := c.Encode(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(v, "alice") || strings.Contains(v, "p|w") {
		t.Fatal("cookie value leaks plaintext")
	}
	got, err := c.Decode(v, now)
	if err != nil {
		t.Fatal(err)
	}
	if got != s {
		t.Errorf("got %+v", got)
	}
}

func TestSessionRejects(t *testing.T) {
	c, _ := NewCodec(key)
	other, _ := NewCodec(bytes.Repeat([]byte{8}, 32))
	now := time.Unix(1_800_000_000, 0)
	valid, _ := c.Encode(Session{Email: "a@b", Mode: ModeSSO, Expires: now.Add(time.Minute).Unix()})
	expired, _ := c.Encode(Session{Email: "a@b", Mode: ModeSSO, Expires: now.Add(-time.Second).Unix()})
	foreign, _ := other.Encode(Session{Email: "a@b", Mode: ModeSSO, Expires: now.Add(time.Minute).Unix()})
	tampered := valid[:len(valid)-2] + "AA"

	for name, v := range map[string]string{"expired": expired, "other key": foreign, "tampered": tampered, "garbage": "!!", "empty": ""} {
		t.Run(name, func(t *testing.T) {
			if _, err := c.Decode(v, now); !errors.Is(err, ErrInvalidSession) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestNewCodecKeyLength(t *testing.T) {
	if _, err := NewCodec([]byte("short")); err == nil {
		t.Fatal("expected error")
	}
}
