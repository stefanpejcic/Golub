#!/bin/bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Load test for the README: 10 accounts with the same mailbox, each sending one message-list request (50 messages) at a time.
# Peak RAM is the cgroup peak of every container a client needs, new IMAP connections are counted at the mailserver.
# Needs accounts bench1..bench10@golub.test, each a copy of alice's mailbox with the Perf folder from perf-seed.sh,
# and for snappymail and afterlogic the sessions recorded by sessions.mjs next to this script. Run as root on the bench host.
set -u
cd "$(dirname "$0")/.."
P=$(grep '^E2E_PASSWORD=' test.env | cut -d= -f2-)
MS=golub-e2e_mailserver_1
N=10; REQ=${REQ:-200}
CLIENTS=${CLIENTS:-golub roundcube snappymail afterlogic}
BENCH=$(pwd)/bench
d=$(mktemp -d); cd "$d"
mspid=$(podman inspect $MS --format '{{.State.Pid}}')
ipt() { nsenter -t "$mspid" -n iptables "$@"; }

port() { case $1 in golub) echo 18083 ;; roundcube) echo 18082 ;; snappymail) echo 18084 ;; afterlogic) echo 18086 ;; esac; }
containers() { if [ "$1" = afterlogic ]; then echo bench-afterlogic bench-afterlogic-db; else echo "bench-$1"; fi; }
wait_up() { for _ in $(seq 1 90); do curl -s -o /dev/null "$1" && return 0; sleep 1; done; return 1; }

# session i for a client: cookie in c$i, extra headers in h$i, request body per folder in b$i.<folder>
prepare() {
  local c=$1 i U
  case $c in
    golub)
      for i in $(seq 1 $N); do
        U=bench$i@golub.test
        curl -s -c g$i -H 'Content-Type: application/json' --data "{\"email\":\"$U\",\"password\":\"$P\"}" 127.0.0.1:18083/api/login -o /dev/null
        echo "golub_session=$(awk '$6=="golub_session"{print $7}' g$i)" > c$i; : > h$i
      done ;;
    roundcube)
      for i in $(seq 1 $N); do
        U=bench$i@golub.test
        curl -s -c r$i -b r$i '127.0.0.1:18082/?_task=login' -o l$i.html
        local tok; tok=$(grep -o 'name="_token" value="[^"]*"' l$i.html | head -1 | cut -d'"' -f4)
        curl -s -c r$i -b r$i -o /dev/null --data-urlencode "_token=$tok" --data "_task=login&_action=login&_timezone=UTC" --data-urlencode "_user=$U" --data-urlencode "_pass=$P" '127.0.0.1:18082/?_task=login'
        curl -s -c r$i -b r$i '127.0.0.1:18082/?_task=mail&_mbox=INBOX' -o m$i.html
        echo "X-Roundcube-Request: $(grep -o '"request_token":"[^"]*"' m$i.html | head -1 | cut -d'"' -f4)" > h$i
        awk '$6 ~ /^roundcube_/{printf "%s=%s; ", $6, $7}' r$i > c$i
      done ;;
    *)
      python3 - "$BENCH/sessions-$c.json" <<'PY'
import json, sys
for i, s in enumerate(json.load(open(sys.argv[1]))[:10], 1):
    open(f'c{i}', 'w').write(s['cookie'])
    open(f'h{i}', 'w').write(''.join(f'{k}: {v}\n' for k, v in s['headers'].items() if k.lower() != 'content-type'))
    open(f't{i}', 'w').write(s['headers'].get('content-type', ''))
    open('url', 'w').write(s['url'])
    for folder, body in s['bodies'].items():
        open(f'b{i}.{folder}', 'w').write(body)
PY
      ;;
  esac
}

request() { # client session folder -> runs ab
  local c=$1 i=$2 f=$3 p; p=$(port "$c")
  local hdr=(); while IFS= read -r line; do [ -n "$line" ] && hdr+=(-H "$line"); done < h$i
  case $c in
    golub) ab -n "$REQ" -c 1 -C "$(cat c$i)" "http://127.0.0.1:$p/api/messages?folder=$f&page=1&size=50" ;;
    roundcube) ab -n "$REQ" -c 1 -C "$(cat c$i)" "${hdr[@]}" "http://127.0.0.1:$p/?_task=mail&_action=list&_mbox=$f&_page=1&_remote=1" ;;
    *) ab -n "$REQ" -c 1 -C "$(cat c$i)" "${hdr[@]}" -p "b$i.$f" -T "$(cat t$i)" "http://127.0.0.1:$p$(cat url)" ;;
  esac
}

run() { # client folder
  local c=$1 f=$2 p; p=$(port "$c")
  # a fresh start resets the cgroup peak; recorded sessions survive it
  podman restart $(containers "$c") >/dev/null
  wait_up "http://127.0.0.1:$p/" || { echo "$c did not come back"; return; }
  sleep 5
  prepare "$c"
  ipt -I INPUT -p tcp --dport 143 --syn -m comment --comment benchimap -j ACCEPT
  local start pids="" i
  start=$(date +%s.%N)
  for i in $(seq 1 $N); do request "$c" "$i" "$f" > ab$i.txt 2>&1 & pids="$pids $!"; done
  wait $pids
  local total conns peak=0 ct
  total=$(echo "$(date +%s.%N) - $start" | bc)
  conns=$(ipt -L INPUT -v -x -n | awk '/benchimap/{print $1}')
  ipt -D INPUT -p tcp --dport 143 --syn -m comment --comment benchimap -j ACCEPT
  for ct in $(containers "$c"); do
    peak=$((peak + $(cat "/sys/fs/cgroup$(podman inspect "$ct" --format '{{.State.CgroupPath}}')/memory.peak")))
  done
  printf "%-10s %-6s %7.1f req/s  mean %6.1f ms  worst p95 %5s ms  non-2xx %s  imap connections %s  peak RAM %.1f MB\n" "$c" "$f" \
    "$(echo "$N * $REQ / $total" | bc -l)" \
    "$(cat ab*.txt | awk '/Time per request.*\(mean\)$/{s+=$4;n++} END{print s/n}')" \
    "$(cat ab*.txt | awk '/ 95%/{if($2>m)m=$2} END{print m}')" \
    "$(cat ab*.txt | awk '/Non-2xx/{s+=$3} END{print s+0}')" \
    "$conns" "$(echo "$peak / 1048576" | bc -l)"
  rm -f ab*.txt
}

for f in INBOX Perf; do for c in $CLIENTS; do run "$c" "$f"; done; done
cd /; rm -rf "$d"
echo DONE
