// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticPrecompressed(t *testing.T) {
	static := fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html>")},
		"assets/app.js":    {Data: []byte("console.log('plain')")},
		"assets/app.js.br": {Data: []byte("BROTLI")},
		"assets/app.js.gz": {Data: []byte("GZIP")},
	}
	h := New(Options{Static: static})
	get := func(enc string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/assets/app.js", nil)
		if enc != "" {
			r.Header.Set("Accept-Encoding", enc)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	rec := get("gzip, deflate, br")
	if rec.Body.String() != "BROTLI" || rec.Header().Get("Content-Encoding") != "br" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("br: %q %v", rec.Body.String(), rec.Header())
	}
	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("missing Vary")
	}
	if rec = get("gzip"); rec.Body.String() != "GZIP" || rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("gz: %q %v", rec.Body.String(), rec.Header())
	}
	if rec = get(""); rec.Body.String() != "console.log('plain')" || rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("identity: %q %v", rec.Body.String(), rec.Header())
	}
}

func TestAPIGzip(t *testing.T) {
	h, _ := newMailServer(t, 30)
	c := login(t, h)
	r := httptest.NewRequest("GET", "/api/messages?folder=INBOX&size=50", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.AddCookie(c)
	rec := serve(h, r)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("not compressed: %v", rec.Header())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.Contains(string(body), "Message 030") {
		t.Fatalf("%s", body)
	}

	small := httptest.NewRequest("GET", "/healthz", nil)
	small.Header.Set("Accept-Encoding", "gzip")
	if rec := serve(h, small); rec.Header().Get("Content-Encoding") != "" {
		t.Error("tiny responses should not be compressed")
	}
	plain := httptest.NewRequest("GET", "/api/messages?folder=INBOX", nil)
	plain.AddCookie(c)
	if rec := serve(h, plain); rec.Header().Get("Content-Encoding") != "" || rec.Code != http.StatusOK {
		t.Error("clients without gzip must get plain responses")
	}
}
