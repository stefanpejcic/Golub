// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/imaptest"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/proxy"
)

// newMessageServer appends the named fixtures to INBOX in order; UIDs start at 1.
func newMessageServer(t *testing.T, fixtures ...string) http.Handler {
	t.Helper()
	mem := imapmemserver.New()
	u := imaptest.User(t, mem, "alice@golub.test", "secret")
	for i, f := range fixtures {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mail", f))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := u.Append("INBOX", literal(string(raw)), &imap.AppendOptions{Time: time.Unix(int64(1_800_000_000+i), 0)}); err != nil {
			t.Fatal(err)
		}
	}
	pool := imapx.NewPool(&imapx.Dialer{Endpoint: imaptest.Listen(t, mem)}, time.Minute)
	t.Cleanup(pool.Close)
	h, _ := newAuthServer(t, func(o *Options) {
		o.Mail = pool
		o.Pool = pool
		o.Proxy = proxy.New([]byte("0123456789abcdef0123456789abcdef"))
	})
	return h
}

type testMessageView struct {
	UID           uint32
	Subject       string
	Text          string
	HTML          bool
	Seen          bool
	BlockedImages int
	From          []struct{ Name, Address string }
	Attachments   []struct {
		ID, Filename, ContentType string
		Size                      int
	}
}

func getMessage(t *testing.T, h http.Handler, c *http.Cookie, uid int) (*httptest.ResponseRecorder, testMessageView) {
	t.Helper()
	r := httptest.NewRequest("GET", fmt.Sprintf("/api/message?folder=INBOX&uid=%d", uid), nil)
	r.AddCookie(c)
	rec := serve(h, r)
	var v testMessageView
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	return rec, v
}

func TestMessageView(t *testing.T) {
	h := newMessageServer(t, "01-plain.eml", "05-attachment.eml", "02-html.eml")
	c := login(t, h)

	rec, v := getMessage(t, h, c, 1)
	if rec.Code != 200 || v.Subject != "Plain text" || !strings.Contains(v.Text, "čćžšđ") || v.HTML || v.From[0].Name != "Bob Sender" {
		t.Fatalf("%d %+v", rec.Code, v)
	}
	if !v.Seen {
		t.Error("opening should mark the message read")
	}

	_, v = getMessage(t, h, c, 2)
	if len(v.Attachments) != 2 || v.Attachments[0].Filename != "report.pdf" {
		t.Fatalf("%+v", v.Attachments)
	}

	_, v = getMessage(t, h, c, 3)
	if !v.HTML || v.BlockedImages != 1 {
		t.Fatalf("%+v", v)
	}

	r := httptest.NewRequest("GET", "/api/folders", nil)
	r.AddCookie(c)
	var f struct{ Folders []imapx.Folder }
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &f)
	if f.Folders[0].Unread != 0 {
		t.Errorf("unread after opening all = %d", f.Folders[0].Unread)
	}
}

func TestMessageErrors(t *testing.T) {
	h := newMessageServer(t, "01-plain.eml")
	c := login(t, h)
	for path, want := range map[string]int{
		"/api/message?folder=INBOX&uid=0":             400,
		"/api/message?folder=INBOX&uid=abc":           400,
		"/api/message?folder=INBOX":                   400,
		"/api/message?folder=INBOX&uid=99":            404,
		"/api/message?folder=Nope&uid=1":              404,
		"/api/message/part?folder=INBOX&uid=1":        400,
		"/api/message/part?folder=INBOX&uid=1&part=7": 404,
	} {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(c)
		if rec := serve(h, r); rec.Code != want {
			t.Errorf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
	if rec := serve(h, httptest.NewRequest("GET", "/api/message?folder=INBOX&uid=1", nil)); rec.Code != 401 {
		t.Errorf("anonymous: %d", rec.Code)
	}
}

func body(t *testing.T, h http.Handler, c *http.Cookie, uid int, images bool) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/api/message/body?folder=INBOX&uid=%d", uid)
	if images {
		path += "&images=1"
	}
	r := httptest.NewRequest("GET", path, nil)
	r.Host = "mail.example.test"
	r.AddCookie(c)
	return serve(h, r)
}

func TestMessageBody(t *testing.T) {
	h := newMessageServer(t, "02-html.eml", "04-inline-cid.eml", "21-xss-svg-onload.eml")
	c := login(t, h)

	rec := body(t, h, c, 1, false)
	csp := rec.Header().Get("Content-Security-Policy")
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") ||
		!strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "frame-ancestors 'self'") || strings.Contains(csp, "script-src") {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("body must be frameable by the app")
	}
	if strings.Contains(rec.Body.String(), "example.com/banner.png") {
		t.Error("remote image loaded without consent")
	}

	rec = body(t, h, c, 1, true)
	if !strings.Contains(rec.Body.String(), `src="/proxy?u=https%3A%2F%2Fexample.com%2Fbanner.png&amp;s=`) || strings.Contains(rec.Body.String(), "track.png") {
		t.Errorf("proxy url missing: %s", rec.Body)
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "img-src data: http://mail.example.test/proxy") {
		t.Errorf("csp = %s", rec.Header().Get("Content-Security-Policy"))
	}

	if rec = body(t, h, c, 2, false); !strings.Contains(rec.Body.String(), `src="data:image/png;base64,`) {
		t.Errorf("cid not inlined: %s", rec.Body)
	}
	if rec = body(t, h, c, 3, true); strings.Contains(strings.ToLower(rec.Body.String()), "onload") || strings.Contains(rec.Body.String(), "evil.test") {
		t.Errorf("xss survived: %s", rec.Body)
	}
}

func TestPartDownload(t *testing.T) {
	h := newMessageServer(t, "05-attachment.eml", "02-html.eml")
	c := login(t, h)
	_, v := getMessage(t, h, c, 1)

	for _, a := range v.Attachments {
		r := httptest.NewRequest("GET", fmt.Sprintf("/api/message/part?folder=INBOX&uid=1&part=%s", a.ID), nil)
		r.AddCookie(c)
		rec := serve(h, r)
		cd := rec.Header().Get("Content-Disposition")
		if rec.Code != 200 || !strings.HasPrefix(cd, "attachment;") || rec.Body.Len() != a.Size {
			t.Fatalf("%s: %d %q", a.Filename, rec.Code, cd)
		}
		if a.Filename == "članarina šema.txt" && !strings.Contains(cd, "filename*=UTF-8''%C4%8Dlanarina%20%C5%A1ema.txt") {
			t.Errorf("filename* = %q", cd)
		}
		if a.Filename == "report.pdf" && rec.Header().Get("Content-Type") != "application/pdf" {
			t.Errorf("pdf type %q", rec.Header().Get("Content-Type"))
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
			t.Error("downloads must be sandboxed")
		}
	}
}

func TestRiskyTypes(t *testing.T) {
	for in, want := range map[string]string{
		"text/html":             "application/octet-stream",
		"image/svg+xml":         "application/octet-stream",
		"application/xhtml+xml": "application/octet-stream",
		"text/plain":            "text/plain",
		"application/pdf":       "application/pdf",
		"":                      "application/octet-stream",
	} {
		if got := downloadType(in); got != want {
			t.Errorf("%q -> %q", in, got)
		}
	}
}

func TestMessageViewWithoutMarkingRead(t *testing.T) {
	h := newMessageServer(t, "01-plain.eml")
	c := login(t, h)
	r := httptest.NewRequest("GET", "/api/message?folder=INBOX&uid=1&seen=0", nil)
	r.AddCookie(c)
	var v testMessageView
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &v)
	if v.Seen {
		t.Fatal("seen=0 must not mark the message read")
	}
}

func TestRawAndHeaders(t *testing.T) {
	h := newMessageServer(t, "06-windows-1250.eml")
	c := login(t, h)
	get := func(q string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/message/raw?folder=INBOX&uid=1"+q, nil)
		r.AddCookie(c)
		return serve(h, r)
	}
	rec := get("&download=1")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "message/rfc822" ||
		!strings.Contains(rec.Header().Get("Content-Disposition"), `filename*=UTF-8''Pozdrav%20%C5%A1ef%C4%8De.eml`) {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	rec = get("")
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") ||
		!strings.Contains(rec.Body.String(), "Subject: =?windows-1250?Q?") {
		t.Fatalf("source view: %v %q", rec.Header(), rec.Body.String()[:80])
	}
	rec = get("&headers=1")
	body := rec.Body.String()
	if !strings.Contains(body, "Message-ID:") || strings.Contains(body, "Šta ima") || strings.Contains(body, "\r\n\r\n") {
		t.Fatalf("headers only: %q", body)
	}
}

func TestPartPreview(t *testing.T) {
	h := newMessageServer(t, "05-attachment.eml", "02-html.eml")
	c := login(t, h)
	_, v := getMessage(t, h, c, 1)

	for _, a := range v.Attachments {
		r := httptest.NewRequest("GET", fmt.Sprintf("/api/message/part?folder=INBOX&uid=1&part=%s&inline=1", a.ID), nil)
		r.AddCookie(c)
		rec := serve(h, r)
		cd, csp := rec.Header().Get("Content-Disposition"), rec.Header().Get("Content-Security-Policy")
		if rec.Code != 200 || !strings.HasPrefix(cd, "inline;") {
			t.Fatalf("%s: %d %q", a.Filename, rec.Code, cd)
		}
		switch a.Filename {
		case "report.pdf":
			// Chrome refuses to show a PDF under a sandbox CSP, so it gets a strict policy without one.
			if strings.Contains(csp, "sandbox") || !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "frame-ancestors 'self'") {
				t.Errorf("pdf csp %q", csp)
			}
		case "članarina šema.txt":
			if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" || !strings.Contains(csp, "sandbox") {
				t.Errorf("text: %q %q", rec.Header().Get("Content-Type"), csp)
			}
		}
	}
}

func TestPreviewOnlySafeTypes(t *testing.T) {
	for ct, want := range map[string]bool{
		"image/png": true, "image/jpeg": true, "application/pdf": true, "text/plain; charset=windows-1250": true,
		"image/svg+xml": false, "text/html": false, "application/zip": false, "": false,
	} {
		if got := previewable(ct); got != want {
			t.Errorf("%q = %v", ct, got)
		}
	}
}
