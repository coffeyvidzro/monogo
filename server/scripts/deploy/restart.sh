#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=lib.sh
. "$script_dir/lib.sh"

require_root
require_file "$LEAMOUT_ENV"
require_symlink "$LEAMOUT_CURRENT"
require_file "$LEAMOUT_COMPOSE"

if [ "$#" -eq 0 ]; then
  compose restart
else
  compose restart "$@"
fi
