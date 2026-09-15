#!/bin/sh
set -eu

LEAMOUT_ENV=/etc/leamout/leamout.env
LEAMOUT_CURRENT=/opt/leamout/current
LEAMOUT_COMPOSE=/opt/leamout/current/compose.yaml

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_root() {
  [ "$(id -u)" -eq 0 ] || fail "Leamout deployment operations must run as root"
}

require_file() {
  [ -f "$1" ] || fail "required file not found: $1"
}

require_symlink() {
  [ -L "$1" ] || fail "required symlink not found: $1"
}

compose() {
  docker compose \
    --env-file "$LEAMOUT_ENV" \
    -f "$LEAMOUT_COMPOSE" \
    "$@"
}
