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
- [x] IDLE with polling fallback

### 9. Hardening
- [x] Perf budgets in CI (Server-Timing, 50k folder: first page, deep page, open)
- [x] Logo: carrier pigeon mascot in Go cyan with speed lines, plus a head mark for small sizes
- [ ] Tag `v0.1.0` (see Release below)

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

## 1.0 release, in this order
- [x] Drag and drop messages onto folders
- [x] IDNA and SMTPUTF8 addresses
- [x] Mobile layout with a folder select and full-screen message
- [x] Branding through env vars: name, logo, accent colour
- [x] In-app confirm and prompt dialogs
- [ ] Attachment previews (images, PDF, text)
- [ ] Spell checking via the browser
- [ ] Cache sorted UIDs per connection so paging and search on very large folders skip the full SORT
- [x] Address book with groups, autocomplete, addresses collected from Sent, vCard import/export (stored in a hidden IMAP folder)
- [x] Threaded view (IMAP THREAD)
- [x] Advanced search: body text, all folders, date range, has attachments, unread, flagged
- [x] Import/export: .eml/.mbox upload into a folder, folder export

## Release
- [ ] Benchmark against Roundcube on the test server: image and download size in MB, RAM usage, latency
- [ ] README: benchmark results and the Roundcube comparison
- [x] README rewrite: privacy first, feature list, quick start, SSO
- [ ] Tag `v0.1.0`, publish the Docker image and test it on the server

## Future versions

### Notifications and avatars
- [ ] Tab title with unread count ("(3) Inbox – Golub"), keep refreshing while the tab is in the background
- [ ] Favicon dot for new mail, optional desktop notifications (setting, browser permission)
- [ ] Avatars: initials in a stable colour by default, no network requests
- [ ] BIMI logos for senders whose mail passed DMARC (checked from the server's Authentication-Results), loaded through the proxy as images only
- [ ] Optional Gravatar (off by default, through the proxy), in-memory cache

### AI assistant (optional, privacy first)
Off unless the admin configures an endpoint (`GOLUB_AI_URL`, `GOLUB_AI_MODEL`, `GOLUB_AI_KEY`; OpenAI-compatible API, so a local Ollama or vLLM works). Nothing leaves the server unless the user clicks an AI action; no background analysis, no storage, users can turn it off in settings.
- [ ] Summarize a message or a long thread
- [ ] Reply suggestions: two or three short drafts in the user's language, inserted into compose for editing
- [ ] Compose helpers: rewrite, shorten, fix grammar, change tone, translate the draft
- [ ] Translate an incoming message in place
- [ ] Natural-language search turned into IMAP SEARCH ("invoices from Marko last month")
- [ ] Extract action items, dates and deadlines from a message
- [ ] Explain why a message looks like phishing or spam
- [ ] Admin limits: per-user rate limit, max input size, strip attachments and quoted history before sending

### Later
PGP, dark message view for HTML mail, RTL locales.

### Docs, website and demo

- [ ] Documentation: install, configuration reference, SSO integration, reverse proxy examples
- [ ] Project website
- [ ] Public live demo with throwaway accounts, reset on a schedule
