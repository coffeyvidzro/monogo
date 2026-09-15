#!/bin/sh
set -eu

config=${OPENSIPS_CONFIG:-/etc/opensips/opensips.cfg}
advertised_address=${OPENSIPS_ADVERTISED_ADDRESS:-}
database_password=${OPENSIPS_DATABASE_PASSWORD:-}

[ -n "$database_password" ] || {
  echo "OPENSIPS_DATABASE_PASSWORD is required" >&2
  exit 1
}
case "$database_password" in
  *[!A-Za-z0-9]*)
    echo "OPENSIPS_DATABASE_PASSWORD must be alphanumeric" >&2
    exit 1
    ;;
esac

# The source configuration keeps the contributor credential solely so the
# image can be syntax-checked during build. Replace it before OpenSIPS starts;
# production always supplies a deployment-owned generated credential.
tmp=$(mktemp)
sed "s#postgres://leamout:leamout@postgres:5432/leamout#postgres://leamout:${database_password}@postgres:5432/leamout#g" "$config" > "$tmp"
cat "$tmp" > "$config"
rm -f "$tmp"

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

if [ -n "$advertised_address" ]; then
  case "$advertised_address" in
    *[!A-Za-z0-9._-]*)
      echo "OPENSIPS_ADVERTISED_ADDRESS must be a hostname or IPv4 address" >&2
      exit 1
      ;;
  esac

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
fi

# Always validate the fully materialized runtime configuration before the
# daemon starts so credential or advertised-address rendering fails closed.
opensips -C -f "$config"

exec "$@"
