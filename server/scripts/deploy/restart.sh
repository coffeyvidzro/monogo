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

if [ "$#" -eq 0 ]; then
  compose restart $LEAMOUT_RUNTIME_SERVICES
  exit 0
fi

for service in "$@"; do
  require_runtime_service "$service"
  service_exists "$service" || fail "Compose service is missing: $service"
done

compose restart "$@"
