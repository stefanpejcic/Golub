// SPDX-License-Identifier: GPL-3.0-or-later

package smtpx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stefanpejcic/golub/internal/config"
	"github.com/stefanpejcic/golub/internal/smtptest"
)

func TestSend(t *testing.T) {
	srv := smtptest.Start(t, map[string]string{"alice@golub.test": "secret"})
	s := &Sender{Endpoint: srv.Endpoint, Timeout: 5 * time.Second}
	msg := "From: alice@golub.test\r\nSubject: hi\r\n\r\nbody\r\n"
	err := s.Send(context.Background(), "alice@golub.test", "secret", "alice@golub.test", []string{"bob@golub.test", "carol@golub.test"}, []byte(msg))
	if err != nil {
		t.Fatal(err)
	}
	got := srv.Messages()
	if len(got) != 1 || got[0].User != "alice@golub.test" || got[0].From != "alice@golub.test" || len(got[0].To) != 2 || !strings.Contains(got[0].Data, "Subject: hi") {
		t.Fatalf("%+v", got)
	}
}

func TestSendErrors(t *testing.T) {
	srv := smtptest.Start(t, map[string]string{"alice@golub.test": "secret"})
	s := &Sender{Endpoint: srv.Endpoint, Timeout: 5 * time.Second}
	ctx := context.Background()
	err := s.Send(ctx, "alice@golub.test", "wrong", "alice@golub.test", []string{"bob@golub.test"}, []byte("x"))
	if !errors.Is(err, ErrAuth) {
		t.Errorf("bad password: %v", err)
	}
	err = s.Send(ctx, "alice@golub.test", "secret", "alice@golub.test", []string{"x@rejected.test"}, []byte("x"))
	var rej *RecipientError
	if !errors.As(err, &rej) || rej.Address != "x@rejected.test" {
		t.Errorf("rejected rcpt: %v", err)
	}
	if err := s.Send(ctx, "a", "b", "a@b.c", nil, []byte("x")); err == nil {
		t.Error("no recipients must fail")
	}

	dead := config.Endpoint{Host: "127.0.0.1", Port: 1, TLS: config.TLSNone}
	if err := (&Sender{Endpoint: dead, Timeout: time.Second}).Send(ctx, "a", "b", "a@b.c", []string{"x@y.z"}, []byte("x")); !errors.Is(err, ErrUnavailable) {
		t.Errorf("unreachable: %v", err)
	}
}
