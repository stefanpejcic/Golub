// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/imaptest"
	"github.com/stefanpejcic/golub/internal/imapx"
)

func newMailServer(t *testing.T, inbox int) (http.Handler, *imapx.Pool) {
	t.Helper()
	mem := imapmemserver.New()
	u := imaptest.User(t, mem, "alice@golub.test", "secret", "Sent", "Work")
	imaptest.User(t, mem, "alice@golub.test*golubmaster", "masterpw")
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= inbox; i++ {
		raw := fmt.Sprintf("From: Bob <bob@golub.test>\r\nTo: alice@golub.test\r\nSubject: Message %03d\r\n\r\nbody\r\n", i)
		var flags []imap.Flag
		if i%2 == 0 {
			flags = []imap.Flag{imap.FlagSeen}
		}
		if _, err := u.Append("INBOX", literal(raw), &imap.AppendOptions{Flags: flags, Time: base.Add(time.Duration(i) * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	pool := imapx.NewPool(&imapx.Dialer{Endpoint: imaptest.Listen(t, mem)}, time.Minute)
	t.Cleanup(pool.Close)
	h, _ := newAuthServer(t, func(o *Options) {
		o.Mail = pool
		o.Pool = pool
	})
	return h, pool
}

func authed(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	c := login(t, h)
	r := httptest.NewRequest("GET", path, nil)
	r.AddCookie(c)
	return serve(h, r)
}

func TestFolders(t *testing.T) {
	h, _ := newMailServer(t, 3)
	rec := authed(t, h, "/api/folders")
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var body struct{ Folders []imapx.Folder }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Folders) != 3 || body.Folders[0].Name != "INBOX" || body.Folders[0].Unread != 2 || body.Folders[0].Total != 3 {
		t.Fatalf("folders = %+v", body.Folders)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("mail data must not be cached")
	}
}

func TestMessages(t *testing.T) {
	h, _ := newMailServer(t, 7)
	rec := authed(t, h, "/api/messages?folder=INBOX&page=2&size=3")
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var body struct {
		Total, Page, Size int
		Messages          []imapx.Summary
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 7 || body.Page != 2 || body.Size != 3 || len(body.Messages) != 3 || body.Messages[0].Subject != "Message 004" {
		t.Fatalf("body = %+v", body)
	}
}

func TestMessagesValidation(t *testing.T) {
	h, _ := newMailServer(t, 1)
	cases := map[string]struct {
		status int
		code   string
	}{
		"/api/messages":                                     {400, "request.invalid"},
		"/api/messages?folder=INBOX&page=0":                 {400, "request.invalid"},
		"/api/messages?folder=INBOX&size=1000":              {400, "request.invalid"},
		"/api/messages?folder=INBOX&page=x":                 {400, "request.invalid"},
		"/api/messages?folder=" + url.QueryEscape("a\r\nb"): {400, "request.invalid"},
		"/api/messages?folder=Missing":                      {404, "folder.not_found"},
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			rec := authed(t, h, path)
			if rec.Code != want.status || errCode(t, rec) != want.code {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestMailRequiresSession(t *testing.T) {
	h, _ := newMailServer(t, 1)
	for _, p := range []string{"/api/folders", "/api/messages?folder=INBOX"} {
		rec := serve(h, httptest.NewRequest("GET", p, nil))
		if rec.Code != 401 || errCode(t, rec) != "auth.required" {
			t.Errorf("%s: %d %s", p, rec.Code, rec.Body)
		}
	}
}

func TestMailUsesMasterLoginForSSO(t *testing.T) {
	h, _ := newMailServer(t, 2)
	tok := ssoToken(t)
	rec := serve(h, httptest.NewRequest("GET", "/sso?token="+url.QueryEscape(tok), nil))
	c := cookie(rec, SessionCookie)
	if c == nil {
		t.Fatalf("sso failed: %d %v", rec.Code, rec.Header())
	}
	r := httptest.NewRequest("GET", "/api/folders", nil)
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestMailPasswordChangedLogsOut(t *testing.T) {
	h, _ := newMailServer(t, 1)
	c := login(t, h)
	codec := testCodec(t)
	s, err := codec.Decode(c.Value, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	s.Password = "stale"
	v, _ := codec.Encode(s)
	r := httptest.NewRequest("GET", "/api/folders", nil)
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: v})
	rec := serve(h, r)
	if rec.Code != 401 || errCode(t, rec) != "auth.required" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if cl := cookie(rec, SessionCookie); cl == nil || cl.MaxAge >= 0 {
		t.Error("stale session cookie not cleared")
	}
}
