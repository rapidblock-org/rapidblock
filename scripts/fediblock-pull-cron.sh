#!/bin/bash
set -euo pipefail

if [ -z "${NOW:+isset}" ]; then
  sleep $((RANDOM % 3600))
fi

readonly PUBLIC_KEY_FILE="/etc/rapidblock/rapidblock.pub"
readonly POSTGRESQL_URL="postgresql:///mastodon?host=/run/postgresql&port=5433"

tmproot="$(mktemp -d -t "rapidblock-pull.$$.XXXXXXXX")"
trap 'cd /; rm -rf "$tmproot"' EXIT

cd "$tmproot"
curl --remote-name-all -fsSLR \
  https://rapidblock.org/blocklist.json \
  https://rapidblock.org/blocklist.json.sig
rapidblock -m verify -p "$PUBLIC_KEY_FILE" -d blocklist.json -s blocklist.json.sig -t
rapidblock -m apply -d blocklist.json -D "$POSTGRESQL_URL"
