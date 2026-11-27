// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/stefanpejcic/golub/internal/config"
)

func peerAddr(r *http.Request) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	a, err := netip.ParseAddr(host)
	return a.Unmap(), err == nil
}

func trusted(a netip.Addr, trust []netip.Prefix) bool {
	for _, p := range trust {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// isSecure reports whether the browser reached us over HTTPS.
func isSecure(r *http.Request, https bool, trust []netip.Prefix) bool {
	if https || r.TLS != nil {
		return true
	}
	a, ok := peerAddr(r)
	return ok && trusted(a, trust) && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// clientIP walks X-Forwarded-For from the right, skipping trusted proxies.
func clientIP(r *http.Request, trust []netip.Prefix) string {
	a, ok := peerAddr(r)
	if !ok {
		return r.RemoteAddr
	}
	if !trusted(a, trust) {
		return a.String()
	}
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		h, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		if !trusted(h.Unmap(), trust) {
			return h.Unmap().String()
		}
		a = h.Unmap()
	}
	return a.String()
}

// localeFromAccept picks the best supported locale from Accept-Language.
func localeFromAccept(header, def string) string {
	best, bestQ := "", -1.0
	for _, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				q = f
			}
		}
		if l := supportedLocale(tag); l != "" && q > bestQ {
			best, bestQ = l, q
		}
	}
	if best == "" {
		return def
	}
	return best
}

// supportedLocale maps a BCP 47 tag to a shipped locale; Serbian means Latin only.
func supportedLocale(tag string) string {
	tag = strings.TrimSpace(tag)
	if slices.Contains(config.Locales, tag) {
		return tag
	}
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(tag, "_", "-")), "-")
	switch parts[0] {
	case "en":
		return "en"
	case "sr":
		if slices.Contains(parts[1:], "cyrl") {
			return ""
		}
		return "sr-Latn"
	}
	return ""
}
