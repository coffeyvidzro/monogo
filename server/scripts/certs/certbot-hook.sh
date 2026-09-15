#!/bin/sh
set -eu

umask 077

[ -n "${RENEWED_LINEAGE:-}" ] || exit 0

install -d -o root -g root -m 0700 /etc/leamout/certs
install -o root -g root -m 0644 "$RENEWED_LINEAGE/fullchain.pem" /etc/leamout/certs/fullchain.pem.new
install -o root -g 65534 -m 0640 "$RENEWED_LINEAGE/privkey.pem" /etc/leamout/certs/privkey.pem.new
mv /etc/leamout/certs/fullchain.pem.new /etc/leamout/certs/fullchain.pem
mv /etc/leamout/certs/privkey.pem.new /etc/leamout/certs/privkey.pem

if [ -f /etc/leamout/leamout.env ] && [ -L /opt/leamout/current ]; then
  for service in opensips coturn; do
    if docker compose --env-file /etc/leamout/leamout.env -f /opt/leamout/current/compose.yaml ps -q "$service" 2>/dev/null | grep -q .; then
      docker compose --env-file /etc/leamout/leamout.env -f /opt/leamout/current/compose.yaml restart "$service"
    fi
  done
fi
