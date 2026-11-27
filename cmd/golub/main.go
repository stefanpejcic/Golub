// SPDX-License-Identifier: GPL-3.0-or-later

// Command golub runs the Golub webmail server.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stefanpejcic/golub/internal/auth"
	"github.com/stefanpejcic/golub/internal/config"
	"github.com/stefanpejcic/golub/internal/httpapi"
	"github.com/stefanpejcic/golub/internal/imapx"
	"github.com/stefanpejcic/golub/internal/proxy"
	"github.com/stefanpejcic/golub/internal/tlsx"
	"github.com/stefanpejcic/golub/internal/ui"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Println(version)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Getenv, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "golub:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string, logOut io.Writer) error {
	cfg, err := config.Load(getenv, config.SystemLookup)
	if err != nil {
		return fmt.Errorf("invalid configuration:\n%w", err)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(logOut, &slog.HandlerOptions{Level: level})))

	codec, err := auth.NewCodec(cfg.SecretKey)
	if err != nil {
		return err
	}
	imapTLS, err := tlsx.ClientConfig(cfg.IMAP.Host, cfg.TLSCA)
	if err != nil {
		return err
	}
	pool := imapx.NewPool(&imapx.Dialer{Endpoint: cfg.IMAP, TLS: imapTLS}, 5*time.Minute)
	defer pool.Close()

	opts := httpapi.Options{
		Version:        version,
		Static:         ui.FS(),
		HTTPS:          cfg.HTTPS(),
		Codec:          codec,
		Mail:           pool,
		Pool:           pool,
		Proxy:          proxy.New(cfg.SecretKey),
		MasterUser:     cfg.MasterUser,
		MasterPassword: cfg.MasterPassword,
		SessionTTL:     cfg.SessionTTL,
		DefaultLocale:  cfg.DefaultLocale,
		DefaultTheme:   cfg.DefaultTheme,
		TrustProxy:     cfg.TrustProxy,
	}
	if cfg.SSOEnabled() {
		opts.SSO = auth.NewSSOVerifier(cfg.SSOSecret)
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           httpapi.New(opts),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn),
	}

	if cfg.HTTPS() {
		reloader, err := tlsx.NewReloader(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsx.ServerConfig(reloader)
		go reloadOnHUP(ctx, reloader)
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Listen, "https", cfg.HTTPS(), "version", version)
		if cfg.HTTPS() {
			errc <- srv.ListenAndServeTLS("", "")
		} else {
			errc <- srv.ListenAndServe()
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func reloadOnHUP(ctx context.Context, r *tlsx.Reloader) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			if err := r.Reload(); err != nil {
				slog.Error("tls reload failed, keeping current certificate", "err", err)
			} else {
				slog.Info("tls certificate reloaded on SIGHUP")
			}
		}
	}
}
