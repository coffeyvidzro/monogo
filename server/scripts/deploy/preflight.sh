#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=lib.sh
. "$script_dir/lib.sh"

require_root
command -v docker >/dev/null 2>&1 || fail "Docker is required"
docker compose version >/dev/null 2>&1 || fail "Docker Compose plugin is required"
docker info >/dev/null 2>&1 || fail "Docker daemon is unavailable"
require_file "$LEAMOUT_ENV"
require_symlink "$LEAMOUT_CURRENT"
require_file "$LEAMOUT_COMPOSE"
require_file /etc/leamout/certs/fullchain.pem
require_file /etc/leamout/certs/privkey.pem

compose config --quiet
