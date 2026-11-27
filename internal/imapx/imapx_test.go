// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/config"
	"github.com/stefanpejcic/golub/internal/imaptest"
)

func startServer(t *testing.T) config.Endpoint {
	t.Helper()
	mem := imapmemserver.New()
	mem.AddUser(imapmemserver.NewUser("alice@golub.test", "secret"))
	return listen(t, mem)
}

func listen(t *testing.T, mem *imapmemserver.Server) config.Endpoint {
	t.Helper()
	return imaptest.Listen(t, mem)
}

func TestCheck(t *testing.T) {
	d := &Dialer{Endpoint: startServer(t)}
	ctx := context.Background()
	if err := d.Check(ctx, "alice@golub.test", "secret"); err != nil {
		t.Fatalf("valid login: %v", err)
	}
	if err := d.Check(ctx, "alice@golub.test", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
}

func TestUnavailable(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().(*net.TCPAddr)
	l.Close()
	d := &Dialer{Endpoint: config.Endpoint{Host: "127.0.0.1", Port: addr.Port, TLS: config.TLSNone}, Timeout: time.Second}
	if err := d.Check(context.Background(), "a", "b"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestStartTLSRequiredButMissing(t *testing.T) {
	ep := startServer(t)
	ep.TLS = config.TLSStartTLS
	d := &Dialer{Endpoint: ep, Timeout: time.Second}
	if err := d.Check(context.Background(), "alice@golub.test", "secret"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}
