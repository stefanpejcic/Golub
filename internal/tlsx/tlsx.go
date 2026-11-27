// SPDX-License-Identifier: GPL-3.0-or-later

// Package tlsx builds TLS configs and reloads server certificates from disk.
package tlsx

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// CheckInterval is how often the cert file mtime is checked during handshakes.
const CheckInterval = time.Minute

// Reloader serves a certificate pair and picks up renewed files without a restart.
type Reloader struct {
	certPath, keyPath string
	now               func() time.Time

	mu        sync.Mutex
	cert      *tls.Certificate
	mtime     time.Time
	lastCheck time.Time
}

// NewReloader loads and validates the pair, failing if it is unusable.
func NewReloader(certPath, keyPath string) (*Reloader, error) {
	r := &Reloader{certPath: certPath, keyPath: keyPath, now: time.Now}
	if err := r.Reload(); err != nil {
		return nil, err
	}
	return r, nil
}

// Reload reads the pair from disk; on error the current certificate stays in use.
func (r *Reloader) Reload() error {
	st, err := os.Stat(r.certPath)
	if err != nil {
		return fmt.Errorf("tls cert: %w", err)
	}
	if _, err := os.Stat(r.keyPath); err != nil {
		return fmt.Errorf("tls key: %w", err)
	}
	c, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return fmt.Errorf("tls pair %s, %s: %w", r.certPath, r.keyPath, err)
	}
	r.mu.Lock()
	r.cert = &c
	r.mtime = st.ModTime()
	r.lastCheck = r.now()
	r.mu.Unlock()
	return nil
}

// GetCertificate is meant for tls.Config.GetCertificate.
func (r *Reloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	due := r.now().Sub(r.lastCheck) >= CheckInterval
	if due {
		r.lastCheck = r.now()
	}
	mtime := r.mtime
	r.mu.Unlock()

	if due {
		if st, err := os.Stat(r.certPath); err == nil && !st.ModTime().Equal(mtime) {
			if err := r.Reload(); err != nil {
				slog.Error("tls reload failed, keeping current certificate", "err", err)
			} else {
				slog.Info("tls certificate reloaded", "path", r.certPath)
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cert, nil
}

// ServerConfig returns the HTTPS server config backed by r.
func ServerConfig(r *Reloader) *tls.Config {
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: r.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	}
}

// ClientConfig returns the config for connecting to a mail server, trusting caPath in addition to system roots.
func ClientConfig(serverName, caPath string) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}
	if caPath == "" {
		return cfg, nil
	}
	pem, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("GOLUB_TLS_CA: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("GOLUB_TLS_CA: no certificates found in " + caPath)
	}
	cfg.RootCAs = pool
	return cfg, nil
}
