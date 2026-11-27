#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Appends testdata/mail fixtures; accounts and the master user come pre-provisioned from e2e/dms.
# Expects a freshly started stack: docker compose -f e2e/docker-compose.yml up -d --wait
set -euo pipefail

cd "$(dirname "$0")"
set -a
. ./test.env
set +a

IMAP="imap://127.0.0.1:${E2E_IMAP_PORT:-1143}"
ALICE="alice@${E2E_DOMAIN}"
BOB="bob@${E2E_DOMAIN}"
PAGING_COUNT=${PAGING_COUNT:-120}

imap() { curl -sS --fail "$@"; }

echo "waiting for dovecot to accept logins"
for i in $(seq 1 60); do
  if imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'NOOP' >/dev/null 2>&1 &&
     imap -u "$ALICE*$E2E_MASTER_USER:$E2E_MASTER_PASSWORD" "$IMAP/" -X 'NOOP' >/dev/null 2>&1; then
    break
  fi
  [ "$i" = 60 ] && { echo "logins never succeeded" >&2; exit 1; }
  sleep 2
done

existing=$(imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'STATUS INBOX (MESSAGES)' | grep -o 'MESSAGES [0-9]*' | cut -d' ' -f2)
if [ "${existing:-0}" != 0 ]; then
  echo "INBOX already has $existing messages; recreate the stack to reseed" >&2
  exit 1
fi

echo "appending fixtures"
fixtures=(../testdata/mail/*.eml)
for f in "${fixtures[@]}"; do
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/INBOX" -T "$f"
done

echo "appending $PAGING_COUNT paging messages"
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'CREATE Paging' >/dev/null
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
for i in $(seq 1 "$PAGING_COUNT"); do
  printf 'From: Bob <%s>\r\nTo: Alice <%s>\r\nSubject: Paging message %03d\r\nDate: %s\r\nMessage-ID: <paging-%d@%s>\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nMessage number %d.\r\n' \
    "$BOB" "$ALICE" "$i" "$(date -u -R -d "2026-01-01 +$i hours")" "$i" "$E2E_DOMAIN" "$i" >"$tmp"
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Paging" -T "$tmp"
done

# curl APPENDs with \Seen; reset so the mailbox looks like freshly delivered mail.
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/INBOX" -X 'STORE 1:* -FLAGS (\Seen)' >/dev/null
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Paging" -X 'STORE 1:* -FLAGS (\Seen)' >/dev/null
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/INBOX" -X 'STORE 1 +FLAGS (\Seen)' >/dev/null
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/INBOX" -X 'STORE 2 +FLAGS (\Flagged)' >/dev/null

count() { imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X "STATUS $1 (MESSAGES)" | grep -o 'MESSAGES [0-9]*' | cut -d' ' -f2; }
inbox=$(count INBOX)
paging=$(count Paging)
echo "INBOX=$inbox Paging=$paging"
[ "$inbox" = "${#fixtures[@]}" ] && [ "$paging" = "$PAGING_COUNT" ] || { echo "seed verification failed" >&2; exit 1; }
echo "seed ok"
