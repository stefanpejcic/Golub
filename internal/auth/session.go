// SPDX-License-Identifier: GPL-3.0-or-later

// Package auth handles the encrypted session cookie and SSO tokens.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// Login modes stored in the session.
const (
	ModePassword = "password"
	ModeSSO      = "sso"
)

// ErrInvalidSession covers missing, tampered and expired cookies alike.
var ErrInvalidSession = errors.New("auth: invalid session")

var sessionAAD = []byte("golub-session-v1")

// Session is everything Golub knows about a logged-in user.
type Session struct {
	Email    string `json:"e"`
	Mode     string `json:"m"`
	Password string `json:"p,omitempty"`
	Expires  int64  `json:"x"`
	Locale   string `json:"l,omitempty"`
	Theme    string `json:"t,omitempty"`
}

// Codec seals sessions with AES-256-GCM.
type Codec struct {
	aead cipher.AEAD
}

// NewCodec builds a codec from a 32-byte key.
func NewCodec(key []byte) (*Codec, error) {
	if len(key) != 32 {
		return nil, errors.New("auth: session key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Codec{aead: aead}, nil
}

// Encode returns the cookie value for s.
func (c *Codec) Encode(s Session) (string, error) {
	plain, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(c.aead.Seal(nonce, nonce, plain, sessionAAD)), nil
}

// Decode opens a cookie value and rejects it once expired.
func (c *Codec) Decode(v string, now time.Time) (Session, error) {
	var s Session
	raw, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil || len(raw) < c.aead.NonceSize() {
		return s, ErrInvalidSession
	}
	n := c.aead.NonceSize()
	plain, err := c.aead.Open(nil, raw[:n], raw[n:], sessionAAD)
	if err != nil {
		return s, ErrInvalidSession
	}
	if err := json.Unmarshal(plain, &s); err != nil || s.Email == "" || now.Unix() >= s.Expires {
		return Session{}, ErrInvalidSession
	}
	return s, nil
}
