// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stefanpejcic/golub/internal/contacts"
)

func getBook(t *testing.T, h http.Handler, c *http.Cookie) contacts.Book {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/contacts", nil)
	r.AddCookie(c)
	rec := serve(h, r)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var b contacts.Book
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	return b
}

func putBook(t *testing.T, h http.Handler, c *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := jsonReq("PUT", "/api/contacts", body)
	r.AddCookie(c)
	return serve(h, r)
}

func TestContactsSaveAndLoad(t *testing.T) {
	h, c, _ := newComposeServer(t)
	if b := getBook(t, h, c); len(b.Contacts) != 0 || b.Groups == nil {
		t.Fatalf("empty book %+v", b)
	}
	rec := putBook(t, h, c, `{"groups":["Work"],"contacts":[{"name":"Bob","emails":["bob@golub.test"],"groups":["Work"]}]}`)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	b := getBook(t, h, c)
	if len(b.Contacts) != 1 || b.Contacts[0].ID == "" || b.Contacts[0].Groups[0] != "Work" {
		t.Fatalf("%+v", b)
	}
	if rec := putBook(t, h, c, `{"contacts":[{"name":"x","emails":["nope"]}]}`); rec.Code != 400 {
		t.Fatalf("invalid book: %d", rec.Code)
	}
	r := httptest.NewRequest("GET", "/api/folders", nil)
	r.AddCookie(c)
	if strings.Contains(serve(h, r).Body.String(), "golub-contacts") {
		t.Error("contacts folder must stay hidden")
	}
}

func TestContactsImportExport(t *testing.T) {
	h, c, _ := newComposeServer(t)
	putBook(t, h, c, `{"contacts":[{"name":"Bob","emails":["bob@golub.test"]}]}`)
	vcf := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Robert\r\nEMAIL:bob@golub.test\r\nTEL:123\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Anna\r\nEMAIL:anna@golub.test\r\nCATEGORIES:Friends\r\nEND:VCARD\r\n"
	r := httptest.NewRequest("POST", "/api/contacts/import", strings.NewReader(vcf))
	r.Header.Set("Content-Type", "text/vcard")
	r.AddCookie(c)
	rec := serve(h, r)
	var res struct{ Added, Updated int }
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != 200 || res.Added != 1 || res.Updated != 1 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	b := getBook(t, h, c)
	if len(b.Contacts) != 2 || b.Contacts[0].Phones[0] != "123" || b.Groups[0] != "Friends" {
		t.Fatalf("%+v", b)
	}

	r = httptest.NewRequest("GET", "/api/contacts/export?group=Friends", nil)
	r.AddCookie(c)
	rec = serve(h, r)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/vcard") || !strings.Contains(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	if !strings.Contains(body, "FN:Anna") || strings.Contains(body, "Bob") {
		t.Fatalf("group export: %s", body)
	}

	r = httptest.NewRequest("POST", "/api/contacts/import", strings.NewReader("hello"))
	r.Header.Set("Content-Type", "text/vcard")
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 400 {
		t.Fatalf("garbage import: %d", rec.Code)
	}
}

func TestSendCollectsRecipients(t *testing.T) {
	h, c, _ := newComposeServer(t)
	putBook(t, h, c, `{"contacts":[{"name":"Bob","emails":["bob@golub.test"]}]}`)
	sendJSON(t, h, c, "/api/compose/send", map[string]any{"identity": "default", "to": []string{"Bob <bob@golub.test>", "Carol <carol@golub.test>"}, "subject": "s", "text": "x"})
	b := getBook(t, h, c)
	if len(b.Collected) != 1 || b.Collected[0].Email != "carol@golub.test" || b.Collected[0].Name != "Carol" {
		t.Fatalf("%+v", b.Collected)
	}

	r := jsonReq("PUT", "/api/settings", `{"collectRecipients":false}`)
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 200 {
		t.Fatalf("settings %d %s", rec.Code, rec.Body)
	}
	sendJSON(t, h, c, "/api/compose/send", map[string]any{"identity": "default", "to": []string{"dave@golub.test"}, "subject": "s", "text": "x"})
	if b := getBook(t, h, c); len(b.Collected) != 1 {
		t.Fatalf("collecting is off: %+v", b.Collected)
	}
}
