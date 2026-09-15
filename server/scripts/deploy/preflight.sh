#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=lib.sh
. "$script_dir/lib.sh"

require_root
require_command docker
require_command stat
require_command readlink

docker compose version >/dev/null 2>&1 || fail "Docker Compose plugin is required"
docker info >/dev/null 2>&1 || fail "Docker daemon is unavailable"

require_file "$LEAMOUT_ENV"
require_owner "$LEAMOUT_ENV" 0
require_group "$LEAMOUT_ENV" 0
require_mode "$LEAMOUT_ENV" 600

require_current_release
require_file "$LEAMOUT_COMPOSE"
require_file "$LEAMOUT_CURRENT/VERSION"

require_file "$LEAMOUT_FULLCHAIN"
require_owner "$LEAMOUT_FULLCHAIN" 0
require_group "$LEAMOUT_FULLCHAIN" 0
require_mode "$LEAMOUT_FULLCHAIN" 644

require_file "$LEAMOUT_PRIVKEY"
require_owner "$LEAMOUT_PRIVKEY" 0
require_group "$LEAMOUT_PRIVKEY" 65534
require_mode "$LEAMOUT_PRIVKEY" 640

compose config --quiet

for service in $LEAMOUT_RUNTIME_SERVICES; do
  service_exists "$service" || fail "required Compose service is missing: $service"
done

printf '%s\n' "Leamout deployment preflight passed"
