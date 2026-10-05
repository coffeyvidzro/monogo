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

advertised=${OPENSIPS_ADVERTISED_ADDRESS:-}
case "$advertised" in
  *[!a-zA-Z0-9.-]*) echo "Invalid SIP advertised address" >&2; exit 1 ;;
esac
external=$(ip -4 route get 1.1.1.1 | awk '{ for (i = 1; i <= NF; i++) if ($i == "src") { print $(i+1); exit } }')
tmp=$(mktemp)
# A wildcard bind accepts packets, but is not a usable SIP routing identity.
# Bind each global IPv4 interface explicitly. Only the default-route socket
# advertises the public deployment identity; preserve internal/custom sockets.
addresses=$(ip -4 -o address show scope global | awk '{ split($4, addr, "/"); print addr[1] }')
if [ -z "$addresses" ]; then
  echo "OpenSIPS requires a global IPv4 interface address" >&2
  exit 1
fi
awk -v url="$OPENSIPS_DATABASE_URL" -v has_skip="$auth_db_has_skip" -v addresses="$addresses" -v advertised="$advertised" -v external="$external" '
  /^socket = (udp|tcp|tls|wss):0\.0\.0\.0:[0-9]+$/ {
    split($3, socket, ":")
    count = split(addresses, ips, "\n")
    for (i = 1; i <= count; i++) {
      line = "socket = " socket[1] ":" ips[i] ":" socket[3]
      if (advertised != "" && ips[i] == external) {
        line = line " as " advertised ":" socket[3]
      }
      print line
    }
    next
  }
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
