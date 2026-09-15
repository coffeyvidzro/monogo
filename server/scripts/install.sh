#!/bin/sh
set -eu

umask 077

repo="coffeyvidzro/monogo"
base_url="https://github.com/${repo}/releases/download"
install_bin="/usr/local/bin/leamout"
minisign_public_key='RWQsuFIP3yDqo+6K''v2NIBpY8W5S+FRMY''5NIvLzRbK6ohsOu6qqzsJXzR'

fail() {
  echo "leamout install: $*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"
command -v sha256sum >/dev/null 2>&1 || fail "sha256sum is required"

if ! command -v minisign >/dev/null 2>&1; then
  command -v apt-get >/dev/null 2>&1 || fail "minisign is required; automatic installation currently requires apt-get"
  DEBIAN_FRONTEND=noninteractive apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y minisign
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
[ "$os" = "linux" ] || fail "only Linux is supported"

case "$(uname -m)" in
  x86_64|amd64)
    arch="amd64"
    ;;
  aarch64|arm64)
    fail "linux/arm64 is not published yet"
    ;;
  *)
    fail "unsupported architecture: $(uname -m)"
    ;;
esac

version=${LEAMOUT_VERSION:-}
if [ -z "$version" ]; then
  version=$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
    | awk -F '"' '/"tag_name"[[:space:]]*:/ { print $4; exit }')
  [ -n "$version" ] || fail "could not resolve the latest Leamout release"
fi

case "$version" in
  v*) tag="$version"; version=${version#v} ;;
  *)  tag="v${version}" ;;
esac

bundle="leamout_${version}_${os}_${arch}"
archive="${bundle}.tar.gz"
release_url="${base_url}/${tag}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

printf 'Downloading Leamout %s for %s/%s...\n' "$version" "$os" "$arch"
curl -fL "$release_url/$archive" -o "$tmp/$archive"
curl -fL "$release_url/checksums.txt" -o "$tmp/checksums.txt"
curl -fL "$release_url/checksums.txt.minisig" -o "$tmp/checksums.txt.minisig"

minisign -V \
  -m "$tmp/checksums.txt" \
  -x "$tmp/checksums.txt.minisig" \
  -P "$minisign_public_key" \
  -q || fail "release checksum signature verification failed"

expected=$(awk -v file="$archive" '$2 == file { print $1; exit }' "$tmp/checksums.txt")
[ -n "$expected" ] || fail "$archive is missing from checksums.txt"
printf '%s  %s\n' "$expected" "$tmp/$archive" | sha256sum -c -

tar -xzf "$tmp/$archive" -C "$tmp"
root="$tmp/$bundle"
[ -x "$root/leamout" ] || fail "release archive does not contain the Leamout installer"

install -m 0755 "$root/leamout" "$install_bin"

printf 'Installed Leamout CLI to %s\n' "$install_bin"
printf 'Starting self-hosted installer...\n\n'

if [ -r /dev/tty ]; then
  LEAMOUT_BUNDLE_DIR="$root" "$install_bin" install </dev/tty
else
  fail "interactive terminal is required to configure the deployment"
fi
