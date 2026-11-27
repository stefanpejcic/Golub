// SPDX-License-Identifier: GPL-3.0-or-later

package sanitize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stefanpejcic/golub/internal/mime"
)

func fixture(t *testing.T, name string) *mime.Message {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mail", name))
	if err != nil {
		t.Fatal(err)
	}
	m, err := mime.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

var forbidden = []string{
	"<script", "onload", "onerror", "javascript:", "<meta", "<base", "<form", "<input", "<iframe",
	"<object", "<svg", "expression(", "@import", "url(", "data:text/html", "evil.test", "<style",
}

func TestXSSFixtures(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "mail", "2*-xss-*.eml"))
	if len(files) < 7 {
		t.Fatalf("expected xss fixtures, got %d", len(files))
	}
	for _, f := range files {
		for _, remote := range []bool{false, true} {
			t.Run(filepath.Base(f), func(t *testing.T) {
				m := fixture(t, filepath.Base(f))
				out := HTML(m.HTML, Options{RemoteImages: remote, Proxy: func(u string) string { return "/proxy?u=" + u }}).HTML
				low := strings.ToLower(out)
				for _, bad := range forbidden {
					if strings.Contains(low, bad) {
						t.Errorf("remote=%v output contains %q:\n%s", remote, bad, out)
					}
				}
			})
		}
	}
}

func TestKeepsFormatting(t *testing.T) {
	in := `<table width="600" cellpadding="4" style="border-collapse:collapse"><tr><td align="center" bgcolor="#eeeeee" style="color:#333;font-size:14px;padding:8px">Hi <b>there</b></td></tr></table><p style="position:fixed;color:red">x</p>`
	out := HTML(in, Options{}).HTML
	for _, want := range []string{`width="600"`, `align="center"`, `bgcolor="#eeeeee"`, "color: #333", "font-size: 14px", "<b>there</b>", "color: red"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "position") {
		t.Errorf("position allowed: %s", out)
	}
}

func TestLinks(t *testing.T) {
	out := HTML(`<a href="https://example.com/x">a</a><a href="mailto:bob@x.test">m</a><a href="/relative">r</a>`, Options{}).HTML
	if !strings.Contains(out, `href="https://example.com/x"`) || !strings.Contains(out, `target="_blank"`) || !strings.Contains(out, "noopener") || !strings.Contains(out, "noreferrer") {
		t.Errorf("link not hardened: %s", out)
	}
	if !strings.Contains(out, `href="mailto:bob@x.test"`) {
		t.Errorf("mailto dropped: %s", out)
	}
	if strings.Contains(out, "/relative") {
		t.Errorf("relative link kept: %s", out)
	}
}

func TestRemoteImagesBlockedByDefault(t *testing.T) {
	m := fixture(t, "02-html.eml")
	r := HTML(m.HTML, Options{})
	if r.BlockedImages != 1 || r.Trackers != 1 || strings.Contains(r.HTML, "example.com/") && strings.Contains(r.HTML, "<img") {
		t.Fatalf("%+v", r)
	}
	r = HTML(m.HTML, Options{RemoteImages: true, Proxy: func(u string) string { return "/proxy?u=" + u }})
	if r.BlockedImages != 0 || !strings.Contains(r.HTML, `src="/proxy?u=https://example.com/banner.png"`) || strings.Contains(r.HTML, "track.png") {
		t.Fatalf("%+v", r)
	}
}

func TestCIDBecomesDataURI(t *testing.T) {
	m := fixture(t, "04-inline-cid.eml")
	r := HTML(m.HTML, Options{CID: func(id string) (string, bool) {
		p := m.CID(id)
		if p == nil {
			return "", false
		}
		return DataURI(p.ContentType, p.Content), true
	}})
	if !strings.Contains(r.HTML, `src="data:image/png;base64,iVBOR`) {
		t.Fatalf("%s", r.HTML)
	}
	if strings.Contains(r.HTML, "cid:") {
		t.Errorf("cid left: %s", r.HTML)
	}
}

func TestDataURIOnlyForImages(t *testing.T) {
	if DataURI("image/svg+xml", []byte("<svg/>")) != "" || DataURI("text/html", []byte("x")) != "" {
		t.Fatal("unsafe data uri allowed")
	}
}

func TestDocumentWrapper(t *testing.T) {
	doc := Document("<p>x</p>", false)
	if !strings.HasPrefix(doc, "<!doctype html>") || !strings.Contains(doc, `<meta charset="utf-8">`) || !strings.Contains(doc, "<p>x</p>") {
		t.Fatal(doc)
	}
}

func TestTrackingPixelsRemovedEvenWhenImagesAllowed(t *testing.T) {
	in := `<p>hi</p>` +
		`<img src="https://t.example/open.gif" width="1" height="1">` +
		`<img src="https://t.example/o2.gif" width="0" height="0">` +
		`<img src="https://t.example/o3.gif" style="display:none">` +
		`<img src="https://t.example/o4.gif" style="width: 1px; height: 1px">` +
		`<img src="https://cdn.example/logo.png" width="120" height="40">`
	r := HTML(in, Options{RemoteImages: true, Proxy: func(u string) string { return "/proxy?u=" + u }})
	if strings.Contains(r.HTML, "t.example") {
		t.Fatalf("tracker kept: %s", r.HTML)
	}
	if !strings.Contains(r.HTML, "/proxy?u=https://cdn.example/logo.png") {
		t.Fatalf("real image dropped: %s", r.HTML)
	}
	if r.Trackers != 4 {
		t.Errorf("trackers = %d", r.Trackers)
	}
	blocked := HTML(in, Options{})
	if blocked.BlockedImages != 1 || blocked.Trackers != 4 {
		t.Errorf("without consent: blocked %d trackers %d", blocked.BlockedImages, blocked.Trackers)
	}
}

func TestDocumentDark(t *testing.T) {
	light, dark := Document("<p>hi</p>", false), Document("<p>hi</p>", true)
	if strings.Contains(light, "invert") {
		t.Error("light view must not invert")
	}
	for _, want := range []string{"filter:invert(1) hue-rotate(180deg)", "img,", "background-image"} {
		if !strings.Contains(dark, want) {
			t.Errorf("dark view missing %q", want)
		}
	}
	if !strings.Contains(dark, "<p>hi</p>") {
		t.Error("body lost")
	}
}

func TestDocumentFollowsTextDirection(t *testing.T) {
	if !strings.Contains(Document("<p>مرحبا</p>", false), `<body dir="auto">`) {
		t.Error("the body has to pick its direction from the text, so Arabic and Hebrew mail reads right to left")
	}
}
