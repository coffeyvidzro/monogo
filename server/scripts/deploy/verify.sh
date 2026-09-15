#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=lib.sh
. "$script_dir/lib.sh"

require_root
require_file "$LEAMOUT_ENV"
require_symlink "$LEAMOUT_CURRENT"
require_file "$LEAMOUT_COMPOSE"

compose config --quiet

services="postgres redis nats server worker caddy freeswitch rtpengine opensips coturn"
for service in $services; do
  container=$(compose ps -q "$service")
  [ -n "$container" ] || fail "service has no container: $service"

  running=$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)
  [ "$running" = "true" ] || fail "service is not running: $service"
done

for service in postgres redis; do
  container=$(compose ps -q "$service")
  health=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container" 2>/dev/null || true)
  [ "$health" = "healthy" ] || fail "service is not healthy: $service ($health)"
done

printf '%s\n' "Leamout deployment verified"
