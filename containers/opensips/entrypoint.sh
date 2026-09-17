#!/bin/sh
set -eu

config=${OPENSIPS_CONFIG:-/etc/opensips/opensips.cfg}
sip_domain=${SIP_DOMAIN:-sip.leamout.com}
: "${OPENSIPS_DATABASE_URL:?OPENSIPS_DATABASE_URL must be set}"

# The public SIP hostname is deployment configuration and is provided to the
# Cloud image at runtime.
tmp=$(mktemp)
awk -v domain="$sip_domain" '
  BEGIN { replaced = 0 }

  /^[[:space:]]*#?[[:space:]]*advertised_address[[:space:]]*=/ {
    if (!replaced) {
      print "advertised_address = \"" domain "\""
      print "alias = udp:" domain ":5060"
      print "alias = tcp:" domain ":5060"
      print "alias = tls:" domain ":5061"
      print "alias = wss:" domain ":5062"
      replaced = 1
    }
    next
  }

  { print }

  END {
    if (!replaced) {
      exit 42
    }
  }
' "$config" > "$tmp" || {
  rc=$?
  rm -f "$tmp"
  if [ "$rc" -eq 42 ]; then
    echo "advertised_address anchor is missing from $config" >&2
  fi
  exit "$rc"
}
cat "$tmp" > "$config"
rm -f "$tmp"

# Keep database credentials in deployment configuration instead of baking them
# into the image. Both modules must use the same Cloud database.
tmp=$(mktemp)
awk -v url="$OPENSIPS_DATABASE_URL" '
  /^modparam\("sqlops", "db_url",/ {
    print "modparam(\"sqlops\", \"db_url\", \"" url "\")"
    next
  }
  /^modparam\("auth_db", "db_url",/ {
    print "modparam(\"auth_db\", \"db_url\", \"" url "\")"
    next
  }
  { print }
' "$config" > "$tmp"
cat "$tmp" > "$config"
rm -f "$tmp"

# Use the system trust store for outbound carrier TLS unless the deployment
# explicitly provides a carrier CA bundle.
if [ ! -f /etc/opensips/tls/carrier-ca.pem ]; then
  cp /etc/ssl/certs/ca-certificates.crt /etc/opensips/tls/carrier-ca.pem
fi

# Validate the effective runtime configuration before starting OpenSIPS.
opensips -C -f "$config"

exec "$@"
