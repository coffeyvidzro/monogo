#!/bin/sh
set -eu

config=${OPENSIPS_CONFIG:-/etc/opensips/opensips.cfg}
: "${OPENSIPS_DATABASE_URL:?OPENSIPS_DATABASE_URL must be set}"

# Keep database credentials in deployment configuration instead of baking them
# into the image. Routing queries and digest authentication use the same
# deployment database.
#
# auth_db reads the Leamout-owned opensips_inbound_trunk_credentials view rather
# than an OpenSIPS-managed subscriber table, so there is no OpenSIPS `version`
# metadata row to check for it.
auth_db_has_skip=0
if grep -Fq 'modparam("auth_db", "skip_version_check",' "$config"; then
  auth_db_has_skip=1
fi

tmp=$(mktemp)
awk -v url="$OPENSIPS_DATABASE_URL" -v has_skip="$auth_db_has_skip" '
  /^modparam\("sqlops", "db_url",/ {
    print "modparam(\"sqlops\", \"db_url\", \"" url "\")"
    next
  }
  /^modparam\("auth_db", "db_url",/ {
    print "modparam(\"auth_db\", \"db_url\", \"" url "\")"
    if (!has_skip) {
      print "modparam(\"auth_db\", \"skip_version_check\", 1)"
    }
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
