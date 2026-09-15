#!/usr/bin/env bash
set -euo pipefail

archive=${1:-}

if [[ -z "$archive" ]]; then
  echo "usage: verify-bundle.sh <archive>" >&2
  exit 2
fi

if [[ ! -f "$archive" ]]; then
  echo "bundle archive not found: $archive" >&2
  exit 1
fi

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

tar -C "$workdir" -xzf "$archive"

mapfile -t roots < <(find "$workdir" -mindepth 1 -maxdepth 1 -type d -print)
if [[ ${#roots[@]} -ne 1 ]]; then
  echo "bundle must contain exactly one top-level directory" >&2
  exit 1
fi

root=${roots[0]}
required=(
  "leamout"
  "VERSION"
  "deploy/docker/compose.yaml"
  "deploy/docker/Caddyfile"
  "server/migrations/atlas.sum"
  "containers/nats/nats-server.conf"
  "containers/coturn/turnserver.conf"
  "server/scripts/certs/certbot-hook.sh"
  "server/scripts/certs/setup-certbot.sh"
  "server/scripts/certs/issue-certificates.sh"
  "server/scripts/deploy/lib.sh"
  "server/scripts/deploy/preflight.sh"
  "server/scripts/deploy/up.sh"
  "server/scripts/deploy/verify.sh"
  "server/scripts/deploy/restart.sh"
)

for path in "${required[@]}"; do
  if [[ ! -f "$root/$path" ]]; then
    echo "bundle missing required file: $path" >&2
    exit 1
  fi
done

for script in "$root"/server/scripts/certs/*.sh "$root"/server/scripts/deploy/*.sh; do
  sh -n "$script"
  if [[ ! -x "$script" ]]; then
    echo "bundle script is not executable: ${script#"$root/"}" >&2
    exit 1
  fi
done

"$root/leamout" --help >/dev/null

printf 'verified %s\n' "$archive"
