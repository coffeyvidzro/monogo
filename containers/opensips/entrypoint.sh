#!/bin/sh
set -eu

config=${OPENSIPS_CONFIG:-/etc/opensips/opensips.cfg}

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

# Leamout currently uses one fixed public SIP hostname.
tmp=$(mktemp)
awk '
  BEGIN { replaced = 0 }

  /^[[:space:]]*#?[[:space:]]*advertised_address[[:space:]]*=/ {
    if (!replaced) {
      print "advertised_address = \"sip.leamout.com\""
      print "alias = udp:sip.leamout.com:5060"
      print "alias = tcp:sip.leamout.com:5060"
      print "alias = tls:sip.leamout.com:5061"
      print "alias = wss:sip.leamout.com:5062"
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

# Validate the effective runtime configuration before starting OpenSIPS.
opensips -C -f "$config"

exec "$@"
