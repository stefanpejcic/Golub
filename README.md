# Golub

Fast, lightweight, stateless webmail client written in Go. Single binary, no database, SSO-ready.

Golub talks IMAP and SMTP to any Dovecot + Postfix stack. The reference stack is [docker-mailserver](https://github.com/docker-mailserver/docker-mailserver).

[OpenPanel](https://openpanel.com) uses Golub as its default webmail client.

## Status

Early development. See [ROADMAP.md](ROADMAP.md).

## Image

```
ghcr.io/stefanpejcic/golub
```

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).

## Configuration

All settings come from environment variables and are validated at startup.

| Variable | Default | Notes |
|---|---|---|
| `GOLUB_LISTEN` | `:8080` | |
| `GOLUB_TLS_CERT`, `GOLUB_TLS_KEY` | | Both or neither. Reloaded on change and on SIGHUP. |
| `GOLUB_TRUST_PROXY` | | Comma-separated CIDRs allowed to set `X-Forwarded-*`. |
| `GOLUB_IMAP_HOST` | | Required. |
| `GOLUB_IMAP_PORT` | 993 / 143 | Depends on `GOLUB_IMAP_TLS`. |
| `GOLUB_IMAP_TLS` | `starttls` | `implicit`, `starttls` or `none` (private hosts only). |
| `GOLUB_SMTP_HOST` | IMAP host | |
| `GOLUB_SMTP_PORT` | 465 / 587 | Depends on `GOLUB_SMTP_TLS`. |
| `GOLUB_SMTP_TLS` | `starttls` | Same as IMAP. |
| `GOLUB_TLS_CA` | | Extra CA bundle for mail servers. |
| `GOLUB_SECRET_KEY` | | Required. 32 random bytes, base64: `openssl rand -base64 32`. |
| `GOLUB_SSO_SECRET` | | Enables SSO, at least 32 characters. |
| `GOLUB_MASTER_USER`, `GOLUB_MASTER_PASSWORD` | | Required with SSO. |
| `GOLUB_SESSION_TTL` | `12h` | |
| `GOLUB_DEFAULT_LOCALE` | `en` | `en` or `sr-Latn`. |
| `GOLUB_DEFAULT_THEME` | `system` | `system`, `light` or `dark`. |
| `GOLUB_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |

## Development

```
cd web && npm ci && npm run build && cd ..
go test -race ./...
go run ./cmd/golub
```

The frontend builds into `internal/ui/dist`, which the Go binary embeds.
