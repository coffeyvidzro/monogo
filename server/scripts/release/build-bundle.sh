#!/usr/bin/env bash
set -euo pipefail

version=${1:-}
os=${2:-linux}
arch=${3:-amd64}

if [[ -z "$version" ]]; then
  echo "usage: build-bundle.sh <version> [os] [arch]" >&2
  exit 2
fi

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$repo_root"

bundle="leamout_${version}_${os}_${arch}"
root="dist/${bundle}"

rm -rf "$root"
mkdir -p \
  "$root/deploy/docker" \
  "$root/server" \
  "$root/containers/nats" \
  "$root/containers/coturn" \
  "$root/server/scripts/certs" \
  "$root/server/scripts/deploy"

(
  cd server
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
    -trimpath \
    -ldflags="-s -w" \
    -o "../$root/leamout" \
    ./cmd/leamout
)

cp deploy/docker/compose.yaml "$root/deploy/docker/compose.yaml"
cp deploy/docker/Caddyfile "$root/deploy/docker/Caddyfile"
cp containers/nats/nats-server.conf "$root/containers/nats/nats-server.conf"
cp containers/coturn/turnserver.conf "$root/containers/coturn/turnserver.conf"
cp deploy/docker/SELF_HOSTED.md "$root/README.md"
cp server/scripts/certs/*.sh "$root/server/scripts/certs/"
cp server/scripts/deploy/*.sh "$root/server/scripts/deploy/"
chmod 0755 "$root/server/scripts/certs/"*.sh
chmod 0755 "$root/server/scripts/deploy/"*.sh
printf '%s\n' "$version" > "$root/VERSION"

tar -C dist -czf "dist/${bundle}.tar.gz" "$bundle"
(
  cd dist
  sha256sum "${bundle}.tar.gz" > checksums.txt
)

printf '%s\n' "dist/${bundle}.tar.gz"
