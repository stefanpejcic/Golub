// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"encoding/base64"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func base() map[string]string {
	return map[string]string{
		"GOLUB_IMAP_HOST":  "mail.example.com",
		"GOLUB_SECRET_KEY": testKey,
	}
}

func with(kv ...string) map[string]string {
	m := base()
	for i := 0; i < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func publicLookup(string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("203.0.113.10")}, nil
}

func privateLookup(string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("10.0.0.5")}, nil
}

func TestDefaults(t *testing.T) {
	c, err := Load(env(base()), publicLookup)
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != ":8080" {
		t.Errorf("Listen = %q", c.Listen)
	}
	if c.IMAP.Port != 143 || c.IMAP.TLS != TLSStartTLS {
		t.Errorf("IMAP = %+v", c.IMAP)
	}
	if c.SMTP.Host != "mail.example.com" || c.SMTP.Port != 587 || c.SMTP.TLS != TLSStartTLS {
		t.Errorf("SMTP = %+v", c.SMTP)
	}
	if c.SessionTTL != 12*time.Hour {
		t.Errorf("SessionTTL = %v", c.SessionTTL)
	}
	if c.DefaultLocale != "en" || c.DefaultTheme != "system" || c.LogLevel != "info" {
		t.Errorf("defaults = %q %q %q", c.DefaultLocale, c.DefaultTheme, c.LogLevel)
	}
	if len(c.SecretKey) != 32 {
		t.Errorf("SecretKey len = %d", len(c.SecretKey))
	}
	if c.HTTPS() {
		t.Error("HTTPS() should be false without cert")
	}
}

func TestImplicitPorts(t *testing.T) {
	c, err := Load(env(with("GOLUB_IMAP_TLS", "implicit", "GOLUB_SMTP_TLS", "implicit")), publicLookup)
	if err != nil {
		t.Fatal(err)
	}
	if c.IMAP.Port != 993 || c.SMTP.Port != 465 {
		t.Errorf("ports = %d %d", c.IMAP.Port, c.SMTP.Port)
	}
}

func TestValid(t *testing.T) {
	cases := map[string]map[string]string{
		"explicit ports": with("GOLUB_IMAP_PORT", "1143", "GOLUB_SMTP_PORT", "1587"),
		"tls pair":       with("GOLUB_TLS_CERT", "/c.pem", "GOLUB_TLS_KEY", "/k.pem"),
		"trust proxy":    with("GOLUB_TRUST_PROXY", "10.0.0.0/8, 192.168.1.1/32"),
		"sso":            with("GOLUB_SSO_SECRET", strings.Repeat("s", 32), "GOLUB_MASTER_USER", "admin", "GOLUB_MASTER_PASSWORD", "pw"),
		"locale sr":      with("GOLUB_DEFAULT_LOCALE", "sr-Latn"),
		"theme dark":     with("GOLUB_DEFAULT_THEME", "dark"),
		"ttl":            with("GOLUB_SESSION_TTL", "30m"),
		"log debug":      with("GOLUB_LOG_LEVEL", "debug"),
	}
	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(env(m), publicLookup); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInvalid(t *testing.T) {
	cases := []struct {
		name, wantVar string
		env           map[string]string
	}{
		{"missing imap host", "GOLUB_IMAP_HOST", map[string]string{"GOLUB_SECRET_KEY": testKey}},
		{"missing secret", "GOLUB_SECRET_KEY", map[string]string{"GOLUB_IMAP_HOST": "x"}},
		{"secret not base64", "GOLUB_SECRET_KEY", with("GOLUB_SECRET_KEY", "!!!")},
		{"secret too short", "GOLUB_SECRET_KEY", with("GOLUB_SECRET_KEY", base64.StdEncoding.EncodeToString([]byte("short")))},
		{"cert only", "GOLUB_TLS_KEY", with("GOLUB_TLS_CERT", "/c.pem")},
		{"key only", "GOLUB_TLS_CERT", with("GOLUB_TLS_KEY", "/k.pem")},
		{"bad cidr", "GOLUB_TRUST_PROXY", with("GOLUB_TRUST_PROXY", "nope")},
		{"bad imap port", "GOLUB_IMAP_PORT", with("GOLUB_IMAP_PORT", "70000")},
		{"bad smtp port", "GOLUB_SMTP_PORT", with("GOLUB_SMTP_PORT", "abc")},
		{"bad imap tls", "GOLUB_IMAP_TLS", with("GOLUB_IMAP_TLS", "maybe")},
		{"bad smtp tls", "GOLUB_SMTP_TLS", with("GOLUB_SMTP_TLS", "insecure")},
		{"imap none on public host", "GOLUB_IMAP_TLS", with("GOLUB_IMAP_TLS", "none")},
		{"smtp none on public host", "GOLUB_SMTP_TLS", with("GOLUB_SMTP_TLS", "none")},
		{"sso short secret", "GOLUB_SSO_SECRET", with("GOLUB_SSO_SECRET", "short", "GOLUB_MASTER_USER", "a", "GOLUB_MASTER_PASSWORD", "b")},
		{"sso no master user", "GOLUB_MASTER_USER", with("GOLUB_SSO_SECRET", strings.Repeat("s", 32), "GOLUB_MASTER_PASSWORD", "b")},
		{"sso no master password", "GOLUB_MASTER_PASSWORD", with("GOLUB_SSO_SECRET", strings.Repeat("s", 32), "GOLUB_MASTER_USER", "a")},
		{"bad ttl", "GOLUB_SESSION_TTL", with("GOLUB_SESSION_TTL", "forever")},
		{"negative ttl", "GOLUB_SESSION_TTL", with("GOLUB_SESSION_TTL", "-1h")},
		{"bad locale", "GOLUB_DEFAULT_LOCALE", with("GOLUB_DEFAULT_LOCALE", "sr-Cyrl")},
		{"bad theme", "GOLUB_DEFAULT_THEME", with("GOLUB_DEFAULT_THEME", "blue")},
		{"bad log level", "GOLUB_LOG_LEVEL", with("GOLUB_LOG_LEVEL", "loud")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(env(tc.env), publicLookup)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantVar) {
				t.Errorf("error %q does not name %s", err, tc.wantVar)
			}
		})
	}
}

func TestPlainOnPrivateHost(t *testing.T) {
	m := with("GOLUB_IMAP_TLS", "none", "GOLUB_SMTP_TLS", "none")
	c, err := Load(env(m), privateLookup)
	if err != nil {
		t.Fatal(err)
	}
	if c.IMAP.Port != 143 || c.SMTP.Port != 587 {
		t.Errorf("ports = %d %d", c.IMAP.Port, c.SMTP.Port)
	}
}

func TestPlainRejectsMixedResolution(t *testing.T) {
	mixed := func(string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("198.51.100.1")}, nil
	}
	if _, err := Load(env(with("GOLUB_IMAP_TLS", "none")), mixed); err == nil {
		t.Fatal("expected error when any address is public")
	}
}

func TestPlainLookupFailure(t *testing.T) {
	fail := func(string) ([]netip.Addr, error) { return nil, errors.New("no such host") }
	_, err := Load(env(with("GOLUB_IMAP_TLS", "none")), fail)
	if err == nil || !strings.Contains(err.Error(), "GOLUB_IMAP_HOST") {
		t.Fatalf("err = %v", err)
	}
}

func TestErrorsAreJoined(t *testing.T) {
	_, err := Load(env(map[string]string{"GOLUB_LOG_LEVEL": "x"}), publicLookup)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, v := range []string{"GOLUB_IMAP_HOST", "GOLUB_SECRET_KEY", "GOLUB_LOG_LEVEL"} {
		if !strings.Contains(err.Error(), v) {
			t.Errorf("missing %s in %q", v, err)
		}
	}
}

func TestBranding(t *testing.T) {
	dir := t.TempDir()
	logo := filepath.Join(dir, "logo.svg")
	if err := os.WriteFile(logo, []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(env(with("GOLUB_BRAND_NAME", "Host Mail", "GOLUB_BRAND_ACCENT", "#FF6600", "GOLUB_BRAND_LOGO", logo)), publicLookup)
	if err != nil {
		t.Fatal(err)
	}
	if c.BrandName != "Host Mail" || c.BrandAccent != "#ff6600" || c.BrandLogoType != "image/svg+xml" || len(c.BrandLogo) == 0 {
		t.Fatalf("%+v", c)
	}
	d, _ := Load(env(base()), publicLookup)
	if d.BrandName != "Golub" || d.BrandLogo != nil || d.BrandAccent != "" {
		t.Fatalf("defaults %+v", d)
	}

	big := filepath.Join(dir, "big.png")
	_ = os.WriteFile(big, make([]byte, 600<<10), 0o600)
	txt := filepath.Join(dir, "logo.txt")
	_ = os.WriteFile(txt, []byte("x"), 0o600)
	for name, kv := range map[string][]string{
		"accent":       {"GOLUB_BRAND_ACCENT", "orange"},
		"name control": {"GOLUB_BRAND_NAME", "a\nb"},
		"name long":    {"GOLUB_BRAND_NAME", strings.Repeat("x", 65)},
		"logo missing": {"GOLUB_BRAND_LOGO", filepath.Join(dir, "nope.png")},
		"logo big":     {"GOLUB_BRAND_LOGO", big},
		"logo type":    {"GOLUB_BRAND_LOGO", txt},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(env(with(kv...)), publicLookup); err == nil || !strings.Contains(err.Error(), kv[0]) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
