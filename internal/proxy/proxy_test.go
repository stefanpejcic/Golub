// SPDX-License-Identifier: GPL-3.0-or-later

package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

var png = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func newProxy(allowLoopback bool) *Proxy {
	p := New([]byte("0123456789abcdef0123456789abcdef"))
	if allowLoopback {
		p.allow = func(a netip.Addr) bool { return a.IsLoopback() || Public(a) }
	}
	return p
}

func get(p *Proxy, target string, sign bool) *httptest.ResponseRecorder {
	path := "/proxy?u=" + url.QueryEscape(target) + "&s=bad"
	if sign {
		path = p.URL(target)
	}
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func origin(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s.URL
}

func TestPublic(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1",
		"0.0.0.0", "::1", "fe80::1", "fc00::1", "224.0.0.1", "::ffff:127.0.0.1", "198.18.0.1", "64:ff9b::a00:1"}
	for _, s := range blocked {
		if Public(netip.MustParseAddr(s)) {
			t.Errorf("%s treated as public", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "2606:4700::1111"} {
		if !Public(netip.MustParseAddr(s)) {
			t.Errorf("%s treated as private", s)
		}
	}
}

func TestSignature(t *testing.T) {
	p := newProxy(true)
	u := origin(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(png) })
	if rec := get(p, u, false); rec.Code != http.StatusForbidden {
		t.Fatalf("unsigned: %d", rec.Code)
	}
	if rec := get(p, u, true); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), png) {
		t.Fatalf("signed: %d", rec.Code)
	}
}

func TestServesImageSafely(t *testing.T) {
	p := newProxy(true)
	u := origin(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Set-Cookie", "tracker=1")
		_, _ = w.Write(png)
	})
	rec := get(p, u, true)
	h := rec.Header()
	if h.Get("Content-Type") != "image/png" || h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Set-Cookie") != "" || !strings.Contains(h.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Errorf("headers = %v", h)
	}
}

func TestSSRF(t *testing.T) {
	p := newProxy(false)
	u := origin(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(png) })
	port := u[strings.LastIndex(u, ":"):]
	for _, target := range []string{u, "http://localhost" + port + "/", "http://10.0.0.1/x.png", "http://[::1]" + port + "/"} {
		if rec := get(p, target, true); rec.Code != http.StatusBadGateway {
			t.Errorf("%s: %d", target, rec.Code)
		}
	}
}

func TestRedirectToPrivateBlocked(t *testing.T) {
	p := newProxy(true)
	inner := origin(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(png) })
	outer := origin(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, inner, http.StatusFound) })
	p.allow = func(a netip.Addr) bool { return Public(a) || a.IsLoopback() }
	if rec := get(p, outer, true); rec.Code != http.StatusOK {
		t.Fatalf("allowed redirect: %d", rec.Code)
	}
	p = newProxy(false)
	calls := 0
	p.allow = func(netip.Addr) bool { calls++; return calls == 1 }
	if rec := get(p, outer, true); rec.Code != http.StatusBadGateway {
		t.Fatalf("redirect to blocked address: %d", rec.Code)
	}
}

func TestRejects(t *testing.T) {
	p := newProxy(true)
	cases := map[string]http.HandlerFunc{
		"html": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<script>alert(1)</script>"))
		},
		"svg": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`))
		},
		"lying type": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("<html><script>alert(1)</script></html>"))
		},
		"too big": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(append(png, make([]byte, MaxSize)...))
		},
		"404": func(w http.ResponseWriter, _ *http.Request) { http.NotFound(w, nil) },
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := get(p, origin(t, h), true); rec.Code == http.StatusOK {
				t.Fatalf("served %s", name)
			}
		})
	}
	for _, bad := range []string{"file:///etc/passwd", "ftp://x/y", "gopher://x", "http://", "not a url"} {
		if rec := get(p, bad, true); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: %d", bad, rec.Code)
		}
	}
}

func TestOneClickUnsubscribe(t *testing.T) {
	p := newProxy(true)
	var got string
	u := origin(t, func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = r.Method + " " + r.Header.Get("Content-Type") + " " + string(b)
		if r.Header.Get("Cookie") != "" || r.Header.Get("Referer") != "" {
			got = "leaked identity"
		}
	})
	if err := p.Unsubscribe(context.Background(), u+"/u/abc"); err != nil {
		t.Fatal(err)
	}
	if got != "POST application/x-www-form-urlencoded List-Unsubscribe=One-Click" {
		t.Fatalf("got %q", got)
	}
	if err := newProxy(false).Unsubscribe(context.Background(), u+"/u/abc"); err == nil {
		t.Error("loopback target must be refused")
	}
	if err := p.Unsubscribe(context.Background(), "ftp://x/y"); err == nil {
		t.Error("non-http target must be refused")
	}
}
