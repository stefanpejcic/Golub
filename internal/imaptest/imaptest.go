// SPDX-License-Identifier: GPL-3.0-or-later

// Package imaptest runs an in-memory IMAP server for tests.
package imaptest

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/stefanpejcic/golub/internal/config"
)

// Listen serves mem on a loopback port until the test ends.
func Listen(t testing.TB, mem *imapmemserver.Server) config.Endpoint {
	t.Helper()
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}, imap.CapSpecialUse: {}, imap.CapListStatus: {}},
		InsecureAuth: true,
	})
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	host, port, _ := net.SplitHostPort(l.Addr().String())
	p, _ := strconv.Atoi(port)
	return config.Endpoint{Host: host, Port: p, TLS: config.TLSNone}
}

// User creates a user with an INBOX plus the given extra folders.
func User(t testing.TB, mem *imapmemserver.Server, name, pass string, folders ...string) *imapmemserver.User {
	t.Helper()
	u := imapmemserver.NewUser(name, pass)
	for _, f := range append([]string{"INBOX"}, folders...) {
		if err := u.Create(f, nil); err != nil {
			t.Fatal(err)
		}
	}
	mem.AddUser(u)
	return u
}
