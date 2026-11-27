// SPDX-License-Identifier: GPL-3.0-or-later

package tlsx

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePair(t *testing.T, dir, cn string) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		DNSNames:              []string{cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath = filepath.Join(dir, cn+".crt")
	keyPath = filepath.Join(dir, cn+".key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func cn(t *testing.T, c *tls.Certificate) string {
	t.Helper()
	x, err := x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return x.Subject.CommonName
}

func TestNewReloaderValidation(t *testing.T) {
	dir := t.TempDir()
	aCert, aKey := writePair(t, dir, "a.test")
	_, bKey := writePair(t, dir, "b.test")
	garbage := filepath.Join(dir, "garbage")
	if err := os.WriteFile(garbage, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := map[string][2]string{
		"missing cert": {filepath.Join(dir, "none.crt"), aKey},
		"missing key":  {aCert, filepath.Join(dir, "none.key")},
		"mismatch":     {aCert, bKey},
		"garbage cert": {garbage, aKey},
	}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewReloader(p[0], p[1]); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	r, err := NewReloader(aCert, aKey)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := cn(t, c); got != "a.test" {
		t.Errorf("cn = %q", got)
	}
}

func TestReloadOnMtimeChange(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writePair(t, dir, "a.test")
	r, err := NewReloader(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	r.now = func() time.Time { return now }

	newCert, newKey := writePair(t, dir, "b.test")
	replace(t, newCert, certPath)
	replace(t, newKey, keyPath)
	future := now.Add(2 * time.Second)
	if err := os.Chtimes(certPath, future, future); err != nil {
		t.Fatal(err)
	}

	c, _ := r.GetCertificate(nil)
	if got := cn(t, c); got != "a.test" {
		t.Errorf("reloaded before interval: %q", got)
	}

	now = now.Add(CheckInterval + time.Second)
	c, _ = r.GetCertificate(nil)
	if got := cn(t, c); got != "b.test" {
		t.Errorf("not reloaded after interval: %q", got)
	}
}

func TestFailedReloadKeepsOld(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writePair(t, dir, "a.test")
	r, err := NewReloader(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certPath, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.Reload(); err == nil {
		t.Fatal("expected reload error")
	}
	c, _ := r.GetCertificate(nil)
	if got := cn(t, c); got != "a.test" {
		t.Errorf("cn = %q", got)
	}
}

func TestExplicitReload(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writePair(t, dir, "a.test")
	r, err := NewReloader(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	newCert, newKey := writePair(t, dir, "b.test")
	replace(t, newCert, certPath)
	replace(t, newKey, keyPath)
	if err := r.Reload(); err != nil {
		t.Fatal(err)
	}
	c, _ := r.GetCertificate(nil)
	if got := cn(t, c); got != "b.test" {
		t.Errorf("cn = %q", got)
	}
}

func TestServerConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writePair(t, dir, "a.test")
	r, err := NewReloader(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ServerConfig(r)
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x", cfg.MinVersion)
	}
	if cfg.GetCertificate == nil {
		t.Error("GetCertificate not set")
	}
}

func TestClientConfig(t *testing.T) {
	dir := t.TempDir()
	caPath, _ := writePair(t, dir, "ca.test")

	cfg, err := ClientConfig("mail.test", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RootCAs != nil || cfg.ServerName != "mail.test" || cfg.InsecureSkipVerify {
		t.Errorf("unexpected default client config: %+v", cfg)
	}

	cfg, err = ClientConfig("mail.test", caPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RootCAs == nil {
		t.Error("RootCAs not set")
	}

	if _, err := ClientConfig("mail.test", filepath.Join(dir, "missing.pem")); err == nil {
		t.Error("expected error for missing CA")
	}
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("not pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ClientConfig("mail.test", bad); err == nil {
		t.Error("expected error for CA without certificates")
	}
}

func replace(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
}
