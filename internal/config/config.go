// SPDX-License-Identifier: GPL-3.0-or-later

// Package config parses and validates Golub settings from environment variables.
package config

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// TLSMode is the transport security used towards a mail server.
type TLSMode string

// Supported mail server TLS modes.
const (
	TLSImplicit TLSMode = "implicit"
	TLSStartTLS TLSMode = "starttls"
	TLSNone     TLSMode = "none"
)

// Endpoint is a mail server address and how to secure it.
type Endpoint struct {
	Host string
	Port int
	TLS  TLSMode
}

// Addr returns host:port.
func (e Endpoint) Addr() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// Config holds the validated runtime settings.
type Config struct {
	Listen         string
	TLSCert        string
	TLSKey         string
	TrustProxy     []netip.Prefix
	IMAP           Endpoint
	SMTP           Endpoint
	TLSCA          string
	SecretKey      []byte
	SSOSecret      string
	MasterUser     string
	MasterPassword string
	SessionTTL     time.Duration
	DefaultLocale  string
	DefaultTheme   string
	LogLevel       string
	BrandName      string
	BrandAccent    string
	BrandLogo      []byte
	BrandLogoType  string
}

// HTTPS reports whether Golub terminates TLS itself.
func (c *Config) HTTPS() bool { return c.TLSCert != "" }

// SSOEnabled reports whether the SSO endpoint is active.
func (c *Config) SSOEnabled() bool { return c.SSOSecret != "" }

// Locales lists the supported UI locales.
var Locales = []string{"en", "sr-Latn", "ar"}

// Themes lists the supported theme settings.
var Themes = []string{"system", "light", "dark"}

// LookupFunc resolves a host name to addresses.
type LookupFunc func(host string) ([]netip.Addr, error)

// SystemLookup resolves with the default resolver.
func SystemLookup(host string) ([]netip.Addr, error) {
	if a, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{a}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

const minSSOSecret = 32

// Load reads all GOLUB_* variables through getenv and validates them.
func Load(getenv func(string) string, lookup LookupFunc) (*Config, error) {
	var errs []error
	bad := func(name, format string, a ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", name, fmt.Sprintf(format, a...)))
	}
	get := func(name, def string) string {
		if v := strings.TrimSpace(getenv(name)); v != "" {
			return v
		}
		return def
	}

	c := &Config{
		Listen:         get("GOLUB_LISTEN", ":8080"),
		TLSCert:        get("GOLUB_TLS_CERT", ""),
		TLSKey:         get("GOLUB_TLS_KEY", ""),
		TLSCA:          get("GOLUB_TLS_CA", ""),
		SSOSecret:      getenv("GOLUB_SSO_SECRET"),
		MasterUser:     get("GOLUB_MASTER_USER", ""),
		MasterPassword: getenv("GOLUB_MASTER_PASSWORD"),
		DefaultLocale:  get("GOLUB_DEFAULT_LOCALE", "en"),
		DefaultTheme:   get("GOLUB_DEFAULT_THEME", "system"),
		LogLevel:       get("GOLUB_LOG_LEVEL", "info"),
	}

	switch {
	case c.TLSCert != "" && c.TLSKey == "":
		bad("GOLUB_TLS_KEY", "required when GOLUB_TLS_CERT is set")
	case c.TLSKey != "" && c.TLSCert == "":
		bad("GOLUB_TLS_CERT", "required when GOLUB_TLS_KEY is set")
	}

	if v := get("GOLUB_TRUST_PROXY", ""); v != "" {
		for _, s := range strings.Split(v, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			p, err := netip.ParsePrefix(s)
			if err != nil {
				bad("GOLUB_TRUST_PROXY", "invalid CIDR %q", s)
				continue
			}
			c.TrustProxy = append(c.TrustProxy, p.Masked())
		}
	}

	c.IMAP = endpoint(get, bad, "IMAP", get("GOLUB_IMAP_HOST", ""), 993, 143)
	if c.IMAP.Host == "" {
		bad("GOLUB_IMAP_HOST", "required")
	}
	c.SMTP = endpoint(get, bad, "SMTP", get("GOLUB_SMTP_HOST", c.IMAP.Host), 465, 587)
	for _, e := range []struct {
		proto string
		ep    Endpoint
	}{{"IMAP", c.IMAP}, {"SMTP", c.SMTP}} {
		if e.ep.TLS == TLSNone && e.ep.Host != "" {
			if err := requirePrivate(e.ep.Host, lookup); err != nil {
				bad("GOLUB_"+e.proto+"_TLS", "none is only allowed for loopback or private hosts: %v", err)
			}
		}
	}

	if k := getenv("GOLUB_SECRET_KEY"); k == "" {
		bad("GOLUB_SECRET_KEY", "required (32 random bytes, base64)")
	} else if key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(k)); err != nil {
		bad("GOLUB_SECRET_KEY", "not valid base64")
	} else if len(key) != 32 {
		bad("GOLUB_SECRET_KEY", "must decode to 32 bytes, got %d", len(key))
	} else {
		c.SecretKey = key
	}

	if c.SSOSecret != "" {
		if len(c.SSOSecret) < minSSOSecret {
			bad("GOLUB_SSO_SECRET", "must be at least %d characters", minSSOSecret)
		}
		if c.MasterUser == "" {
			bad("GOLUB_MASTER_USER", "required when GOLUB_SSO_SECRET is set")
		}
		if c.MasterPassword == "" {
			bad("GOLUB_MASTER_PASSWORD", "required when GOLUB_SSO_SECRET is set")
		}
	}

	c.SessionTTL = 12 * time.Hour
	if v := get("GOLUB_SESSION_TTL", ""); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			bad("GOLUB_SESSION_TTL", "must be a positive duration like 12h")
		} else {
			c.SessionTTL = d
		}
	}

	if !oneOf(c.DefaultLocale, Locales) {
		bad("GOLUB_DEFAULT_LOCALE", "must be one of %s", strings.Join(Locales, ", "))
	}
	if !oneOf(c.DefaultTheme, Themes) {
		bad("GOLUB_DEFAULT_THEME", "must be one of %s", strings.Join(Themes, ", "))
	}
	if !oneOf(c.LogLevel, []string{"debug", "info", "warn", "error"}) {
		bad("GOLUB_LOG_LEVEL", "must be one of debug, info, warn, error")
	}

	c.BrandName = get("GOLUB_BRAND_NAME", "Golub")
	if len(c.BrandName) > 64 || strings.ContainsFunc(c.BrandName, unicode.IsControl) {
		bad("GOLUB_BRAND_NAME", "must be at most 64 printable characters")
	}
	if v := get("GOLUB_BRAND_ACCENT", ""); v != "" {
		if !accentRe.MatchString(v) {
			bad("GOLUB_BRAND_ACCENT", "must be a colour like #2563eb")
		} else {
			c.BrandAccent = strings.ToLower(v)
		}
	}
	if path := get("GOLUB_BRAND_LOGO", ""); path != "" {
		ct, ok := logoTypes[strings.ToLower(filepath.Ext(path))]
		data, err := os.ReadFile(path)
		switch {
		case !ok:
			bad("GOLUB_BRAND_LOGO", "must be a .png, .jpg, .svg or .webp file")
		case err != nil:
			bad("GOLUB_BRAND_LOGO", "%v", err)
		case len(data) > maxLogo:
			bad("GOLUB_BRAND_LOGO", "must be at most %d KB", maxLogo>>10)
		default:
			c.BrandLogo, c.BrandLogoType = data, ct
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return c, nil
}

const maxLogo = 512 << 10

var (
	accentRe  = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	logoTypes = map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".svg": "image/svg+xml", ".webp": "image/webp"}
)

func endpoint(get func(string, string) string, bad func(string, string, ...any), proto, host string, implicitPort, plainPort int) Endpoint {
	e := Endpoint{Host: host, TLS: TLSMode(get("GOLUB_"+proto+"_TLS", string(TLSStartTLS)))}
	switch e.TLS {
	case TLSImplicit:
		e.Port = implicitPort
	case TLSStartTLS, TLSNone:
		e.Port = plainPort
	default:
		bad("GOLUB_"+proto+"_TLS", "must be implicit, starttls or none")
	}
	if v := get("GOLUB_"+proto+"_PORT", ""); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			bad("GOLUB_"+proto+"_PORT", "must be 1-65535")
		} else {
			e.Port = p
		}
	}
	return e
}

func requirePrivate(host string, lookup LookupFunc) error {
	addrs, err := lookup(host)
	if err != nil {
		return fmt.Errorf("GOLUB_IMAP_HOST/GOLUB_SMTP_HOST %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("%q resolved to no addresses", host)
	}
	for _, a := range addrs {
		a = a.Unmap()
		if !a.IsLoopback() && !a.IsPrivate() {
			return fmt.Errorf("%q resolves to public address %s", host, a)
		}
	}
	return nil
}

func oneOf(v string, set []string) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}
