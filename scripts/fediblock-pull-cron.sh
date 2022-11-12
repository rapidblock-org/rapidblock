#!/bin/bash
set -euo pipefail

if [ -z "${NOW:+isset}" ]; then
  sleep $((RANDOM % 3600))
fi

readonly PUBLIC_KEY_FILE="/etc/fediblock/fediblock.pub"
readonly POSTGRESQL_URL="postgresql:///mastodon?host=/run/postgresql&port=5433"

tmproot="$(mktemp -d -t "fediblock-pull.$$.XXXXXXXX")"
trap 'cd /; rm -rf "$tmproot"' EXIT

cd "$tmproot"
curl --remote-name-all -fsSLR \
  https://chronos-tachyon.net/fediblock/blocklist.json \
  https://chronos-tachyon.net/fediblock/blocklist.json.sig
fediblock -m verify -p "$PUBLIC_KEY_FILE" -d blocklist.json -s blocklist.json.sig -t
fediblock -m apply -d blocklist.json -D "$POSTGRESQL_URL"
