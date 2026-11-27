// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"slices"
	"time"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/config"
	"github.com/stefanpejcic/golub/internal/imapx"
)

// Cookie names; the pref cookies are readable by the pre-paint script and hold no secrets.
const (
	SessionCookie = "golub_session"
	LangCookie    = "golub_lang"
	ThemeCookie   = "golub_theme"
)

const prefCookieMaxAge = 365 * 24 * 3600

type authAPI struct {
	o       Options
	limiter *failLimiter
	now     func() time.Time
}

func newAuthAPI(o Options) *authAPI {
	return &authAPI{o: o, limiter: newFailLimiter(), now: time.Now}
}

type sessionView struct {
	Email  string `json:"email"`
	Mode   string `json:"mode"`
	Locale string `json:"locale"`
	Theme  string `json:"theme"`
}

func view(s auth.Session) sessionView {
	return sessionView{Email: s.Email, Mode: s.Mode, Locale: s.Locale, Theme: s.Theme}
}

// current returns the session from the request cookie, if valid.
func (a *authAPI) current(r *http.Request) (auth.Session, bool) {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return auth.Session{}, false
	}
	s, err := a.o.Codec.Decode(c.Value, a.now())
	return s, err == nil
}

func (a *authAPI) session(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s, ok := a.current(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.required")
		return
	}
	writeJSON(w, http.StatusOK, view(s))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Locale   string `json:"locale"`
	Theme    string `json:"theme"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "request.unsupported_media_type")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return false
	}
	return true
}

func (a *authAPI) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	ip := clientIP(r, a.o.TrustProxy)
	if a.limiter.blocked(ip) {
		writeError(w, http.StatusTooManyRequests, "auth.rate_limited")
		return
	}
	if !auth.ValidEmail(req.Email) {
		a.limiter.fail(ip)
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials")
		return
	}
	if err := a.o.Mail.Check(r.Context(), req.Email, req.Password); err != nil {
		if errors.Is(err, imapx.ErrInvalidCredentials) {
			a.limiter.fail(ip)
			slog.Info("login failed", "ip", ip)
			writeError(w, http.StatusUnauthorized, "auth.invalid_credentials")
			return
		}
		slog.Error("imap unavailable during login", "err", err)
		writeError(w, http.StatusBadGateway, "mail.unavailable")
		return
	}
	s := auth.Session{
		Email:    req.Email,
		Mode:     auth.ModePassword,
		Password: req.Password,
		Locale:   a.pickLocale(req.Locale, r),
		Theme:    a.pickTheme(req.Theme),
	}
	if !a.issue(w, r, &s) {
		return
	}
	writeJSON(w, http.StatusOK, view(s))
}

func (a *authAPI) logout(w http.ResponseWriter, r *http.Request) {
	a.clearSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (a *authAPI) clearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure follows the request scheme so plain-HTTP setups behind a proxy still work
		Name: SessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isSecure(r, a.o.HTTPS, a.o.TrustProxy),
	})
}

type prefsRequest struct {
	Locale string `json:"locale"`
	Theme  string `json:"theme"`
}

func (a *authAPI) prefs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var req prefsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if (req.Locale != "" && !slices.Contains(config.Locales, req.Locale)) ||
		(req.Theme != "" && !slices.Contains(config.Themes, req.Theme)) {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	s, ok := a.current(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth.required")
		return
	}
	if req.Locale != "" {
		s.Locale = req.Locale
	}
	if req.Theme != "" {
		s.Theme = req.Theme
	}
	if !a.issue(w, r, &s) {
		return
	}
	writeJSON(w, http.StatusOK, view(s))
}

func (a *authAPI) sso(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	fail := func(code string) {
		http.Redirect(w, r, "/?error="+code, http.StatusSeeOther)
	}
	claims, err := a.o.SSO.Verify(r.URL.Query().Get("token"))
	if err != nil {
		slog.Info("sso token rejected", "err", err, "ip", clientIP(r, a.o.TrustProxy))
		fail("auth.sso_invalid")
		return
	}
	if err := a.o.Mail.Check(r.Context(), claims.Email+"*"+a.o.MasterUser, a.o.MasterPassword); err != nil {
		if errors.Is(err, imapx.ErrInvalidCredentials) {
			slog.Error("sso master login rejected", "email", claims.Email)
			fail("auth.sso_invalid")
			return
		}
		slog.Error("imap unavailable during sso", "err", err)
		fail("mail.unavailable")
		return
	}
	s := auth.Session{
		Email:  claims.Email,
		Mode:   auth.ModeSSO,
		Locale: a.pickLocale(claims.Lang, r),
		Theme:  a.pickTheme(claims.Theme),
	}
	if !a.issue(w, r, &s) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *authAPI) pickLocale(explicit string, r *http.Request) string {
	if slices.Contains(config.Locales, explicit) {
		return explicit
	}
	if c, err := r.Cookie(LangCookie); err == nil && slices.Contains(config.Locales, c.Value) {
		return c.Value
	}
	return localeFromAccept(r.Header.Get("Accept-Language"), a.o.DefaultLocale)
}

func (a *authAPI) pickTheme(explicit string) string {
	if slices.Contains(config.Themes, explicit) {
		return explicit
	}
	return a.o.DefaultTheme
}

// issue sets the session and pref cookies; it writes an error response on failure.
func (a *authAPI) issue(w http.ResponseWriter, r *http.Request, s *auth.Session) bool {
	if s.Expires == 0 {
		s.Expires = a.now().Add(a.o.SessionTTL).Unix()
	}
	v, err := a.o.Codec.Encode(*s)
	if err != nil {
		slog.Error("session encode", "err", err)
		writeError(w, http.StatusInternalServerError, "server.error")
		return false
	}
	secure := isSecure(r, a.o.HTTPS, a.o.TrustProxy)
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure follows the request scheme
		Name: SessionCookie, Value: v, Path: "/", Expires: time.Unix(s.Expires, 0),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure,
	})
	for name, val := range map[string]string{LangCookie: s.Locale, ThemeCookie: s.Theme} {
		http.SetCookie(w, &http.Cookie{ //nolint:gosec // read by the pre-paint theme script, holds no secrets
			Name: name, Value: val, Path: "/", MaxAge: prefCookieMaxAge,
			SameSite: http.SameSiteLaxMode, Secure: secure,
		})
	}
	return true
}
