#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

export DOMAIN="${DOMAIN:-localhost}"
export PUBLIC_IP="${PUBLIC_IP:-127.0.0.1}"
export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-acceptance-postgres-password}"
export MINIO_ROOT_USER="${MINIO_ROOT_USER:-acceptance-root}"
export MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-acceptance-root-password}"
export MINIO_APP_ACCESS_KEY="${MINIO_APP_ACCESS_KEY:-acceptance-app}"
export MINIO_APP_SECRET_KEY="${MINIO_APP_SECRET_KEY:-acceptance-app-password}"
CERT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/leamout-webrtc-v1.XXXXXX")

export WEBRTC_V1_CERT_DIR="$CERT_DIR"
export FREESWITCH_ESL_PASSWORD="${FREESWITCH_ESL_PASSWORD:-webrtc-v1-esl-secret}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA}"
export TURN_REALM="${TURN_REALM:-webrtc-v1.local}"
export TURN_AUTH_SECRET="${TURN_AUTH_SECRET:-webrtc-v1-turn-secret-0123456789abcdef}"
export TURN_EXTERNAL_IP="${TURN_EXTERNAL_IP:-127.0.0.1}"
export TURN_PUBLIC_URLS="${TURN_PUBLIC_URLS:-turn:127.0.0.1:3478?transport=udp}"
export RTPENGINE_PUBLIC_IP="${RTPENGINE_PUBLIC_IP:-172.31.0.10}"
export LEAMOUT_API_URL="${LEAMOUT_API_URL:-http://127.0.0.1:8080}"
export LEAMOUT_API_TOKEN="${LEAMOUT_API_TOKEN:-lm_org_v1smoke0_v1smoke0abcdefghijklmnopqrstuvwx}"
export LEAMOUT_WSS_URL="${LEAMOUT_WSS_URL:-wss://127.0.0.1:5062}"

COMPOSE="docker compose -f deploy/compose.yaml -f tests/webrtc-v1/compose.yaml"

cleanup() {
    status=$?
    trap - EXIT INT TERM

    if [ "$status" -ne 0 ]; then
        printf '\n%s\n' "=== WebRTC v1 diagnostics: compose ps ==="
        (cd "$REPO_ROOT" && $COMPOSE ps -a) || true
        printf '\n%s\n' "=== WebRTC v1 diagnostics: service logs ==="
        (
            cd "$REPO_ROOT" &&
                $COMPOSE logs --no-color --tail=300 \
                    server opensips rtpengine freeswitch coturn postgres redis nats
        ) || true
    fi

    if [ "${WEBRTC_V1_KEEP_STACK:-0}" != "1" ]; then
        (cd "$REPO_ROOT" && $COMPOSE down -v --remove-orphans) >/dev/null 2>&1 || true
        rm -rf "$CERT_DIR"
    else
        printf '%s\n' "WebRTC v1 stack retained; certificates: $CERT_DIR"
    fi
    exit "$status"
}
trap cleanup EXIT INT TERM

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "openssl is required" >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "npm is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }

if [ -z "${WEBRTC_V1_TURN_MIN_PORT:-}" ] || [ -z "${WEBRTC_V1_TURN_MAX_PORT:-}" ]; then
    turn_range=$(python3 - <<'PY'
import random
import socket

width = 8
for _ in range(256):
    start = random.randint(61000, 64999 - width)
    sockets = []
    try:
        for port in range(start, start + width):
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.bind(("0.0.0.0", port))
            sockets.append(sock)
    except OSError:
        pass
    else:
        print(start, start + width - 1)
        break
    finally:
        for sock in sockets:
            sock.close()
else:
    raise SystemExit("could not find a free UDP relay range")
PY
    )
    set -- $turn_range
    export WEBRTC_V1_TURN_MIN_PORT="$1"
    export WEBRTC_V1_TURN_MAX_PORT="$2"
fi
printf '%s\n' "Using Coturn relay ports ${WEBRTC_V1_TURN_MIN_PORT}-${WEBRTC_V1_TURN_MAX_PORT}"

cat >"$CERT_DIR/opensips.ext" <<'EOF'
subjectAltName=DNS:webrtc-v1.local,DNS:opensips,IP:127.0.0.1
extendedKeyUsage=serverAuth
EOF

openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
    -keyout "$CERT_DIR/ca.key" \
    -out "$CERT_DIR/ca.crt" \
    -subj "/CN=Leamout WebRTC v1 Acceptance CA" >/dev/null 2>&1

openssl req -newkey rsa:2048 -nodes \
    -keyout "$CERT_DIR/opensips-privkey.pem" \
    -out "$CERT_DIR/opensips.csr" \
    -subj "/CN=webrtc-v1.local" >/dev/null 2>&1

openssl x509 -req \
    -in "$CERT_DIR/opensips.csr" \
    -CA "$CERT_DIR/ca.crt" \
    -CAkey "$CERT_DIR/ca.key" \
    -CAcreateserial -days 1 \
    -out "$CERT_DIR/opensips-fullchain.pem" \
    -extfile "$CERT_DIR/opensips.ext" >/dev/null 2>&1
cp "$CERT_DIR/ca.crt" "$CERT_DIR/opensips-carrier-ca.pem"
chmod 0644 "$CERT_DIR/ca.crt" "$CERT_DIR/opensips-fullchain.pem" "$CERT_DIR/opensips-carrier-ca.pem"
chmod 0600 "$CERT_DIR/ca.key" "$CERT_DIR/opensips-privkey.pem"

cd "$REPO_ROOT"
$COMPOSE config --quiet
$COMPOSE up -d --build postgres redis nats rtpengine freeswitch coturn

printf '%s\n' "Waiting for PostgreSQL..."
i=0
until $COMPOSE exec -T postgres pg_isready -U leamout -d leamout >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -lt 60 ] || { echo "PostgreSQL did not become ready" >&2; exit 1; }
    sleep 1
done

printf '%s\n' "Applying migrations and WebRTC fixture..."
$COMPOSE up --build migrate
$COMPOSE exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U leamout -d leamout \
    <tests/webrtc-v1/bootstrap.sql >/dev/null

printf '%s\n' "Starting OpenSIPS and API..."
$COMPOSE up -d --build opensips server

printf '%s\n' "Waiting for API readiness..."
ready=0
for _ in $(seq 1 90); do
    if curl --fail --silent --output /dev/null "$LEAMOUT_API_URL/readyz"; then
        ready=1
        break
    fi
    sleep 1
done
[ "$ready" -eq 1 ] || { echo "API did not become ready within 90 seconds" >&2; exit 1; }

printf '%s\n' "Running forced TURN Chromium call..."
npm test --prefix tests/webrtc-v1
