// SPDX-License-Identifier: GPL-3.0-or-later

// Package imapx connects to the IMAP server and wraps mailbox operations.
package imapx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/stefanpejcic/golub/internal/config"
)

// Errors returned by Dialer, mapped to API error codes by the HTTP layer.
var (
	ErrInvalidCredentials = errors.New("imap: invalid credentials")
	ErrUnavailable        = errors.New("imap: server unavailable")
)

// Dialer opens authenticated IMAP connections.
type Dialer struct {
	Endpoint config.Endpoint
	TLS      *tls.Config
	Timeout  time.Duration
}

func (d *Dialer) dial() (*imapclient.Client, error) {
	timeout := d.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	opts := &imapclient.Options{TLSConfig: d.TLS, Dialer: &net.Dialer{Timeout: timeout}}
	addr := d.Endpoint.Addr()
	switch d.Endpoint.TLS {
	case config.TLSImplicit:
		return imapclient.DialTLS(addr, opts)
	case config.TLSNone:
		return imapclient.DialInsecure(addr, opts)
	default:
		return imapclient.DialStartTLS(addr, opts)
	}
}

// Login connects and authenticates; the caller owns the returned client.
func (d *Dialer) Login(ctx context.Context, user, pass string) (*imapclient.Client, error) {
	c, err := d.dial()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	if err := c.Login(user, pass).Wait(); err != nil {
		_ = c.Close()
		var imapErr *imap.Error
		if errors.As(err, &imapErr) && imapErr.Type == imap.StatusResponseTypeNo {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return c, nil
}

// Check verifies credentials and closes the connection.
func (d *Dialer) Check(ctx context.Context, user, pass string) error {
	c, err := d.Login(ctx, user, pass)
	if err != nil {
		return err
	}
	_ = c.Logout().Wait()
	return c.Close()
}
