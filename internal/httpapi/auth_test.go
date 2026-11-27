// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/imapx"
)

const testSSOSecret = "0123456789abcdef0123456789abcdef"

type fakeMail struct {
	users map[string]string
	calls []string
	down  bool
}

func (f *fakeMail) Check(_ context.Context, user, pass string) error {
	f.calls = append(f.calls, user)
	if f.down {
		return imapx.ErrUnavailable
	}
	if p, ok := f.users[user]; ok && p == pass {
		return nil
	}
	return imapx.ErrInvalidCredentials
}

func newAuthServer(t *testing.T, mutate ...func(*Options)) (http.Handler, *fakeMail) {
	t.Helper()
	codec := testCodec(t)
	fm := &fakeMail{users: map[string]string{
		"alice@golub.test":             "secret",
		"alice@golub.test*golubmaster": "masterpw",
	}}
	o := Options{
		Version:        "t",
		Static:         testFS,
		Codec:          codec,
		Mail:           fm,
		SSO:            auth.NewSSOVerifier(testSSOSecret),
		MasterUser:     "golubmaster",
		MasterPassword: "masterpw",
		SessionTTL:     time.Hour,
		DefaultLocale:  "en",
		DefaultTheme:   "system",
	}
	for _, m := range mutate {
		m(&o)
	}
	return New(o), fm
}

func testCodec(t *testing.T) *auth.Codec {
	t.Helper()
	c, err := auth.NewCodec(bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ssoToken(t *testing.T) string {
	t.Helper()
	return auth.SignSSO(testSSOSecret, auth.SSOClaims{Email: "alice@golub.test", Expires: time.Now().Add(30 * time.Second).Unix(), Nonce: fmt.Sprintf("nonce-%d", time.Now().UnixNano())})
}

func jsonReq(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func cookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var b struct{ Error string }
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	return b.Error
}

func login(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	rec := serve(h, jsonReq("POST", "/api/login", `{"email":"alice@golub.test","password":"secret"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	c := cookie(rec, SessionCookie)
	if c == nil {
		t.Fatal("no session cookie")
	}
	return c
}

func TestLoginLogout(t *testing.T) {
	h, _ := newAuthServer(t)

	rec := serve(h, httptest.NewRequest("GET", "/api/session", nil))
	if rec.Code != http.StatusUnauthorized || errCode(t, rec) != "auth.required" {
		t.Fatalf("anon session: %d %s", rec.Code, rec.Body)
	}

	c := login(t, h)
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure || c.Path != "/" {
		t.Errorf("cookie attrs: %+v", c)
	}

	r := httptest.NewRequest("GET", "/api/session", nil)
	r.AddCookie(c)
	rec = serve(h, r)
	var s struct{ Email, Locale, Theme, Mode string }
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if rec.Code != 200 || s.Email != "alice@golub.test" || s.Mode != "password" || s.Locale != "en" || s.Theme != "system" {
		t.Fatalf("session: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("password leaked in session response")
	}

	r = jsonReq("POST", "/api/logout", "")
	r.AddCookie(c)
	rec = serve(h, r)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rec.Code)
	}
	if cl := cookie(rec, SessionCookie); cl == nil || cl.MaxAge >= 0 {
		t.Errorf("logout cookie: %+v", cl)
	}
}

func TestLoginErrors(t *testing.T) {
	cases := []struct {
		name, body, code string
		status           int
		down             bool
	}{
		{"wrong password", `{"email":"alice@golub.test","password":"nope"}`, "auth.invalid_credentials", 401, false},
		{"bad json", `{`, "request.invalid", 400, false},
		{"missing password", `{"email":"alice@golub.test"}`, "request.invalid", 400, false},
		{"master injection", `{"email":"alice@golub.test*golubmaster","password":"masterpw"}`, "auth.invalid_credentials", 401, false},
		{"mail down", `{"email":"alice@golub.test","password":"secret"}`, "mail.unavailable", 502, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, fm := newAuthServer(t)
			fm.down = tc.down
			rec := serve(h, jsonReq("POST", "/api/login", tc.body))
			if rec.Code != tc.status || errCode(t, rec) != tc.code {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
			for _, u := range fm.calls {
				if strings.Contains(u, "*") {
					t.Fatal("master login attempted via normal login")
				}
			}
		})
	}
}

func TestLoginRequiresJSON(t *testing.T) {
	h, _ := newAuthServer(t)
	r := httptest.NewRequest("POST", "/api/login", strings.NewReader("email=a&password=b"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if rec := serve(h, r); rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestCrossOriginPostRejected(t *testing.T) {
	h, _ := newAuthServer(t)
	r := jsonReq("POST", "/api/login", `{"email":"alice@golub.test","password":"secret"}`)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	if rec := serve(h, r); rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	h, _ := newAuthServer(t)
	var rec *httptest.ResponseRecorder
	for i := 0; i < loginMaxFailures+1; i++ {
		rec = serve(h, jsonReq("POST", "/api/login", `{"email":"alice@golub.test","password":"bad"}`))
	}
	if rec.Code != http.StatusTooManyRequests || errCode(t, rec) != "auth.rate_limited" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestLoginLocaleAndTheme(t *testing.T) {
	h, _ := newAuthServer(t)
	r := jsonReq("POST", "/api/login", `{"email":"alice@golub.test","password":"secret","theme":"dark"}`)
	r.Header.Set("Accept-Language", "sr-RS,sr;q=0.9,en;q=0.5")
	rec := serve(h, r)
	var s struct{ Locale, Theme string }
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if s.Locale != "sr-Latn" || s.Theme != "dark" {
		t.Fatalf("got %+v", s)
	}
	if c := cookie(rec, LangCookie); c == nil || c.Value != "sr-Latn" || c.HttpOnly {
		t.Errorf("lang cookie %+v", c)
	}
	if c := cookie(rec, ThemeCookie); c == nil || c.Value != "dark" {
		t.Errorf("theme cookie %+v", c)
	}
}

func TestPrefsUpdateSession(t *testing.T) {
	h, _ := newAuthServer(t)
	c := login(t, h)
	r := jsonReq("POST", "/api/prefs", `{"locale":"sr-Latn","theme":"light"}`)
	r.AddCookie(c)
	rec := serve(h, r)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	nc := cookie(rec, SessionCookie)
	r = httptest.NewRequest("GET", "/api/session", nil)
	r.AddCookie(nc)
	var s struct{ Locale, Theme string }
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &s)
	if s.Locale != "sr-Latn" || s.Theme != "light" {
		t.Fatalf("got %+v", s)
	}

	r = jsonReq("POST", "/api/prefs", `{"theme":"neon"}`)
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 400 {
		t.Fatalf("invalid theme accepted: %d", rec.Code)
	}
}

func TestSSO(t *testing.T) {
	h, fm := newAuthServer(t)
	tok := auth.SignSSO(testSSOSecret, auth.SSOClaims{Email: "alice@golub.test", Expires: time.Now().Add(30 * time.Second).Unix(), Nonce: "nonce-0123456789", Lang: "sr-Latn", Theme: "dark"})
	rec := serve(h, httptest.NewRequest("GET", "/sso?token="+url.QueryEscape(tok), nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("sso response must not be cached")
	}
	if len(fm.calls) != 1 || fm.calls[0] != "alice@golub.test*golubmaster" {
		t.Fatalf("calls = %v", fm.calls)
	}
	c := cookie(rec, SessionCookie)
	r := httptest.NewRequest("GET", "/api/session", nil)
	r.AddCookie(c)
	var s struct{ Email, Mode, Locale, Theme string }
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &s)
	if s.Email != "alice@golub.test" || s.Mode != "sso" || s.Locale != "sr-Latn" || s.Theme != "dark" {
		t.Fatalf("session %+v", s)
	}

	rec = serve(h, httptest.NewRequest("GET", "/sso?token="+url.QueryEscape(tok), nil))
	if loc := rec.Header().Get("Location"); loc != "/?error=auth.sso_invalid" {
		t.Fatalf("replay location = %q", loc)
	}
}

func TestSSODisabled(t *testing.T) {
	h, _ := newAuthServer(t, func(o *Options) { o.SSO = nil })
	rec := serve(h, httptest.NewRequest("GET", "/sso?token=x", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestSecureCookieBehindTrustedProxy(t *testing.T) {
	h, _ := newAuthServer(t, func(o *Options) { o.TrustProxy = []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")} })
	r := jsonReq("POST", "/api/login", `{"email":"alice@golub.test","password":"secret"}`)
	r.RemoteAddr = "192.0.2.10:5555"
	r.Header.Set("X-Forwarded-Proto", "https")
	if c := cookie(serve(h, r), SessionCookie); c == nil || !c.Secure {
		t.Fatalf("cookie %+v", c)
	}

	r = jsonReq("POST", "/api/login", `{"email":"alice@golub.test","password":"secret"}`)
	r.RemoteAddr = "198.51.100.1:5555"
	r.Header.Set("X-Forwarded-Proto", "https")
	if c := cookie(serve(h, r), SessionCookie); c == nil || c.Secure {
		t.Fatalf("untrusted proxy honored: %+v", c)
	}
}

func TestClientIP(t *testing.T) {
	trust := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	cases := []struct{ remote, xff, want string }{
		{"198.51.100.1:1", "1.2.3.4", "198.51.100.1"},
		{"10.0.0.1:1", "1.2.3.4", "1.2.3.4"},
		{"10.0.0.1:1", "6.6.6.6, 1.2.3.4, 10.0.0.2", "1.2.3.4"},
		{"10.0.0.1:1", "", "10.0.0.1"},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tc.remote
		if tc.xff != "" {
			r.Header.Set("X-Forwarded-For", tc.xff)
		}
		if got := clientIP(r, trust); got != tc.want {
			t.Errorf("%s %q: got %s", tc.remote, tc.xff, got)
		}
	}
}

func TestResolveLocale(t *testing.T) {
	cases := []struct{ accept, def, want string }{
		{"sr-RS,en;q=0.5", "en", "sr-Latn"},
		{"sr-Cyrl-RS", "en", "en"},
		{"de-DE,de", "sr-Latn", "sr-Latn"},
		{"en-GB", "sr-Latn", "en"},
		{"", "en", "en"},
		{"en;q=0.1, sr;q=0.9", "en", "sr-Latn"},
	}
	for _, tc := range cases {
		if got := localeFromAccept(tc.accept, tc.def); got != tc.want {
			t.Errorf("%q: got %s want %s", tc.accept, got, tc.want)
		}
	}
}
