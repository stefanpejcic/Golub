// SPDX-License-Identifier: GPL-3.0-or-later

// Package smtptest runs an in-memory SMTP server for tests.
package smtptest

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/stefanpejcic/golub/internal/config"
)

// Message is one accepted submission.
type Message struct {
	User string
	From string
	To   []string
	Data string
}

// Server records what it accepts.
type Server struct {
	Endpoint config.Endpoint
	users    map[string]string
	mu       sync.Mutex
	messages []Message
}

// Messages returns a copy of everything received so far.
func (s *Server) Messages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.messages...)
}

type session struct {
	srv  *Server
	user string
	msg  Message
}

func (s *session) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *session) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, user, pass string) error {
		if p, ok := s.srv.users[user]; !ok || p != pass {
			return errors.New("invalid credentials")
		}
		s.user = user
		return nil
	}), nil
}
func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	if s.user == "" {
		return smtp.ErrAuthRequired
	}
	s.msg = Message{User: s.user, From: from}
	return nil
}
func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	if strings.HasSuffix(to, "@rejected.test") {
		return &smtp.SMTPError{Code: 550, Message: "no such user"}
	}
	s.msg.To = append(s.msg.To, to)
	return nil
}
func (s *session) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.msg.Data = string(b)
	s.srv.mu.Lock()
	s.srv.messages = append(s.srv.messages, s.msg)
	s.srv.mu.Unlock()
	return nil
}
func (s *session) Reset()        {}
func (s *session) Logout() error { return nil }

// Start serves SMTP on loopback for the duration of the test; users maps login to password.
func Start(t testing.TB, users map[string]string) *Server {
	t.Helper()
	s := &Server{users: users}
	srv := smtp.NewServer(smtp.BackendFunc(func(*smtp.Conn) (smtp.Session, error) { return &session{srv: s}, nil }))
	srv.AllowInsecureAuth = true
	srv.Domain = "localhost"
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	host, port, _ := net.SplitHostPort(l.Addr().String())
	p, _ := strconv.Atoi(port)
	s.Endpoint = config.Endpoint{Host: host, Port: p, TLS: config.TLSNone}
	return s
}
