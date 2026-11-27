#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# Prints the semantic version: the tag on HEAD for releases, otherwise VERSION-dev.<commit count>.
set -eu
cd "$(dirname "$0")/.."
if tag=$(git describe --tags --exact-match --match 'v*' 2>/dev/null); then
  echo "${tag#v}"
else
  echo "$(cat VERSION)-dev.$(git rev-list --count HEAD)"
fi
