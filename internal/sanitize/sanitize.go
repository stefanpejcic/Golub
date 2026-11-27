// SPDX-License-Identifier: GPL-3.0-or-later

// Package sanitize makes untrusted HTML mail safe to render in a sandboxed iframe.
package sanitize

import (
	"bytes"
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Options controls image handling.
type Options struct {
	// CID resolves a Content-ID to an inline data URI.
	CID func(id string) (string, bool)
	// RemoteImages loads remote images through Proxy instead of dropping them.
	RemoteImages bool
	// Proxy maps a remote image URL to a same-origin proxy URL.
	Proxy func(url string) string
}

// Result is sanitized HTML plus how many remote images were held back.
type Result struct {
	HTML          string
	BlockedImages int
}

var (
	colorRe  = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|[a-zA-Z]{3,20})$`)
	sizeRe   = regexp.MustCompile(`^[0-9]{1,4}%?$`)
	alignRe  = regexp.MustCompile(`(?i)^(left|right|center|justify)$`)
	valignRe = regexp.MustCompile(`(?i)^(top|middle|bottom|baseline)$`)
	policy   = newPolicy()
)

func newPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "hr", "div", "span", "b", "strong", "i", "em", "u", "s", "strike", "sub", "sup",
		"small", "big", "font", "center", "blockquote", "pre", "code", "h1", "h2", "h3", "h4", "h5", "h6",
		"abbr", "cite", "q", "dl", "dt", "dd", "address", "del", "ins", "mark", "wbr")
	p.AllowLists()
	p.AllowTables()
	p.AllowImages()
	p.AllowDataURIImages()

	p.AllowAttrs("href").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto", "cid")
	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(false)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.RequireNoReferrerOnLinks(true)
	p.RequireNoFollowOnLinks(true)

	p.AllowAttrs("align").Matching(alignRe).OnElements("p", "div", "table", "td", "th", "tr", "img", "h1", "h2", "h3", "h4", "h5", "h6", "center")
	p.AllowAttrs("valign").Matching(valignRe).OnElements("td", "th", "tr")
	p.AllowAttrs("width", "height").Matching(sizeRe).OnElements("table", "td", "th", "img", "col")
	p.AllowAttrs("bgcolor").Matching(colorRe).OnElements("table", "td", "th", "tr")
	p.AllowAttrs("border", "cellpadding", "cellspacing").Matching(sizeRe).OnElements("table")
	p.AllowAttrs("colspan", "rowspan").Matching(bluemonday.Integer).OnElements("td", "th")
	p.AllowAttrs("color").Matching(colorRe).OnElements("font")
	p.AllowAttrs("size").Matching(regexp.MustCompile(`^[+-]?[1-7]$`)).OnElements("font")
	p.AllowAttrs("face").Matching(regexp.MustCompile(`^[\w\s,'"-]{1,100}$`)).OnElements("font")
	p.AllowAttrs("dir").Matching(regexp.MustCompile(`(?i)^(ltr|rtl|auto)$`)).Globally()

	// No background images, positioning or anything taking a url().
	p.AllowStyles("color", "background-color", "font-size", "font-family", "font-weight", "font-style",
		"text-align", "text-decoration", "text-transform", "line-height", "letter-spacing", "white-space",
		"vertical-align", "width", "height", "max-width", "min-width", "max-height",
		"margin", "margin-top", "margin-right", "margin-bottom", "margin-left",
		"padding", "padding-top", "padding-right", "padding-bottom", "padding-left",
		"border", "border-top", "border-right", "border-bottom", "border-left", "border-color",
		"border-style", "border-width", "border-collapse", "border-spacing", "border-radius",
		"display", "list-style-type", "table-layout", "word-break", "overflow-wrap").Globally()
	return p
}

// HTML sanitizes body, inlines cid images and handles remote images.
func HTML(body string, o Options) Result {
	clean := policy.Sanitize(body)
	ctx := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(clean), ctx)
	if err != nil {
		return Result{}
	}
	var res Result
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if c.Type == html.ElementNode && c.DataAtom == atom.Img && !keepImage(c, o, &res) {
				n.RemoveChild(c)
			} else {
				walk(c)
			}
			c = next
		}
	}
	root := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	walk(root)

	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	res.HTML = buf.String()
	return res
}

// keepImage rewrites src in place and reports whether the img stays.
func keepImage(n *html.Node, o Options, res *Result) bool {
	for i, a := range n.Attr {
		if a.Key != "src" {
			continue
		}
		src := strings.TrimSpace(a.Val)
		lower := strings.ToLower(src)
		switch {
		case strings.HasPrefix(lower, "cid:"):
			if o.CID == nil {
				return false
			}
			uri, ok := o.CID(src[4:])
			if !ok || uri == "" {
				return false
			}
			n.Attr[i].Val = uri
			return true
		case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
			if !o.RemoteImages || o.Proxy == nil {
				res.BlockedImages++
				return false
			}
			n.Attr[i].Val = o.Proxy(src)
			return true
		case strings.HasPrefix(lower, "data:image/") && !strings.HasPrefix(lower, "data:image/svg"):
			return true
		default:
			return false
		}
	}
	return false
}

var inlineTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true, "image/bmp": true, "image/avif": true,
}

// DataURI encodes a raster image; anything else (including SVG) yields "".
func DataURI(contentType string, data []byte) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if !inlineTypes[ct] {
		return ""
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// Document wraps sanitized HTML for the message iframe. HTML mail assumes a light page.
func Document(body string) string {
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="color-scheme" content="light">` +
		`<style>html{background:#fff;color:#1b1f24}body{margin:16px;font:15px/1.5 system-ui,sans-serif;overflow-wrap:anywhere}` +
		`img{max-width:100%;height:auto}table{max-width:100%}blockquote{margin:0 0 0 .5em;padding-left:.75em;border-left:3px solid #d5d8de}</style>` +
		`</head><body>` + body + `</body></html>`
}
