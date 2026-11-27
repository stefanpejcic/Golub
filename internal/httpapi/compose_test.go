// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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
	"github.com/stefanpejcic/golub/internal/smtptest"
	"github.com/stefanpejcic/golub/internal/smtpx"
)

func newComposeServer(t *testing.T, fixtures ...string) (http.Handler, *http.Cookie, *smtptest.Server) {
	t.Helper()
	mem := imapmemserver.New()
	u := imaptest.User(t, mem, "alice@golub.test", "secret", "Sent", "Drafts", "Trash")
	for i, f := range fixtures {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mail", f))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := u.Append("INBOX", literal(string(raw)), &imap.AppendOptions{Time: time.Unix(int64(1_800_000_000+i), 0)}); err != nil {
			t.Fatal(err)
		}
	}
	srv := smtptest.Start(t, map[string]string{"alice@golub.test": "secret"})
	pool := imapx.NewPool(&imapx.Dialer{Endpoint: imaptest.Listen(t, mem)}, time.Minute)
	t.Cleanup(pool.Close)
	h, _ := newAuthServer(t, func(o *Options) {
		o.Mail = pool
		o.Pool = pool
		o.SMTP = &smtpx.Sender{Endpoint: srv.Endpoint, Timeout: 5 * time.Second}
	})
	return h, login(t, h), srv
}

func sendJSON(t *testing.T, h http.Handler, c *http.Cookie, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	r := jsonReq("POST", path, string(b))
	r.AddCookie(c)
	return serve(h, r)
}

func TestSendPlain(t *testing.T) {
	h, c, srv := newComposeServer(t)
	rec := sendJSON(t, h, c, "/api/compose/send", map[string]any{
		"identity": "default",
		"to":       []string{"Bob <bob@golub.test>"},
		"bcc":      []string{"hidden@golub.test"},
		"subject":  "Zdravo",
		"text":     "Kako si?",
	})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"savedToSent":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	msgs := srv.Messages()
	if len(msgs) != 1 || msgs[0].From != "alice@golub.test" || len(msgs[0].To) != 2 {
		t.Fatalf("%+v", msgs)
	}
	if strings.Contains(msgs[0].Data, "hidden@golub.test") {
		t.Error("bcc leaked into the message")
	}
	sent := list(t, h, c, "Sent")
	if len(sent) != 1 || sent[0].Subject != "Zdravo" || !sent[0].Seen {
		t.Fatalf("sent copy: %+v", sent)
	}
}

func TestSendValidation(t *testing.T) {
	h, c, _ := newComposeServer(t)
	cases := map[string]map[string]any{
		"no recipients":    {"identity": "default", "subject": "x", "text": "y"},
		"bad recipient":    {"identity": "default", "to": []string{"not an address"}, "text": "y"},
		"unknown identity": {"identity": "nope", "to": []string{"b@golub.test"}, "text": "y"},
		"header injection": {"identity": "default", "to": []string{"b@golub.test"}, "subject": "a\r\nBcc: x@y.z"},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := sendJSON(t, h, c, "/api/compose/send", body); rec.Code != 400 {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
		})
	}
	rec := sendJSON(t, h, c, "/api/compose/send", map[string]any{"identity": "default", "to": []string{"x@rejected.test"}, "text": "y"})
	if rec.Code != 422 || errCode(t, rec) != "compose.recipient_rejected" || !strings.Contains(rec.Body.String(), "x@rejected.test") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestSendHTMLIsSanitized(t *testing.T) {
	h, c, srv := newComposeServer(t)
	sendJSON(t, h, c, "/api/compose/send", map[string]any{
		"identity": "default", "to": []string{"b@golub.test"}, "subject": "h",
		"text": "plain", "html": `<p>Hi <b>there</b></p><script>alert(1)</script><img src=x onerror=alert(1)>`,
	})
	data := srv.Messages()[0].Data
	if strings.Contains(data, "<script") || strings.Contains(data, "onerror") || !strings.Contains(data, "<b>there</b>") {
		t.Fatalf("%s", data)
	}
}

func TestSendWithUploadAndForwardedAttachment(t *testing.T) {
	h, c, srv := newComposeServer(t, "05-attachment.eml")
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	meta, _ := json.Marshal(map[string]any{
		"identity": "default", "to": []string{"b@golub.test"}, "subject": "Fwd: With attachments", "text": "see",
		"attach":   []map[string]any{{"folder": "INBOX", "uid": 1, "part": attachmentID(t, h, c, "report.pdf")}},
		"answered": map[string]any{"folder": "INBOX", "uid": 1, "forwarded": true},
	})
	_ = w.WriteField("message", string(meta))
	fw, _ := w.CreateFormFile("attachment", "notes.txt")
	_, _ = fw.Write([]byte("uploaded"))
	_ = w.Close()
	r := httptest.NewRequest("POST", "/api/compose/send", &body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.AddCookie(c)
	rec := serve(h, r)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	data := srv.Messages()[0].Data
	if !strings.Contains(data, "notes.txt") || !strings.Contains(data, "report.pdf") {
		t.Fatalf("attachments missing:\n%s", data)
	}
}

func attachmentID(t *testing.T, h http.Handler, c *http.Cookie, name string) string {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/message?folder=INBOX&uid=1&seen=0", nil)
	r.AddCookie(c)
	var v struct {
		Attachments []struct{ ID, Filename string }
	}
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &v)
	for _, a := range v.Attachments {
		if a.Filename == name {
			return a.ID
		}
	}
	t.Fatalf("no attachment %s", name)
	return ""
}

func TestComposeSource(t *testing.T) {
	h, c, _ := newComposeServer(t, "01-plain.eml")
	get := func(mode string) (int, map[string]any) {
		r := httptest.NewRequest("GET", "/api/compose/source?mode="+mode+"&folder=INBOX&uid=1", nil)
		r.AddCookie(c)
		rec := serve(h, r)
		var v map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &v)
		return rec.Code, v
	}
	code, v := get("reply")
	if code != 200 || v["subject"] != "Re: Plain text" || v["inReplyTo"] != "<plain-text@golub.test>" {
		t.Fatalf("%d %+v", code, v)
	}
	to := v["to"].([]any)
	if len(to) != 1 || !strings.Contains(to[0].(string), "bob@golub.test") {
		t.Fatalf("reply to %+v", to)
	}
	quote := v["quote"].(map[string]any)
	if !strings.Contains(quote["text"].(string), "čćžšđ") {
		t.Errorf("quote %+v", quote)
	}

	_, v = get("replyAll")
	if cc, _ := v["cc"].([]any); len(cc) != 0 {
		t.Errorf("self must not be in cc: %+v", cc)
	}
	_, v = get("forward")
	if v["subject"] != "Fwd: Plain text" || v["to"] != nil && len(v["to"].([]any)) != 0 {
		t.Errorf("forward %+v", v)
	}
	_, v = get("edit")
	if a, ok := v["attachments"].([]any); !ok || len(a) != 0 {
		t.Errorf("attachments must be an empty list, got %#v", v["attachments"])
	}
	if code, _ := get("bogus"); code != 400 {
		t.Errorf("bogus mode: %d", code)
	}
}

func TestReplyMarksAnswered(t *testing.T) {
	h, c, _ := newComposeServer(t, "01-plain.eml")
	rec := sendJSON(t, h, c, "/api/compose/send", map[string]any{
		"identity": "default", "to": []string{"bob@golub.test"}, "subject": "Re: Plain text", "text": "ok",
		"inReplyTo": "<plain-text@golub.test>", "answered": map[string]any{"folder": "INBOX", "uid": 1},
	})
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if m := list(t, h, c, "INBOX"); !m[0].Answered {
		t.Fatalf("%+v", m)
	}
}

func TestSaveDraftReplacesPrevious(t *testing.T) {
	h, c, srv := newComposeServer(t)
	rec := sendJSON(t, h, c, "/api/compose/draft", map[string]any{"identity": "default", "subject": "Draft 1", "text": "a"})
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var d struct{ UID uint32 }
	_ = json.Unmarshal(rec.Body.Bytes(), &d)
	if d.UID == 0 {
		t.Fatalf("%s", rec.Body)
	}
	rec = sendJSON(t, h, c, "/api/compose/draft", map[string]any{"identity": "default", "subject": "Draft 2", "text": "b", "draftUid": d.UID})
	drafts := list(t, h, c, "Drafts")
	if rec.Code != 200 || len(drafts) != 1 || drafts[0].Subject != "Draft 2" {
		t.Fatalf("%d %+v", rec.Code, drafts)
	}
	if len(srv.Messages()) != 0 {
		t.Fatal("saving a draft must not send")
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &d)
	sendJSON(t, h, c, "/api/compose/send", map[string]any{"identity": "default", "to": []string{"b@golub.test"}, "subject": "Draft 2", "text": "b", "draftUid": d.UID})
	if n := len(list(t, h, c, "Drafts")); n != 0 {
		t.Fatalf("draft not removed after send: %d", n)
	}
}

func TestReceiptSendAndIgnore(t *testing.T) {
	h, c, srv := newComposeServer(t, "41-urgent-receipt.eml", "01-plain.eml")
	r := httptest.NewRequest("GET", "/api/message?folder=INBOX&uid=1&seen=0", nil)
	r.AddCookie(c)
	var v struct {
		ReceiptTo string
		Priority  int
	}
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &v)
	if v.ReceiptTo != "bob@golub.test" || v.Priority != 1 {
		t.Fatalf("%+v", v)
	}
	if rec := sendJSON(t, h, c, "/api/message/receipt", map[string]any{"folder": "INBOX", "uid": 1, "send": true}); rec.Code != 204 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	msgs := srv.Messages()
	if len(msgs) != 1 || msgs[0].To[0] != "bob@golub.test" || !strings.Contains(msgs[0].Data, "disposition-notification") {
		t.Fatalf("%+v", msgs)
	}
	r = httptest.NewRequest("GET", "/api/message?folder=INBOX&uid=1&seen=0", nil)
	r.AddCookie(c)
	var again struct{ ReceiptTo string }
	_ = json.Unmarshal(serve(h, r).Body.Bytes(), &again)
	if again.ReceiptTo != "" {
		t.Fatal("receipt must not be asked again after it was sent")
	}
	if rec := sendJSON(t, h, c, "/api/message/receipt", map[string]any{"folder": "INBOX", "uid": 2, "send": true}); rec.Code != 400 {
		t.Fatalf("message without receipt request: %d", rec.Code)
	}
}

func TestComposePriorityAndReceipt(t *testing.T) {
	h, c, srv := newComposeServer(t)
	sendJSON(t, h, c, "/api/compose/send", map[string]any{"identity": "default", "to": []string{"b@golub.test"}, "subject": "p", "text": "x", "priority": "high", "receipt": true})
	data := srv.Messages()[0].Data
	if !strings.Contains(data, "X-Priority: 1") || !strings.Contains(data, "Disposition-Notification-To") {
		t.Fatalf("%s", data)
	}
}

func TestMarkReadFolderOp(t *testing.T) {
	h, c, _ := newComposeServer(t, "01-plain.eml", "02-html.eml")
	if rec := post(t, h, c, "/api/folders/markread", `{"name":"INBOX"}`); rec.Code != 204 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	for _, m := range list(t, h, c, "INBOX") {
		if !m.Seen {
			t.Fatalf("%+v", m)
		}
	}
}

func TestUnsubscribeNeedsOneClick(t *testing.T) {
	h, c, _ := newComposeServer(t, "01-plain.eml")
	if rec := sendJSON(t, h, c, "/api/message/unsubscribe", map[string]any{"folder": "INBOX", "uid": 1}); rec.Code != 400 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}
