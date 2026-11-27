// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

var testFS = fstest.MapFS{
	"index.html":    {Data: []byte("<!doctype html><title>golub</title>")},
	"assets/app.js": {Data: []byte("console.log(1)")},
}

func do(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	h := New(Options{Version: "1.2.3", Static: testFS})
	rec := do(h, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q", ct)
	}
	var body struct{ Status, Version string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Version != "1.2.3" {
		t.Errorf("body = %+v", body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("healthz must not be cached")
	}
}

func TestHealthzMethod(t *testing.T) {
	h := New(Options{Static: testFS})
	if rec := do(h, http.MethodPost, "/healthz"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("code = %d", rec.Code)
	}
}

func TestStatic(t *testing.T) {
	h := New(Options{Static: testFS})
	rec := do(h, http.MethodGet, "/assets/app.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Errorf("asset: %d %q", rec.Code, rec.Body.String())
	}
	rec = do(h, http.MethodGet, "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "golub") {
		t.Errorf("index: %d %q", rec.Code, rec.Body.String())
	}
}

func TestSPAFallback(t *testing.T) {
	h := New(Options{Static: testFS})
	rec := do(h, http.MethodGet, "/mail/INBOX")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "golub") {
		t.Errorf("fallback: %d %q", rec.Code, rec.Body.String())
	}
	if rec := do(h, http.MethodGet, "/assets/missing.js"); rec.Code != http.StatusNotFound {
		t.Errorf("missing asset code = %d", rec.Code)
	}
}

func TestUnknownAPI(t *testing.T) {
	h := New(Options{Static: testFS})
	rec := do(h, http.MethodGet, "/api/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d", rec.Code)
	}
	var body struct{ Error string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "api.not_found" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := New(Options{Static: testFS})
	rec := do(h, http.MethodGet, "/")
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "no-referrer",
		"X-Frame-Options":        "DENY",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("%s = %q", k, got)
		}
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing CSP")
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must not be sent over plain HTTP")
	}
}

func TestHSTS(t *testing.T) {
	h := New(Options{Static: testFS, HTTPS: true})
	rec := do(h, http.MethodGet, "/healthz")
	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=31536000" {
		t.Errorf("HSTS = %q", got)
	}
}
