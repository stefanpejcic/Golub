// SPDX-License-Identifier: GPL-3.0-or-later

// Package httpapi wires the HTTP routes, JSON API and static frontend.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/netip"
	"path"
	"strings"
	"time"

	"github.com/stefanpejcic/golub/internal/auth"
)

// MailAuth verifies IMAP credentials.
type MailAuth interface {
	Check(ctx context.Context, user, pass string) error
}

// Options configures the HTTP handler.
type Options struct {
	Version string
	Static  fs.FS
	HTTPS   bool

	Codec          *auth.Codec
	Mail           MailAuth
	Pool           MailPool
	Proxy          ImageProxy
	SMTP           MailSender
	Watcher        Watcher
	SSO            *auth.SSOVerifier
	MasterUser     string
	MasterPassword string
	SessionTTL     time.Duration
	DefaultLocale  string
	DefaultTheme   string
	TrustProxy     []netip.Prefix
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
	if o.Codec != nil {
		a := newAuthAPI(o)
		mux.HandleFunc("GET /api/session", a.session)
		mux.HandleFunc("POST /api/login", a.login)
		mux.HandleFunc("POST /api/logout", a.logout)
		mux.HandleFunc("POST /api/prefs", a.prefs)
		if o.Pool != nil {
			mux.HandleFunc("GET /api/folders", a.withSession(a.folders))
			mux.HandleFunc("GET /api/messages", a.withSession(a.messages))
			mux.HandleFunc("GET /api/message", a.withSession(a.message))
			mux.HandleFunc("GET /api/message/body", a.withSession(a.messageBody))
			mux.HandleFunc("GET /api/message/part", a.withSession(a.messagePart))
			mux.HandleFunc("POST /api/messages/flag", a.withSession(a.setFlag))
			mux.HandleFunc("POST /api/messages/move", a.withSession(a.move))
			mux.HandleFunc("POST /api/messages/copy", a.withSession(a.copyTo))
			mux.HandleFunc("GET /api/message/raw", a.withSession(a.messageRaw))
			mux.HandleFunc("POST /api/messages/delete", a.withSession(a.remove))
			mux.HandleFunc("POST /api/messages/archive", a.withSession(a.toSpecial("archive", "Archive")))
			mux.HandleFunc("POST /api/messages/spam", a.withSession(a.toSpecial("junk", "Junk")))
			mux.HandleFunc("GET /api/settings", a.withSession(a.getSettings))
			mux.HandleFunc("GET /api/quota", a.withSession(a.quota))
			mux.HandleFunc("GET /api/compose/source", a.withSession(a.composeSourceFor))
			mux.HandleFunc("POST /api/compose/draft", a.withSession(a.composeDraft))
			if o.Watcher != nil {
				mux.HandleFunc("GET /api/events", a.withSession(a.events))
			}
			mux.HandleFunc("POST /api/message/unsubscribe", a.withSession(a.unsubscribe))
			if o.SMTP != nil {
				mux.HandleFunc("POST /api/message/receipt", a.withSession(a.receipt))
				mux.HandleFunc("POST /api/compose/send", a.withSession(a.composeSend))
			}
			mux.HandleFunc("PUT /api/settings", a.withSession(a.putSettings))
			for _, op := range []string{"create", "rename", "delete", "empty", "subscribe", "markread"} {
				mux.HandleFunc("POST /api/folders/"+op, a.withSession(a.folderOp(op)))
			}
		}
		if o.SSO != nil {
			mux.HandleFunc("GET /sso", a.sso)
		} else {
			mux.HandleFunc("GET /sso", http.NotFound)
		}
	}
	if o.Proxy != nil {
		mux.Handle("GET /proxy", o.Proxy)
	}
	mux.Handle("/", fallback(spa(o.Static)))
	return securityHeaders(o.HTTPS, http.NewCrossOriginProtection().Handler(gzipAPI(timed(mux))))
}

func securityHeaders(https bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", appCSP)
		h.Set("X-Robots-Tag", "noindex, nofollow, noarchive")
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
			if precompressed(w, r, static, name) {
				return
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
