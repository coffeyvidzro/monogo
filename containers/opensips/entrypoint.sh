#!/bin/sh
set -eu

config=${OPENSIPS_CONFIG:-/etc/opensips/opensips.cfg}
advertised_address=${OPENSIPS_ADVERTISED_ADDRESS:?OPENSIPS_ADVERTISED_ADDRESS must be set}
: "${OPENSIPS_DATABASE_URL:?OPENSIPS_DATABASE_URL must be set}"

# OpenSIPS advertises the address of this deployment. It is routing
# infrastructure for customer-provided SIP trunks, not a Leamout-owned SIP
# hostname or managed carrier edge.
tmp=$(mktemp)
awk -v address="$advertised_address" '
  BEGIN { replaced = 0 }

  /^[[:space:]]*#?[[:space:]]*advertised_address[[:space:]]*=/ {
    if (!replaced) {
      print "advertised_address = \"" address "\""
      print "alias = udp:" address ":5060"
      print "alias = tcp:" address ":5060"
      print "alias = tls:" address ":5061"
      print "alias = wss:" address ":5062"
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
# into the image. Routing queries and digest authentication use the same
# deployment database.
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
