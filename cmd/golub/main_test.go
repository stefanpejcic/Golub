// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestRunRejectsBadConfig(t *testing.T) {
	err := run(context.Background(), envFrom(map[string]string{}), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "GOLUB_IMAP_HOST") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunRejectsMissingCert(t *testing.T) {
	err := run(context.Background(), envFrom(map[string]string{
		"GOLUB_IMAP_HOST":  "127.0.0.1",
		"GOLUB_SECRET_KEY": base64.StdEncoding.EncodeToString(make([]byte, 32)),
		"GOLUB_TLS_CERT":   "/nonexistent/cert.pem",
		"GOLUB_TLS_KEY":    "/nonexistent/key.pem",
	}), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "tls") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunServesHealthz(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, envFrom(map[string]string{
			"GOLUB_LISTEN":     addr,
			"GOLUB_IMAP_HOST":  "127.0.0.1",
			"GOLUB_SECRET_KEY": base64.StdEncoding.EncodeToString(make([]byte, 32)),
		}), io.Discard)
	}()

	url := fmt.Sprintf("http://%s/healthz", addr)
	var resp *http.Response
	for i := 0; i < 50; i++ {
		if resp, err = http.Get(url); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Status, Version string }
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if body.Status != "ok" || body.Version != version {
		t.Errorf("body = %+v", body)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not stop")
	}
}
