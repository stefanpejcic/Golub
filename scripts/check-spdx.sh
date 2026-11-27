#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# Fails if a source file lacks the SPDX license line near the top.
set -eu
cd "$(dirname "$0")/.."
missing=0
for f in $(git ls-files -co --exclude-standard '*.go' '*.ts' '*.mjs' '*.js' '*.svelte' '*.css' '*.sh' | grep -v -e '^internal/ui/dist/' -e '/node_modules/'); do
  if ! head -n 3 "$f" | grep -q 'SPDX-License-Identifier: GPL-3.0-or-later'; then
    echo "missing SPDX header: $f"
    missing=1
  fi
done
exit $missing
