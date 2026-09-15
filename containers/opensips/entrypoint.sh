#!/bin/sh
set -eu

config=${OPENSIPS_CONFIG:-/etc/opensips/opensips.cfg}
sip_domain=${SIP_DOMAIN:-sip.leamout.com}

# Managed customer-facing SIP admission is intentionally disabled for now.
# Keep the rest of the SIP edge intact while removing the unfinished control-
# plane callback path between these markers from the effective runtime config.
if grep -q '# BEGIN MANAGED SIP ADMISSION' "$config"; then
  tmp=$(mktemp)
  awk '
    /# BEGIN MANAGED SIP ADMISSION/ { skip = 1; next }
    /# END MANAGED SIP ADMISSION/ { skip = 0; next }
    !skip { print }
  ' "$config" > "$tmp"
  cat "$tmp" > "$config"
  rm -f "$tmp"
fi

# The public SIP hostname is deployment configuration. Cloud and Self-Hosted
# use the same image and provide the hostname at runtime.
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

# Use the system trust store for outbound carrier TLS unless the deployment
# explicitly provides a carrier CA bundle.
if [ ! -f /etc/opensips/tls/carrier-ca.pem ]; then
  cp /etc/ssl/certs/ca-certificates.crt /etc/opensips/tls/carrier-ca.pem
fi

# Validate the effective runtime configuration before starting OpenSIPS.
opensips -C -f "$config"

exec "$@"
