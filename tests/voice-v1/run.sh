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
CERT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/leamout-voice-v1.XXXXXX")

export VOICE_V1_SUITE_DIR="$SCRIPT_DIR"
export VOICE_V1_CERT_DIR="$CERT_DIR"
export FREESWITCH_ESL_PASSWORD="${FREESWITCH_ESL_PASSWORD:-voice-v1-esl-secret}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA}"
export TURN_REALM="${TURN_REALM:-voice-v1.local}"
export TURN_AUTH_SECRET="${TURN_AUTH_SECRET:-voice-v1-turn-secret-0123456789abcdef}"
export TURN_EXTERNAL_IP="${TURN_EXTERNAL_IP:-127.0.0.1}"
export TURN_PUBLIC_URLS="${TURN_PUBLIC_URLS:-turn:127.0.0.1:3478}"
export RTPENGINE_PUBLIC_IP="${RTPENGINE_PUBLIC_IP:-172.31.0.10}"

COMPOSE="docker compose -f deploy/compose.yaml -f tests/voice-v1/compose.yaml"

cleanup() {
    status=$?
    trap - EXIT INT TERM

    if [ "$status" -ne 0 ]; then
        printf '\n%s\n' "=== Voice v1 diagnostics: compose ps ==="
        (cd "$REPO_ROOT" && $COMPOSE ps -a) || true

        printf '\n%s\n' "=== Voice v1 diagnostics: service logs ==="
        (
            cd "$REPO_ROOT" &&
                $COMPOSE logs --no-color --tail=300 \
                    server \
                    worker \
                    freeswitch \
                    opensips \
                    rtpengine \
                    postgres \
                    nats \
                    redis \
                    voice-v1-carrier \
                    voice-v1-webhook
        ) || true
    fi

    if [ "${VOICE_V1_KEEP_STACK:-0}" != "1" ]; then
        (cd "$REPO_ROOT" && $COMPOSE down -v --remove-orphans) >/dev/null 2>&1 || true
        rm -rf "$CERT_DIR"
    else
        printf '%s\n' "Voice v1 stack retained; certificates: $CERT_DIR"
    fi
    exit "$status"
}
trap cleanup EXIT INT TERM

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "openssl is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }

cat >"$CERT_DIR/webhook.ext" <<'EOF'
subjectAltName=DNS:voice-v1-webhook,IP:127.0.0.1
extendedKeyUsage=serverAuth
EOF

openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
    -keyout "$CERT_DIR/ca.key" \
    -out "$CERT_DIR/ca.crt" \
    -subj "/CN=Leamout Voice v1 Acceptance CA" >/dev/null 2>&1

openssl req -newkey rsa:2048 -nodes \
    -keyout "$CERT_DIR/webhook.key" \
    -out "$CERT_DIR/webhook.csr" \
    -subj "/CN=voice-v1-webhook" >/dev/null 2>&1

openssl x509 -req \
    -in "$CERT_DIR/webhook.csr" \
    -CA "$CERT_DIR/ca.crt" \
    -CAkey "$CERT_DIR/ca.key" \
    -CAcreateserial -days 1 \
    -out "$CERT_DIR/webhook.crt" \
    -extfile "$CERT_DIR/webhook.ext" >/dev/null 2>&1

openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
    -keyout "$CERT_DIR/opensips-privkey.pem" \
    -out "$CERT_DIR/opensips-fullchain.pem" \
    -subj '/CN=voice-v1.local' >/dev/null 2>&1
cp "$CERT_DIR/opensips-fullchain.pem" "$CERT_DIR/opensips-carrier-ca.pem"

chmod 0644 "$CERT_DIR/ca.crt" "$CERT_DIR/webhook.crt" "$CERT_DIR/opensips-fullchain.pem" "$CERT_DIR/opensips-carrier-ca.pem"
chmod 0600 "$CERT_DIR/webhook.key" "$CERT_DIR/opensips-privkey.pem"

cd "$REPO_ROOT"
$COMPOSE config --quiet
$COMPOSE up -d --build postgres redis nats rtpengine freeswitch voice-v1-carrier voice-v1-webhook

printf '%s\n' "Waiting for PostgreSQL..."
i=0
until $COMPOSE exec -T postgres pg_isready -U leamout -d leamout >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -lt 60 ] || { echo "PostgreSQL did not become ready" >&2; exit 1; }
    sleep 1
done

printf '%s\n' "Applying migrations..."
$COMPOSE up --build migrate
$COMPOSE exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U leamout -d leamout \
    <tests/voice-v1/bootstrap.sql >/dev/null

printf '%s\n' "Starting OpenSIPS after database bootstrap..."
$COMPOSE up -d --build opensips

printf '%s\n' "Waiting for OpenSIPS readiness..."
i=0
until $COMPOSE exec -T opensips /usr/local/bin/leamout-opensips-drain status >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -lt 60 ] || { echo "OpenSIPS did not become ready" >&2; $COMPOSE logs --no-color --tail=200 opensips >&2 || true; exit 1; }
    sleep 1
done

printf '%s\n' "Waiting for private FreeSWITCH ESL..."
i=0
until $COMPOSE exec -T freeswitch sh -c '
    fs_cli -H 127.0.0.1 -P 8021 \
        -p "$FREESWITCH_ESL_PASSWORD" \
        -x status >/dev/null 2>&1
'; do
    i=$((i + 1))
    [ "$i" -lt 60 ] || { echo "FreeSWITCH ESL did not become ready/authenticated" >&2; exit 1; }
    sleep 1
done

$COMPOSE up -d --build server worker

ready=0
for _ in $(seq 1 90); do
    if python3 - <<'PY'
import urllib.request
try:
    with urllib.request.urlopen('http://127.0.0.1:8080/readyz', timeout=2) as response:
        raise SystemExit(0 if response.status == 204 else 1)
except Exception:
    raise SystemExit(1)
PY
    then
        ready=1
        break
    fi
    sleep 1
done
[ "$ready" -eq 1 ] || { echo "API did not become ready within 90 seconds" >&2; exit 1; }

python3 tests/voice-v1/acceptance.py
