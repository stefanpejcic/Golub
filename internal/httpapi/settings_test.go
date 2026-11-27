// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/settings"
)

func getSettings(t *testing.T, h http.Handler, c *http.Cookie) settings.Settings {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/settings", nil)
	r.AddCookie(c)
	rec := serve(h, r)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var s settings.Settings
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	return s
}

func put(t *testing.T, h http.Handler, c *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := jsonReq("PUT", "/api/settings", body)
	r.AddCookie(c)
	return serve(h, r)
}

func TestSettingsAPI(t *testing.T) {
	h, c := newMutateServer(t)
	s := getSettings(t, h, c)
	if s.PageSize != 50 || len(s.Identities) != 1 || s.Identities[0].Email != "alice@golub.test" {
		t.Fatalf("defaults %+v", s)
	}
	s.PageSize = 25
	s.Identities[0].Name = "Alice Ž"
	s.Responses = []settings.Response{{ID: "r1", Name: "Thanks", Text: "Thank you!"}}
	b, _ := json.Marshal(s)
	if rec := put(t, h, c, string(b)); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	s = getSettings(t, h, c)
	if s.PageSize != 25 || s.Identities[0].Name != "Alice Ž" || len(s.Responses) != 1 {
		t.Fatalf("saved %+v", s)
	}

	s.PageSize = 7
	b, _ = json.Marshal(s)
	rec := put(t, h, c, string(b))
	if rec.Code != 400 || errCode(t, rec) != "settings.invalid" || !strings.Contains(rec.Body.String(), "pageSize") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	r := jsonReq("PUT", "/api/settings", `{}`)
	if rec := serve(h, r); rec.Code != 401 {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestDeleteModePermanent(t *testing.T) {
	h, c := newMutateServer(t)
	s := getSettings(t, h, c)
	s.DeleteMode = "permanent"
	b, _ := json.Marshal(s)
	put(t, h, c, string(b))
	rec := post(t, h, c, "/api/messages/delete", `{"folder":"INBOX","uids":[1]}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"trashed":false`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if n := len(list(t, h, c, "Trash")); n != 0 {
		t.Fatalf("trash = %d", n)
	}
}

func TestSpecialFolderOverride(t *testing.T) {
	h, c := newMutateServer(t)
	s := getSettings(t, h, c)
	s.SpecialFolders = map[string]string{"trash": "Work"}
	b, _ := json.Marshal(s)
	put(t, h, c, string(b))
	post(t, h, c, "/api/messages/delete", `{"folder":"INBOX","uids":[1]}`)
	if n := len(list(t, h, c, "Work")); n != 1 {
		t.Fatalf("override ignored, work = %d", n)
	}
}

func TestFolderEndpoints(t *testing.T) {
	h, c := newMutateServer(t)
	folders := func() map[string]imapx.Folder {
		r := httptest.NewRequest("GET", "/api/folders", nil)
		r.AddCookie(c)
		var b struct{ Folders []imapx.Folder }
		_ = json.Unmarshal(serve(h, r).Body.Bytes(), &b)
		m := map[string]imapx.Folder{}
		for _, f := range b.Folders {
			m[f.Name] = f
		}
		return m
	}
	steps := []struct{ path, body string }{
		{"/api/folders/create", `{"name":"Projects"}`},
		{"/api/folders/rename", `{"name":"Projects","to":"Clients"}`},
		{"/api/folders/subscribe", `{"name":"Clients","value":false}`},
	}
	for _, s := range steps {
		if rec := post(t, h, c, s.path, s.body); rec.Code != 204 {
			t.Fatalf("%s: %d %s", s.path, rec.Code, rec.Body)
		}
	}
	if f, ok := folders()["Clients"]; !ok || f.Subscribed {
		t.Fatalf("%+v", folders())
	}
	if rec := post(t, h, c, "/api/folders/delete", `{"name":"Clients"}`); rec.Code != 204 {
		t.Fatal(rec.Code)
	}
	if _, ok := folders()["Clients"]; ok {
		t.Fatal("not deleted")
	}
	if rec := post(t, h, c, "/api/folders/empty", `{"name":"INBOX"}`); rec.Code != 204 || folders()["INBOX"].Total != 0 {
		t.Fatal("empty failed")
	}
	for _, bad := range []struct{ path, body string }{
		{"/api/folders/delete", `{"name":"INBOX"}`},
		{"/api/folders/rename", `{"name":"INBOX","to":"X"}`},
		{"/api/folders/create", `{"name":""}`},
		{"/api/folders/create", `{"name":"golub-settings"}`},
	} {
		if rec := post(t, h, c, bad.path, bad.body); rec.Code != 400 {
			t.Errorf("%s %s: %d", bad.path, bad.body, rec.Code)
		}
	}
}
