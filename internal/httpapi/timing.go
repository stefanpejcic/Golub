// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// timed adds a Server-Timing header to API responses, so latency can be measured without the network.
func timed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/events" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&timingWriter{ResponseWriter: w, start: time.Now()}, r)
	})
}

type timingWriter struct {
	http.ResponseWriter
	start time.Time
	wrote bool
}

func (t *timingWriter) WriteHeader(code int) {
	if !t.wrote {
		t.wrote = true
		ms := float64(time.Since(t.start).Microseconds()) / 1000
		t.Header().Set("Server-Timing", fmt.Sprintf("app;dur=%.1f", ms))
	}
	t.ResponseWriter.WriteHeader(code)
}

func (t *timingWriter) Write(b []byte) (int, error) {
	if !t.wrote {
		t.WriteHeader(http.StatusOK)
	}
	return t.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (t *timingWriter) Unwrap() http.ResponseWriter { return t.ResponseWriter }
