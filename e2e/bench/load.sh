#!/bin/bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Load test for the README: 10 accounts with the same mailbox, each sending one list request at a time, against bench/compose.yml.
# Peak RAM comes from the container cgroup, new IMAP connections are counted at the mailserver. Run as root on the bench host.
# Needs accounts bench1..bench10@golub.test, each a copy of alice's mailbox with the Perf folder from perf-seed.sh.
set -u
cd "$(dirname "$0")/.."
P=$(grep '^E2E_PASSWORD=' test.env | cut -d= -f2-)
MS=golub-e2e_mailserver_1
N=10; REQ=200
d=$(mktemp -d); cd "$d"
mspid=$(podman inspect $MS --format '{{.State.Pid}}')
ipt() { nsenter -t "$mspid" -n iptables "$@"; }

wait_up() { for t in $(seq 1 60); do curl -s -o /dev/null "$1" && return 0; sleep 1; done; return 1; }

login() { # client i
  local U=bench$2@golub.test
  if [ "$1" = golub ]; then
    curl -s -c g$2 -H 'Content-Type: application/json' --data "{\"email\":\"$U\",\"password\":\"$P\"}" 127.0.0.1:18083/api/login -o /dev/null
    echo "golub_session=$(awk '$6=="golub_session"{print $7}' g$2)" > c$2; : > t$2
  else
    curl -s -c r$2 -b r$2 '127.0.0.1:18082/?_task=login' -o l$2.html
    local tok; tok=$(grep -o 'name="_token" value="[^"]*"' l$2.html | head -1 | cut -d'"' -f4)
    curl -s -c r$2 -b r$2 -o /dev/null --data-urlencode "_token=$tok" --data "_task=login&_action=login&_timezone=UTC" --data-urlencode "_user=$U" --data-urlencode "_pass=$P" '127.0.0.1:18082/?_task=login'
    curl -s -c r$2 -b r$2 '127.0.0.1:18082/?_task=mail&_mbox=INBOX' -o m$2.html
    grep -o '"request_token":"[^"]*"' m$2.html | head -1 | cut -d'"' -f4 > t$2
    awk '$6 ~ /^roundcube_/{printf "%s=%s; ", $6, $7}' r$2 > c$2
  fi
}

url() { # client folder
  if [ "$1" = golub ]; then echo "http://127.0.0.1:18083/api/messages?folder=$2&page=1&size=50"
  else echo "http://127.0.0.1:18082/?_task=mail&_action=list&_mbox=$2&_page=1&_remote=1"; fi
}

run() { # client folder
  local c=$1 f=$2 port=18083; [ $c = roundcube ] && port=18082
  podman restart bench-$c >/dev/null; wait_up "http://127.0.0.1:$port/" || { echo "$c did not come back"; return; }
  for i in $(seq 1 $N); do login $c $i; done
  ipt -I INPUT -p tcp --dport 143 --syn -m comment --comment benchimap -j ACCEPT
  local start pids="" i
  start=$(date +%s.%N)
  for i in $(seq 1 $N); do
    ab -n $REQ -c 1 -C "$(cat c$i)" -H "X-Roundcube-Request: $(cat t$i)" "$(url $c $f)" > ab$i.txt 2>&1 &
    pids="$pids $!"
  done
  wait $pids
  local total conns peak
  total=$(echo "$(date +%s.%N) - $start" | bc)
  conns=$(ipt -L INPUT -v -x -n | awk '/benchimap/{print $1}')
  ipt -D INPUT -p tcp --dport 143 --syn -m comment --comment benchimap -j ACCEPT
  peak=$(cat "/sys/fs/cgroup$(podman inspect bench-$c --format '{{.State.CgroupPath}}')/memory.peak")
  printf "%-10s %-6s %7.1f req/s  mean %6.1f ms  worst p95 %5s ms  non-2xx %s  imap connections %s  peak RAM %.1f MB\n" $c $f \
    "$(echo "$N * $REQ / $total" | bc -l)" \
    "$(cat ab*.txt | awk '/Time per request.*\(mean\)$/{s+=$4;n++} END{print s/n}')" \
    "$(cat ab*.txt | awk '/ 95%/{if($2>m)m=$2} END{print m}')" \
    "$(cat ab*.txt | awk '/Non-2xx/{s+=$3} END{print s+0}')" \
    "$conns" "$(echo "$peak / 1048576" | bc -l)"
}

for f in INBOX Perf; do run golub $f; run roundcube $f; done
cd /; rm -rf "$d"
echo DONE
