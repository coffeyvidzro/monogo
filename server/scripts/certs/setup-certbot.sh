#!/bin/sh
set -eu

umask 077

if [ "$(id -u)" -ne 0 ]; then
  echo "setup-certbot.sh must run as root" >&2
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
  echo "automatic Let's Encrypt setup currently requires an apt-based Linux host" >&2
  exit 1
fi

if ! command -v certbot >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y certbot
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemd is required for automatic Certbot renewal" >&2
  exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
hook_source="$script_dir/certbot-hook.sh"
hook_target=/etc/letsencrypt/renewal-hooks/deploy/leamout

if [ ! -f "$hook_source" ]; then
  echo "missing Certbot hook: $hook_source" >&2
  exit 1
fi

install -d -o root -g root -m 0755 "$(dirname "$hook_target")"
install -o root -g root -m 0700 "$hook_source" "$hook_target"

systemctl enable --now certbot.timer
