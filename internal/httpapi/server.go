// SPDX-License-Identifier: GPL-3.0-or-later

// Package httpapi wires the HTTP routes, JSON API and static frontend.
package httpapi

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Options configures the HTTP handler.
type Options struct {
	Version string
	Static  fs.FS
	HTTPS   bool
}

const appCSP = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"frame-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'"

// New returns the root handler.
func New(o Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": o.Version})
	})
	mux.Handle("/", fallback(spa(o.Static)))
	return securityHeaders(o.HTTPS, mux)
}

func securityHeaders(https bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", appCSP)
		if https {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// fallback handles everything no explicit route matched.
func fallback(spa http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			writeError(w, http.StatusNotFound, "api.not_found")
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		default:
			spa.ServeHTTP(w, r)
		}
	})
}

// spa serves built assets and falls back to index.html for client-side routes.
func spa(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == "index.html" {
			serveIndex(w, r, static)
			return
		}
		if _, err := fs.Stat(static, name); err == nil {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		} else if !errors.Is(err, fs.ErrNotExist) || path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r, static)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, static fs.FS) {
	b, err := fs.ReadFile(static, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends a stable error code that the client translates.
func writeError(w http.ResponseWriter, code int, errCode string) {
	writeJSON(w, code, map[string]string{"error": errCode})
}
