// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/imaptest"
	"github.com/stefanpejcic/golub/internal/imapx"
)

func newMutateServer(t *testing.T) (http.Handler, *http.Cookie) {
	t.Helper()
	mem := imapmemserver.New()
	u := imaptest.User(t, mem, "alice@golub.test", "secret", "Trash", "Work")
	for i := 1; i <= 4; i++ {
		raw := "From: bob@golub.test\r\nSubject: M" + string(rune('0'+i)) + "\r\n\r\nx\r\n"
		if _, err := u.Append("INBOX", literal(raw), &imap.AppendOptions{Time: time.Unix(int64(1_800_000_000+i), 0)}); err != nil {
			t.Fatal(err)
		}
	}
	pool := imapx.NewPool(&imapx.Dialer{Endpoint: imaptest.Listen(t, mem)}, time.Minute)
	t.Cleanup(pool.Close)
	h, _ := newAuthServer(t, func(o *Options) { o.Mail = pool; o.Pool = pool })
	return h, login(t, h)
}

func post(t *testing.T, h http.Handler, c *http.Cookie, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := jsonReq("POST", path, body)
	r.AddCookie(c)
	return serve(h, r)
}

func list(t *testing.T, h http.Handler, c *http.Cookie, folder string) []imapx.Summary {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/messages?folder="+folder, nil)
	r.AddCookie(c)
	var b struct{ Messages []imapx.Summary }
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &b)
	return b.Messages
}

func TestFlagEndpoint(t *testing.T) {
	h, c := newMutateServer(t)
	if rec := post(t, h, c, "/api/messages/flag", `{"folder":"INBOX","uids":[1,2],"flag":"seen","value":true}`); rec.Code != 204 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if rec := post(t, h, c, "/api/messages/flag", `{"folder":"INBOX","uids":[3],"flag":"flagged","value":true}`); rec.Code != 204 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	seen := map[uint32]bool{}
	flagged := map[uint32]bool{}
	for _, m := range list(t, h, c, "INBOX") {
		seen[m.UID], flagged[m.UID] = m.Seen, m.Flagged
	}
	if !seen[1] || !seen[2] || seen[3] || !flagged[3] || flagged[1] {
		t.Fatalf("seen %v flagged %v", seen, flagged)
	}
	if rec := post(t, h, c, "/api/messages/flag", `{"folder":"INBOX","uids":[1],"flag":"seen","value":false}`); rec.Code != 204 {
		t.Fatal(rec.Code)
	}
}

func TestMoveEndpoint(t *testing.T) {
	h, c := newMutateServer(t)
	if rec := post(t, h, c, "/api/messages/move", `{"folder":"INBOX","uids":[1,2],"to":"Work"}`); rec.Code != 204 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if n := len(list(t, h, c, "Work")); n != 2 {
		t.Fatalf("work = %d", n)
	}
	if rec := post(t, h, c, "/api/messages/move", `{"folder":"INBOX","uids":[3],"to":"Nope"}`); rec.Code != 404 {
		t.Fatalf("missing dest: %d", rec.Code)
	}
}

func TestDeleteEndpoint(t *testing.T) {
	h, c := newMutateServer(t)
	rec := post(t, h, c, "/api/messages/delete", `{"folder":"INBOX","uids":[1]}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"trashed":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	trash := list(t, h, c, "Trash")
	if len(trash) != 1 || len(list(t, h, c, "INBOX")) != 3 {
		t.Fatalf("trash %d", len(trash))
	}
	rec = post(t, h, c, "/api/messages/delete", `{"folder":"Trash","uids":[`+itoa(trash[0].UID)+`]}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"trashed":false`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if n := len(list(t, h, c, "Trash")); n != 0 {
		t.Fatalf("trash after permanent delete = %d", n)
	}
}

func TestMutationValidation(t *testing.T) {
	h, c := newMutateServer(t)
	for _, tc := range []struct{ path, body string }{
		{"/api/messages/flag", `{"folder":"INBOX","uids":[],"flag":"seen","value":true}`},
		{"/api/messages/flag", `{"folder":"INBOX","uids":[0],"flag":"seen","value":true}`},
		{"/api/messages/flag", `{"folder":"INBOX","uids":[1],"flag":"deleted","value":true}`},
		{"/api/messages/move", `{"folder":"INBOX","uids":[1],"to":""}`},
		{"/api/messages/move", `{"folder":"INBOX","uids":[1],"to":"INBOX"}`},
		{"/api/messages/delete", `{"folder":"","uids":[1]}`},
	} {
		if rec := post(t, h, c, tc.path, tc.body); rec.Code != 400 {
			t.Errorf("%s %s: %d", tc.path, tc.body, rec.Code)
		}
	}
	r := jsonReq("POST", "/api/messages/delete", `{"folder":"INBOX","uids":[1]}`)
	if rec := serve(h, r); rec.Code != 401 {
		t.Errorf("anonymous: %d", rec.Code)
	}
}

func itoa(u uint32) string { return strconv.FormatUint(uint64(u), 10) }

func TestArchiveEndpoint(t *testing.T) {
	h, c := newMutateServer(t)
	rec := post(t, h, c, "/api/messages/archive", `{"folder":"INBOX","uids":[1,2]}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"folder":"Archive"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if n := len(list(t, h, c, "Archive")); n != 2 {
		t.Fatalf("archive = %d", n)
	}
	if rec := post(t, h, c, "/api/messages/archive", `{"folder":"INBOX","uids":[3]}`); rec.Code != 200 {
		t.Fatalf("second archive into existing folder: %d %s", rec.Code, rec.Body)
	}
	if rec := post(t, h, c, "/api/messages/archive", `{"folder":"Archive","uids":[1]}`); rec.Code != 400 {
		t.Fatalf("archiving inside Archive: %d", rec.Code)
	}
}
