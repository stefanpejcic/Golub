// SPDX-License-Identifier: GPL-3.0-or-later

// Package smtpx submits messages to the SMTP server.
package smtpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/stefanpejcic/golub/internal/config"
)

// Errors mapped to API codes by the HTTP layer.
var (
	ErrAuth        = errors.New("smtp: authentication failed")
	ErrUnavailable = errors.New("smtp: server unavailable")
)

// RecipientError reports an address the server refused.
type RecipientError struct {
	Address string
	Err     error
}

func (e *RecipientError) Error() string {
	return fmt.Sprintf("smtp: recipient %s rejected: %v", e.Address, e.Err)
}
func (e *RecipientError) Unwrap() error { return e.Err }

// Sender submits mail with the user's credentials.
type Sender struct {
	Endpoint config.Endpoint
	TLS      *tls.Config
	Timeout  time.Duration
}

func (s *Sender) dial(ctx context.Context) (*smtp.Client, error) {
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	d := &net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", s.Endpoint.Addr())
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(2 * timeout))
	switch s.Endpoint.TLS {
	case config.TLSImplicit:
		return smtp.NewClient(tls.Client(conn, s.TLS)), nil
	case config.TLSNone:
		return smtp.NewClient(conn), nil
	default:
		return smtp.NewClientStartTLS(conn, s.TLS)
	}
}

// Send authenticates and submits msg from envelope sender from to every recipient.
func (s *Sender) Send(ctx context.Context, user, pass, from string, to []string, msg []byte) error {
	if len(to) == 0 {
		return errors.New("smtp: no recipients")
	}
	c, err := s.dial(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer c.Close()
	if err := c.Hello("golub"); err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := c.Auth(sasl.NewPlainClient("", user, pass)); err != nil {
		return fmt.Errorf("%w: %w", ErrAuth, err)
	}
	if err := c.Mail(from, nil); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt, nil); err != nil {
			return &RecipientError{Address: rcpt, Err: err}
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := bytes.NewReader(msg).WriteTo(w); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
