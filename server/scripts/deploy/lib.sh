#!/bin/sh
set -eu

LEAMOUT_ENV=/etc/leamout/leamout.env
LEAMOUT_CERT_DIR=/etc/leamout/certs
LEAMOUT_FULLCHAIN=$LEAMOUT_CERT_DIR/fullchain.pem
LEAMOUT_PRIVKEY=$LEAMOUT_CERT_DIR/privkey.pem
LEAMOUT_RELEASES=/opt/leamout/releases
LEAMOUT_CURRENT=/opt/leamout/current
LEAMOUT_COMPOSE=$LEAMOUT_CURRENT/compose.yaml

LEAMOUT_RUNTIME_SERVICES="postgres redis nats server worker caddy freeswitch rtpengine opensips coturn"
LEAMOUT_HEALTH_SERVICES="postgres redis"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_root() {
  [ "$(id -u)" -eq 0 ] || fail "Leamout deployment operations must run as root"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_file() {
  [ -f "$1" ] || fail "required file not found: $1"
}

require_directory() {
  [ -d "$1" ] || fail "required directory not found: $1"
}

require_symlink() {
  [ -L "$1" ] || fail "required symlink not found: $1"
}

require_owner() {
  path=$1
  expected_uid=$2
  actual_uid=$(stat -c '%u' "$path") || fail "could not inspect owner: $path"
  [ "$actual_uid" = "$expected_uid" ] || fail "unexpected owner for $path: uid $actual_uid, want $expected_uid"
}

require_group() {
  path=$1
  expected_gid=$2
  actual_gid=$(stat -c '%g' "$path") || fail "could not inspect group: $path"
  [ "$actual_gid" = "$expected_gid" ] || fail "unexpected group for $path: gid $actual_gid, want $expected_gid"
}

require_mode() {
  path=$1
  expected_mode=$2
  actual_mode=$(stat -c '%a' "$path") || fail "could not inspect mode: $path"
  [ "$actual_mode" = "$expected_mode" ] || fail "unexpected mode for $path: $actual_mode, want $expected_mode"
}

require_current_release() {
  require_symlink "$LEAMOUT_CURRENT"
  target=$(readlink -f "$LEAMOUT_CURRENT") || fail "could not resolve active release: $LEAMOUT_CURRENT"
  require_directory "$target"

  case "$target" in
    "$LEAMOUT_RELEASES"/*) ;;
    *) fail "active release points outside $LEAMOUT_RELEASES: $target" ;;
  esac
}

compose() {
  (
    cd "$LEAMOUT_CURRENT"
    docker compose \
      --env-file "$LEAMOUT_ENV" \
      -f "$LEAMOUT_COMPOSE" \
      "$@"
  )
}

service_exists() {
  service=$1
  compose config --services | grep -Fx "$service" >/dev/null 2>&1
}

require_runtime_service() {
  service=$1
  for allowed in $LEAMOUT_RUNTIME_SERVICES; do
    [ "$service" = "$allowed" ] && return 0
  done
  fail "unsupported runtime service: $service"
}
