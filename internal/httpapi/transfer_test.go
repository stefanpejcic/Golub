// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stefanpejcic/golub/internal/mbox"
)

func TestFolderImportExport(t *testing.T) {
	h, c, _ := newComposeServer(t)
	src := "From bob@golub.test Sat Jan  3 01:05:34 2026\nFrom: Bob <bob@golub.test>\nSubject: first\nDate: Sat, 3 Jan 2026 01:05:34 +0000\n\n>From the start\n\n" +
		"From bob@golub.test Sun Jan  4 01:05:34 2026\nFrom: Bob <bob@golub.test>\nSubject: second\n\nbye\n"
	r := httptest.NewRequest("POST", "/api/folders/import?folder=Sent", strings.NewReader(src))
	r.Header.Set("Content-Type", "application/octet-stream")
	r.AddCookie(c)
	rec := serve(h, r)
	var res struct{ Imported int }
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != 200 || res.Imported != 2 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if got := list(t, h, c, "Sent"); len(got) != 2 {
		t.Fatalf("%+v", got)
	}

	r = httptest.NewRequest("GET", "/api/folders/export?folder=Sent", nil)
	r.AddCookie(c)
	rec = serve(h, r)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/mbox" || !strings.Contains(rec.Header().Get("Content-Disposition"), "Sent.mbox") {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	var subjects []string
	_ = mbox.Read(rec.Body, func(m []byte) error {
		subjects = append(subjects, strings.SplitN(strings.SplitN(string(m), "Subject: ", 2)[1], "\r\n", 2)[0])
		if strings.Contains(string(m), "Subject: first") && !strings.Contains(string(m), "\r\nFrom the start") {
			t.Errorf("escaping lost: %q", m)
		}
		return nil
	})
	if strings.Join(subjects, ",") != "first,second" {
		t.Fatalf("%v", subjects)
	}

	for _, path := range []string{"/api/folders/export?folder=golub-contacts", "/api/folders/export?folder="} {
		r = httptest.NewRequest("GET", path, nil)
		r.AddCookie(c)
		if rec := serve(h, r); rec.Code != 400 {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
}

func TestImportSingleEML(t *testing.T) {
	h, c, _ := newComposeServer(t)
	r := httptest.NewRequest("POST", "/api/folders/import?folder=Sent", strings.NewReader("Subject: lone\r\n\r\nbody\r\n"))
	r.AddCookie(c)
	rec := serve(h, r)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"imported":1`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	r = httptest.NewRequest("POST", "/api/folders/import?folder=Sent", strings.NewReader(""))
	r.AddCookie(c)
	if rec := serve(h, r); rec.Code != 400 {
		t.Fatalf("empty upload: %d", rec.Code)
	}
}
