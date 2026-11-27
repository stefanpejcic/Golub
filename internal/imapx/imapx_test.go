// SPDX-License-Identifier: GPL-3.0-or-later

package imapx

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/config"
)

func startServer(t *testing.T) config.Endpoint {
	t.Helper()
	mem := imapmemserver.New()
	mem.AddUser(imapmemserver.NewUser("alice@golub.test", "secret"))
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
		InsecureAuth: true,
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	host, port, _ := net.SplitHostPort(l.Addr().String())
	p, _ := strconv.Atoi(port)
	return config.Endpoint{Host: host, Port: p, TLS: config.TLSNone}
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
