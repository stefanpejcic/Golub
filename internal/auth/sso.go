// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stefanpejcic/golub/internal/config"
)

// SSO token limits.
const (
	SSOMaxTTL   = 60 * time.Second
	ssoSkew     = 5 * time.Second
	minNonceLen = 16
)

// SSO verification errors.
var (
	ErrSSOInvalid = errors.New("auth: invalid sso token")
	ErrSSOExpired = errors.New("auth: sso token expired")
	ErrSSOReplay  = errors.New("auth: sso token already used")
)

// SSOClaims is the signed payload: email|exp|nonce|lang|theme.
type SSOClaims struct {
	Email   string
	Expires int64
	Nonce   string
	Lang    string
	Theme   string
}

func (c SSOClaims) payload() string {
	return strings.Join([]string{c.Email, strconv.FormatInt(c.Expires, 10), c.Nonce, c.Lang, c.Theme}, "|")
}

func mac(secret, payload string) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(payload))
	return m.Sum(nil)
}

// SignSSO builds a token the way a host app is expected to.
func SignSSO(secret string, c SSOClaims) string {
	p := c.payload()
	return base64.RawURLEncoding.EncodeToString([]byte(p)) + "." + base64.RawURLEncoding.EncodeToString(mac(secret, p))
}

// SSOVerifier checks tokens and remembers nonces until they expire.
type SSOVerifier struct {
	secret string
	now    func() time.Time

	mu   sync.Mutex
	seen map[string]int64
}

// NewSSOVerifier returns a verifier for tokens signed with secret.
func NewSSOVerifier(secret string) *SSOVerifier {
	return &SSOVerifier{secret: secret, now: time.Now, seen: map[string]int64{}}
}

// Verify validates signature, expiry and fields, and consumes the nonce.
func (v *SSOVerifier) Verify(token string) (SSOClaims, error) {
	var c SSOClaims
	p64, m64, ok := strings.Cut(token, ".")
	if !ok {
		return c, ErrSSOInvalid
	}
	payload, err1 := base64.RawURLEncoding.DecodeString(p64)
	sig, err2 := base64.RawURLEncoding.DecodeString(m64)
	if err1 != nil || err2 != nil || !hmac.Equal(sig, mac(v.secret, string(payload))) {
		return c, ErrSSOInvalid
	}
	parts := strings.Split(string(payload), "|")
	if len(parts) != 5 {
		return c, ErrSSOInvalid
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c, ErrSSOInvalid
	}
	c = SSOClaims{Email: parts[0], Expires: exp, Nonce: parts[2], Lang: parts[3], Theme: parts[4]}
	if err := validateClaims(c); err != nil {
		return SSOClaims{}, err
	}

	now := v.now()
	expAt := time.Unix(exp, 0)
	if !now.Before(expAt) {
		return SSOClaims{}, ErrSSOExpired
	}
	if expAt.Sub(now) > SSOMaxTTL+ssoSkew {
		return SSOClaims{}, fmt.Errorf("%w: ttl longer than %s", ErrSSOInvalid, SSOMaxTTL)
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	for n, e := range v.seen {
		if now.Unix() >= e {
			delete(v.seen, n)
		}
	}
	if _, used := v.seen[c.Nonce]; used {
		return SSOClaims{}, ErrSSOReplay
	}
	v.seen[c.Nonce] = exp
	return c, nil
}

func (v *SSOVerifier) seenCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.seen)
}

func validateClaims(c SSOClaims) error {
	if !ValidEmail(c.Email) {
		return fmt.Errorf("%w: email", ErrSSOInvalid)
	}
	if len(c.Nonce) < minNonceLen {
		return fmt.Errorf("%w: nonce", ErrSSOInvalid)
	}
	if c.Lang != "" && !slices.Contains(config.Locales, c.Lang) {
		return fmt.Errorf("%w: lang", ErrSSOInvalid)
	}
	if c.Theme != "" && !slices.Contains(config.Themes, c.Theme) {
		return fmt.Errorf("%w: theme", ErrSSOInvalid)
	}
	return nil
}

// ValidEmail accepts a bare address and rejects anything Dovecot could read as a master login.
func ValidEmail(s string) bool {
	if s == "" || len(s) > 254 || strings.ContainsAny(s, "*|\"<> \t\r\n") {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s && strings.Contains(s, "@")
}
