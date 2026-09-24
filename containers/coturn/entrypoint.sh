#!/bin/sh
set -eu

# Mounted private keys may be root-readable only; copy them into a private
# runtime directory before dropping privileges. Never modify host mounts.
: "${TURN_REALM:?TURN_REALM is required}"
: "${TURN_EXTERNAL_IP:?TURN_EXTERNAL_IP is required}"
: "${TURN_MIN_PORT:=49152}"
: "${TURN_MAX_PORT:=65535}"
: "${TURN_TLS_CERT_FILE:=/etc/coturn/tls/fullchain.pem}"
: "${TURN_TLS_KEY_FILE:=/etc/coturn/tls/privkey.pem}"

if [ -n "${TURN_AUTH_SECRET_FILE:-}" ]; then
    [ -r "$TURN_AUTH_SECRET_FILE" ] || {
        echo "TURN_AUTH_SECRET_FILE is not readable" >&2
        exit 1
    }
    turn_secret=$(cat "$TURN_AUTH_SECRET_FILE")
else
    turn_secret=${TURN_AUTH_SECRET:-}
fi
[ -n "$turn_secret" ] || {
    echo "TURN_AUTH_SECRET or TURN_AUTH_SECRET_FILE is required" >&2
    exit 1
}

# All values are written to a Coturn configuration file. Reject separators
# and newline/control characters rather than allowing configuration injection.
case "$TURN_REALM" in
    *[!A-Za-z0-9.-]* | .* | *.) echo "invalid TURN_REALM" >&2; exit 1 ;;
esac
case "$TURN_EXTERNAL_IP" in
    *[!A-Fa-f0-9.:/]* | */*/*) echo "invalid TURN_EXTERNAL_IP" >&2; exit 1 ;;
esac
case "$turn_secret" in
    *[!A-Za-z0-9_+=./:-]*) echo "invalid TURN authentication secret" >&2; exit 1 ;;
esac
case "$TURN_MIN_PORT:$TURN_MAX_PORT" in
    *[!0-9:]* | :* | *:) echo "invalid TURN relay port range" >&2; exit 1 ;;
esac
[ "$TURN_MIN_PORT" -ge 1 ] && [ "$TURN_MAX_PORT" -le 65535 ] &&
    [ "$TURN_MIN_PORT" -le "$TURN_MAX_PORT" ] || {
        echo "invalid TURN relay port range" >&2
        exit 1
    }
[ -r "$TURN_TLS_CERT_FILE" ] && [ -r "$TURN_TLS_KEY_FILE" ] || {
    echo "TURN TLS certificate or private key is not readable" >&2
    exit 1
}

umask 077
install -d -o coturn -g coturn -m 0700 /run/leamout-coturn
cp "$TURN_TLS_CERT_FILE" /run/leamout-coturn/fullchain.pem
cp "$TURN_TLS_KEY_FILE" /run/leamout-coturn/privkey.pem
cp /etc/coturn/turnserver.conf /run/leamout-coturn/turnserver.conf

{
    printf 'realm=%s\n' "$TURN_REALM"
    printf 'static-auth-secret=%s\n' "$turn_secret"
    printf 'external-ip=%s\n' "$TURN_EXTERNAL_IP"
    printf 'min-port=%s\n' "$TURN_MIN_PORT"
    printf 'max-port=%s\n' "$TURN_MAX_PORT"
    printf 'cert=/run/leamout-coturn/fullchain.pem\n'
    printf 'pkey=/run/leamout-coturn/privkey.pem\n'
} >> /run/leamout-coturn/turnserver.conf
unset turn_secret

chown coturn:coturn /run/leamout-coturn/fullchain.pem \
    /run/leamout-coturn/privkey.pem /run/leamout-coturn/turnserver.conf
chmod 0600 /run/leamout-coturn/fullchain.pem \
    /run/leamout-coturn/privkey.pem /run/leamout-coturn/turnserver.conf

exec gosu coturn:coturn /usr/bin/turnserver \
    -c /run/leamout-coturn/turnserver.conf
