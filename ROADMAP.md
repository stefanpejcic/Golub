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
- [x] Normal login, session cookie, logout
- [x] SSO token flow (`lang`, `theme`)

### 4. Folders and message list
- [x] Folder list with unread counts
- [x] Message list with server-side SORT and paging

### 5. Message view
- [x] MIME parsing, part selection
- [x] Sanitizer and iframe endpoint
- [x] cid images, attachments
- [x] Remote image proxy with SSRF guard

### 6. Mutations
- [x] Flags, move, delete

### 7. Compose and send
- [x] Reply/forward quoting, attachments, Sent copy

### 8. Live refresh
- [ ] IDLE with polling fallback

### 9. Hardening
- [ ] Perf budgets in CI
- [ ] Benchmark against Roundcube and other webmail clients on the test server (latency, memory, page weight), publish results
- [ ] README rewrite for release: drop "early development", position as a Roundcube alternative, clear feature list and quick start
- [x] Logo: carrier pigeon mascot in Go cyan with speed lines, plus a head mark for small sizes
- [ ] Tag `v0.1.0`

## Next: Settings
Stored per user in an unsubscribed IMAP folder `golub-settings` (one JSON message), since the reference server has no IMAP METADATA. Nothing is written by Golub itself.
- [x] Settings store (load/save, defaults, validation) and `/api/settings`
- [x] Settings page bottom-left: Preferences (general, mailbox view, displaying, composing, special folders, server), Folders, Identities, Responses, About
- [x] Folder management: create, rename, delete, subscribe, empty
- [x] Apply preferences: page size, mark-as-read timing, prefer plain text, remote images, show addresses, pretty dates, delete mode, special folder overrides
- [x] Quota usage bar under the message list (IMAP QUOTA)

## UI backlog (requested)
- [x] Message header: icons on the right for all headers and plain text / HTML view
- [x] Message toolbar: Reply, Reply all, Forward, Delete, More (print, download .eml, edit as new, show source, move to, copy to, open in new window)
- [x] Resizable folder / list / preview columns, widths remembered in the browser

## Notifications and avatars (requested)
- [ ] Tab title with unread count ("(3) Inbox – Golub"), keep refreshing while the tab is in the background
- [ ] Favicon dot for new mail, optional desktop notifications (setting, browser permission)
- [ ] Avatars: initials in a stable colour by default, no network requests
- [ ] BIMI logos for senders whose mail passed DMARC (checked from the server's Authentication-Results), loaded through the proxy as images only
- [ ] Optional Gravatar (off by default, through the proxy), in-memory cache

## Later
Search, threading, contacts, multiple identities, drag and drop, PGP, dark message view for HTML mail, RTL locales.
