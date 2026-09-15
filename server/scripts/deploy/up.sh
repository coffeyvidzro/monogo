#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=lib.sh
. "$script_dir/lib.sh"

require_root
require_file "$LEAMOUT_ENV"
require_symlink "$LEAMOUT_CURRENT"
require_file "$LEAMOUT_COMPOSE"

compose pull
compose up -d
