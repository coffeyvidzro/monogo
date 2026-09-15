#!/bin/sh
set -eu

umask 077

if [ "$(id -u)" -ne 0 ]; then
  echo "issue-certificates.sh must run as root" >&2
  exit 1
fi

if [ "$#" -ne 2 ]; then
  echo "usage: issue-certificates.sh <base-domain> <acme-webroot>" >&2
  exit 2
fi

domain=$1
webroot=$2
hook=/etc/letsencrypt/renewal-hooks/deploy/leamout

if ! command -v certbot >/dev/null 2>&1; then
  echo "certbot is required" >&2
  exit 1
fi

if [ ! -x "$hook" ]; then
  echo "Certbot deploy hook is not installed at $hook" >&2
  exit 1
fi

if [ ! -d "$webroot" ]; then
  echo "ACME webroot does not exist: $webroot" >&2
  exit 1
fi

certbot certonly \
  --non-interactive \
  --agree-tos \
  --register-unsafely-without-email \
  --cert-name leamout-sip-turn \
  --webroot \
  --webroot-path "$webroot" \
  -d "sip.$domain" \
  -d "turn.$domain" \
  --deploy-hook "$hook"
