#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=lib.sh
. "$script_dir/lib.sh"

require_root
require_command docker
require_file "$LEAMOUT_ENV"
require_current_release
require_file "$LEAMOUT_COMPOSE"

compose config --quiet

for service in $LEAMOUT_RUNTIME_SERVICES; do
  service_exists "$service" || fail "required Compose service is missing: $service"
done

timeout=${LEAMOUT_VERIFY_TIMEOUT_SECONDS:-120}
case "$timeout" in
  ''|*[!0-9]*) fail "LEAMOUT_VERIFY_TIMEOUT_SECONDS must be a positive integer" ;;
esac
[ "$timeout" -gt 0 ] || fail "LEAMOUT_VERIFY_TIMEOUT_SECONDS must be greater than zero"

started=$(date +%s)

while :; do
  pending=""

  for service in $LEAMOUT_RUNTIME_SERVICES; do
    container=$(compose ps -q "$service")
    if [ -z "$container" ]; then
      pending="$pending $service(no-container)"
      continue
    fi

    running=$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)
    if [ "$running" != "true" ]; then
      pending="$pending $service(not-running)"
      continue
    fi
  done

  for service in $LEAMOUT_HEALTH_SERVICES; do
    container=$(compose ps -q "$service")
    [ -n "$container" ] || continue

    health=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container" 2>/dev/null || true)
    if [ "$health" != "healthy" ]; then
      pending="$pending $service($health)"
    fi
  done

  if [ -z "$pending" ]; then
    printf '%s\n' "Leamout deployment verified"
    exit 0
  fi

  now=$(date +%s)
  elapsed=$((now - started))
  if [ "$elapsed" -ge "$timeout" ]; then
    fail "deployment verification timed out after ${timeout}s:$pending"
  fi

  sleep 2
done
