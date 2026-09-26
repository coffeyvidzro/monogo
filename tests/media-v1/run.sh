#!/bin/sh
set -eu

cd "$(dirname "$0")/../.."
compose='docker compose -f tests/media-v1/compose.yaml'

cleanup() {
  $compose down --volumes --remove-orphans
}
trap cleanup EXIT INT TERM

$compose up --build --wait
python3 tests/media-v1/acceptance.py
