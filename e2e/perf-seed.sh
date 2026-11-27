#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Fills alice's "Perf" folder with many messages by writing Maildir files directly, then indexes them.
# EXEC is the command that runs a shell in the mail server container.
set -euo pipefail

cd "$(dirname "$0")"
COUNT=${PERF_COUNT:-50000}
EXEC=${EXEC:-docker compose exec -T mailserver}
USER_ADDR="alice@golub.test"
DIR="/var/mail/golub.test/alice/.Perf"

echo "writing $COUNT messages into $DIR"
$EXEC sh -c "
  set -e
  rm -rf '$DIR' && mkdir -p '$DIR/cur' '$DIR/new' '$DIR/tmp' && touch '$DIR/maildirfolder'
  awk -v n=$COUNT -v dir='$DIR/cur' 'BEGIN {
    base = 1767225600
    for (i = 1; i <= n; i++) {
      t = base + i * 60
      f = sprintf(\"%s/%d.P%d.golub:2,S\", dir, t, i)
      printf \"From: Perf Sender <perf%d@golub.test>\r\n\", i % 97 > f
      printf \"To: alice@golub.test\r\nSubject: Perf message %06d\r\n\", i > f
      printf \"Date: %s\r\n\", strftime(\"%a, %d %b %Y %H:%M:%S +0000\", t, 1) > f
      printf \"Message-ID: <perf-%d@golub.test>\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nBody %d\r\n\", i, i > f
      close(f)
    }
  }'
  chown -R docker:docker '$DIR'
"
$EXEC doveadm mailbox subscribe -u "$USER_ADDR" Perf
start=$(date +%s)
$EXEC doveadm index -u "$USER_ADDR" Perf
echo "indexed in $(( $(date +%s) - start ))s"
$EXEC doveadm mailbox status -u "$USER_ADDR" messages Perf
