// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

const testPub = "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nxjMEZ\n-----END PGP PUBLIC KEY BLOCK-----\n"

func TestKeyringSaveAndLoad(t *testing.T) {
	h, c, _ := newComposeServer(t)
	r := httptest.NewRequest("GET", "/api/pgp", nil)
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"keys":[]`) {
		t.Fatalf("empty keyring: %d %s", rec.Code, rec.Body)
	}
	body := `{"keys":[{"fingerprint":"` + strings.Repeat("ab", 20) + `","emails":["bob@golub.test"],"private":false,"armored":` + mustJSON(testPub) + `}]}`
	r = jsonReq("PUT", "/api/pgp", body)
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	r = httptest.NewRequest("GET", "/api/pgp", nil)
	r.AddCookie(c)
	if rec := serve(h, r); !strings.Contains(rec.Body.String(), "bob@golub.test") {
		t.Fatalf("%s", rec.Body)
	}
	r = jsonReq("PUT", "/api/pgp", `{"keys":[{"fingerprint":"nope","armored":"x"}]}`)
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 400 {
		t.Fatalf("bad key: %d", rec.Code)
	}
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestSendEncrypted(t *testing.T) {
	h, c, srv := newComposeServer(t)
	send := func(path string, meta map[string]any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		m, _ := json.Marshal(meta)
		_ = mw.WriteField("message", string(m))
		_ = mw.WriteField("pgp", `{"encrypted":"-----BEGIN PGP MESSAGE-----\n\nwV4D\n-----END PGP MESSAGE-----\n"}`)
		_ = mw.Close()
		r := httptest.NewRequest("POST", path, &buf)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		r.AddCookie(c)
		return serve(h, r)
	}
	meta := map[string]any{"identity": "default", "to": []string{"bob@golub.test"}, "subject": "secret plan", "text": "plaintext that must not leak"}
	if rec := send("/api/compose/send", meta); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	data := srv.Messages()[0].Data
	if !strings.Contains(data, "multipart/encrypted") || strings.Contains(data, "plaintext that must not leak") {
		t.Fatalf("%s", data)
	}
	if rec := send("/api/compose/draft", meta); rec.Code != 400 || !strings.Contains(rec.Body.String(), "compose.pgp_draft") {
		t.Fatalf("encrypted drafts would need decrypting to edit: %d %s", rec.Code, rec.Body)
	}
}

func TestMessageViewReportsPGP(t *testing.T) {
	h, c, _ := newComposeServer(t)
	raw := "From: a@golub.test\r\nTo: alice@golub.test\r\nSubject: enc\r\nMIME-Version: 1.0\r\nContent-Type: multipart/encrypted; protocol=\"application/pgp-encrypted\"; boundary=\"b\"\r\n\r\n--b\r\nContent-Type: application/pgp-encrypted\r\n\r\nVersion: 1\r\n--b\r\nContent-Type: application/octet-stream\r\n\r\n-----BEGIN PGP MESSAGE-----\r\n\r\nwV4D\r\n-----END PGP MESSAGE-----\r\n--b--\r\n"
	r := httptest.NewRequest("POST", "/api/folders/import?folder=Sent", strings.NewReader(raw))
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	r = httptest.NewRequest("GET", "/api/message?folder=Sent&uid=1", nil)
	r.AddCookie(c)
	var v struct{ PGP string }
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &v)
	if v.PGP != "encrypted" {
		t.Fatalf("pgp = %q", v.PGP)
	}
}
