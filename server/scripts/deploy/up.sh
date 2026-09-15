#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

"$script_dir/preflight.sh"

# Pull and converge only the active release. Migration ordering remains defined
# by Compose and the Go installer/upgrade orchestration.
. "$script_dir/lib.sh"
compose pull
compose up -d
