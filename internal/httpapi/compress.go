// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"compress/gzip"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

// precompressed serves name.br or name.gz from the embedded build when the browser accepts it.
func precompressed(w http.ResponseWriter, r *http.Request, static fs.FS, name string) bool {
	accept := r.Header.Get("Accept-Encoding")
	for _, enc := range []struct{ token, ext, header string }{{"br", ".br", "br"}, {"gzip", ".gz", "gzip"}} {
		if !strings.Contains(accept, enc.token) {
			continue
		}
		b, err := fs.ReadFile(static, name+enc.ext)
		if err != nil {
			continue
		}
		h := w.Header()
		ct := mime.TypeByExtension(path.Ext(name))
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
		h.Set("Content-Encoding", enc.header)
		h.Add("Vary", "Accept-Encoding")
		_, _ = w.Write(b) //nolint:gosec // our own embedded build output
		return true
	}
	return false
}

const minGzip = 1024

var gzipPool = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}

// gzipAPI compresses larger text responses from the API; downloads and the event stream pass through.
func gzipAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/events" ||
			!strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		cw := &gzipWriter{ResponseWriter: w}
		defer cw.finish()
		next.ServeHTTP(cw, r)
	})
}

type gzipWriter struct {
	http.ResponseWriter
	code    int
	buf     bytes.Buffer
	gz      *gzip.Writer
	decided bool
	plain   bool
}

func compressible(ct string) bool {
	mt, _, _ := mime.ParseMediaType(ct)
	return mt == "application/json" || mt == "text/html" || mt == "text/plain"
}

func (g *gzipWriter) WriteHeader(code int) {
	if g.code == 0 {
		g.code = code
	}
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if g.code == 0 {
		g.code = http.StatusOK
	}
	if !g.decided {
		if g.Header().Get("Content-Encoding") != "" || !compressible(g.Header().Get("Content-Type")) {
			g.start(false)
		} else {
			g.buf.Write(b)
			if g.buf.Len() < minGzip {
				return len(b), nil
			}
			g.start(true)
			return len(b), nil
		}
	}
	if g.plain {
		return g.ResponseWriter.Write(b)
	}
	return g.gz.Write(b)
}

// start sends the header and whatever was buffered, compressed or not.
func (g *gzipWriter) start(compress bool) {
	g.decided = true
	g.plain = !compress
	h := g.Header()
	if compress {
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
		g.ResponseWriter.WriteHeader(g.code)
		g.gz = gzipPool.Get().(*gzip.Writer)
		g.gz.Reset(g.ResponseWriter)
		_, _ = g.gz.Write(g.buf.Bytes())
	} else {
		g.ResponseWriter.WriteHeader(g.code)
		_, _ = g.ResponseWriter.Write(g.buf.Bytes()) //nolint:gosec // passes through what the handler already wrote
	}
	g.buf.Reset()
}

func (g *gzipWriter) finish() {
	if !g.decided {
		if g.code == 0 && g.buf.Len() == 0 {
			return
		}
		if g.code == 0 {
			g.code = http.StatusOK
		}
		g.start(false)
	}
	if g.gz != nil {
		_ = g.gz.Close()
		gzipPool.Put(g.gz)
		g.gz = nil
	}
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (g *gzipWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }
