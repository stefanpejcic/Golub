# Roadmap

## v0

### 1. Skeleton
- [x] Go module, config validation
- [x] Optional TLS with cert reload
- [x] `/healthz`
- [x] Dockerfile (distroless, non-root)
- [x] Svelte shell embedded
- [x] Theme tokens (light/dark) and toggle
- [x] i18n with `en` + `sr-Latn`, locale-key CI check
- [x] LICENSE, SPDX check
- [x] CI green

### 2. Test env
- [x] `e2e/docker-compose.yml` with docker-mailserver + Golub
- [x] Seed script appending `testdata/mail/*.eml`
- [x] Master user configured

### 3. Auth
- [ ] Normal login, session cookie, logout
- [ ] SSO token flow (`lang`, `theme`)

### 4. Folders and message list
- [ ] Folder list with unread counts
- [ ] Message list with server-side SORT and paging

### 5. Message view
- [ ] MIME parsing, part selection
- [ ] Sanitizer and iframe endpoint
- [ ] cid images, attachments
- [ ] Remote image proxy with SSRF guard

### 6. Mutations
- [ ] Flags, move, delete

### 7. Compose and send
- [ ] Reply/forward quoting, attachments, Sent copy

### 8. Live refresh
- [ ] IDLE with polling fallback

### 9. Hardening
- [ ] Perf budgets in CI
- [ ] Tag `v0.1.0`

## Later
Search, threading, Sieve filters, contacts, multiple identities, drag and drop, PGP, dark message view for HTML mail, RTL locales.
