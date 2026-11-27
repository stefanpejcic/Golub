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
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'SUBSCRIBE Paging' >/dev/null
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
for i in $(seq 1 "$PAGING_COUNT"); do
  printf 'From: Bob <%s>\r\nTo: Alice <%s>\r\nSubject: Paging message %03d\r\nDate: %s\r\nMessage-ID: <paging-%d@%s>\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nMessage number %d.\r\n' \
    "$BOB" "$ALICE" "$i" "$(date -u -R -d "2026-01-01 +$i hours")" "$i" "$E2E_DOMAIN" "$i" >"$tmp"
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Paging" -T "$tmp"
done

echo "appending action test messages"
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'CREATE Actions' >/dev/null
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'SUBSCRIBE Actions' >/dev/null
for i in 1 2 3 4 5; do
  printf 'From: Carol <carol@%s>\r\nTo: Alice <%s>\r\nSubject: Action %d\r\nDate: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nAction body %d.\r\n' \
    "$E2E_DOMAIN" "$ALICE" "$i" "$(date -u -R -d "2026-02-01 +$i hours")" "$i" >"$tmp"
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Actions" -T "$tmp"
done
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Actions" -X 'STORE 1:* -FLAGS (\Seen)' >/dev/null

echo "appending bulk test messages"
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'CREATE Bulk' >/dev/null
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'SUBSCRIBE Bulk' >/dev/null
for i in 1 2 3 4 5 6 7 8; do
  printf 'From: Dave <dave@%s>\r\nTo: Alice <%s>\r\nSubject: Bulk %d\r\nDate: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nBulk body %d.\r\n' \
    "$E2E_DOMAIN" "$ALICE" "$i" "$(date -u -R -d "2026-03-01 +$i hours")" "$i" >"$tmp"
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Bulk" -T "$tmp"
done
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Bulk" -X 'STORE 1:* -FLAGS (\Seen)' >/dev/null

# Threads holds two conversations and a lone message for the threaded view.
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'CREATE Threads' >/dev/null
imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X 'SUBSCRIBE Threads' >/dev/null
thread_msg() {
  # id, subject, hour, in-reply-to, references
  {
    printf 'From: Bob <bob@%s>\r\nTo: Alice <%s>\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@golub.test>\r\n' \
      "$E2E_DOMAIN" "$ALICE" "$2" "$(date -u -R -d "2026-04-01 +$3 hours")" "$1"
    [ -n "${4:-}" ] && printf 'In-Reply-To: <%s@golub.test>\r\nReferences: %s\r\n' "$4" "$5"
    printf 'Content-Type: text/plain; charset=utf-8\r\n\r\n%s body.\r\n' "$1"
  } >"$tmp"
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Threads" -T "$tmp"
}
thread_msg plan-1 "Trip plan" 1
thread_msg lone "Lone message" 2
thread_msg plan-2 "Re: Trip plan" 3 plan-1 "<plan-1@golub.test>"
thread_msg budget-1 "Budget" 4
thread_msg plan-3 "Re: Trip plan" 5 plan-2 "<plan-1@golub.test> <plan-2@golub.test>"
thread_msg budget-2 "Re: Budget" 6 budget-1 "<budget-1@golub.test>"

# Drag owns its two folders so the drag test never races the bulk tests.
for f in Drag Dropped; do
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X "CREATE $f" >/dev/null
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/" -X "SUBSCRIBE $f" >/dev/null
done
for i in 1 2; do
  printf 'From: Erin <erin@%s>\r\nTo: Alice <%s>\r\nSubject: Drag %d\r\nDate: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nDrag body %d.\r\n' \
    "$E2E_DOMAIN" "$ALICE" "$i" "$(date -u -R)" "$i" >"$tmp"
  imap -u "$ALICE:$E2E_PASSWORD" "$IMAP/Drag" -T "$tmp"
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
