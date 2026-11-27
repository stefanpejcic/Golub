// SPDX-License-Identifier: GPL-3.0-or-later

// Package proxy fetches remote images for HTML mail with an SSRF guard.
package proxy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"
)

// Limits for proxied images.
const (
	MaxSize      = 5 << 20
	fetchTimeout = 10 * time.Second
	maxRedirects = 3
)

var errBlocked = errors.New("proxy: destination not allowed")

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
}

// Public reports whether a is a globally routable unicast address.
func Public(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsValid() || a.IsLoopback() || a.IsPrivate() || a.IsUnspecified() || a.IsMulticast() ||
		a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() || a.IsInterfaceLocalMulticast() {
		return false
	}
	for _, p := range blockedPrefixes {
		if p.Contains(a) {
			return false
		}
	}
	return a.IsGlobalUnicast()
}

// Proxy serves signed /proxy URLs.
type Proxy struct {
	key    []byte
	allow  func(netip.Addr) bool
	client *http.Client
}

// New returns a proxy whose URLs are signed with a key derived from secret.
func New(secret []byte) *Proxy {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte("golub-proxy-v1"))
	p := &Proxy{key: m.Sum(nil), allow: Public}
	dialer := &net.Dialer{
		Timeout: fetchTimeout,
		// Checked on the address actually dialed, so DNS rebinding and redirects cannot slip through.
		Control: func(_, address string, _ syscall.RawConn) error {
			ap, err := netip.ParseAddrPort(address)
			if err != nil || !p.allow(ap.Addr()) {
				return errBlocked
			}
			return nil
		},
	}
	p.client = &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   fetchTimeout,
			ResponseHeaderTimeout: fetchTimeout,
			MaxIdleConns:          16,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("proxy: too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errBlocked
			}
			return nil
		},
	}
	return p
}

func (p *Proxy) sign(target string) string {
	m := hmac.New(sha256.New, p.key)
	m.Write([]byte(target))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil)[:18])
}

// URL returns the same-origin URL that loads target through the proxy.
func (p *Proxy) URL(target string) string {
	return "/proxy?u=" + url.QueryEscape(target) + "&s=" + p.sign(target)
}

var rasterTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true, "image/bmp": true, "image/x-icon": true,
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	target := r.URL.Query().Get("u")
	if !hmac.Equal([]byte(r.URL.Query().Get("s")), []byte(p.sign(target))) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}

	body, err := p.fetch(r.Context(), u.String())
	if err != nil {
		slog.Debug("proxy fetch failed", "host", u.Hostname(), "err", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	ct := http.DetectContentType(body)
	if !rasterTypes[ct] {
		http.Error(w, "not an image", http.StatusBadGateway)
		return
	}
	h.Set("Content-Type", ct)
	h.Set("Cache-Control", "private, max-age=86400")
	h.Set("Content-Disposition", "inline")
	_, _ = w.Write(body) //nolint:gosec // only sniffed raster images get here, with nosniff and a sandbox CSP
}

func (p *Proxy) fetch(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil) //nolint:gosec // fetching remote URLs is the point; the dialer rejects non-public addresses
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Golub image proxy")
	req.Header.Set("Accept", "image/*")
	resp, err := p.client.Do(req) //nolint:gosec // see above
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxSize {
		return nil, errors.New("too large")
	}
	return body, nil
}
