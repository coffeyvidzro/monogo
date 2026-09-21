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
# Disposable acceptance-only storage credentials; do not use root credentials in production.
export MINIO_APP_ACCESS_KEY="$MINIO_ROOT_USER"
export MINIO_APP_SECRET_KEY="$MINIO_ROOT_PASSWORD"
CERT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/leamout-cloud-managed.XXXXXX")
export CLOUD_MANAGED_SUITE_DIR="$SCRIPT_DIR"
export CLOUD_MANAGED_CERT_DIR="$CERT_DIR"
export FREESWITCH_ESL_PASSWORD="${FREESWITCH_ESL_PASSWORD:-cloud-managed-esl-secret}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA}"
export TURN_AUTH_SECRET="${TURN_AUTH_SECRET:-cloud-managed-turn-secret-0123456789}"
export TURN_PUBLIC_URLS="${TURN_PUBLIC_URLS:-turn:127.0.0.1:3478}"
export TURN_REALM="${TURN_REALM:-cloud-managed.local}"
export TURN_EXTERNAL_IP="${TURN_EXTERNAL_IP:-127.0.0.1}"
export RTPENGINE_PUBLIC_IP="${RTPENGINE_PUBLIC_IP:-172.31.0.10}"
COMPOSE="docker compose -f deploy/compose.yaml -f tests/cloud-managed/compose.yaml"

freeswitch_sip_ready() {
    $COMPOSE exec -T freeswitch sh -c '
        output=$(fs_cli -H 127.0.0.1 -P 8021 \
            -p "$FREESWITCH_ESL_PASSWORD" \
            -x "sofia status profile internal" 2>&1) || exit 1
        case "$output" in
            *BIND-URL*":5060"*) exit 0 ;;
            *) exit 1 ;;
        esac
    '
}

show_freeswitch_sip_status() {
    $COMPOSE exec -T freeswitch sh -c '
        fs_cli -H 127.0.0.1 -P 8021 \
            -p "$FREESWITCH_ESL_PASSWORD" \
            -x "sofia status profile internal" 2>&1
    ' || true
}

cleanup() {
    status=$?; trap - EXIT INT TERM
    if [ "$status" -ne 0 ]; then
        (cd "$REPO_ROOT" && $COMPOSE ps -a) || true
        (cd "$REPO_ROOT" && $COMPOSE logs --no-color --tail=400 server worker opensips freeswitch cloud-managed-provider cloud-managed-wholesale postgres redis migrate) || true
    fi
    if [ "${CLOUD_MANAGED_KEEP_STACK:-0}" != "1" ]; then
        (cd "$REPO_ROOT" && $COMPOSE down -v --remove-orphans) >/dev/null 2>&1 || true
        rm -rf "$CERT_DIR"
    fi
    exit "$status"
}
trap cleanup EXIT INT TERM

openssl req -x509 -newkey rsa:2048 -nodes -keyout "$CERT_DIR/privkey.pem" \
    -out "$CERT_DIR/fullchain.pem" -subj '/CN=cloud-managed.local' -days 1 >/dev/null 2>&1
cp "$CERT_DIR/fullchain.pem" "$CERT_DIR/carrier-ca.pem"

cd "$REPO_ROOT"
$COMPOSE config --quiet
$COMPOSE up -d --build postgres redis nats rtpengine freeswitch cloud-managed-provider cloud-managed-wholesale
until $COMPOSE exec -T postgres pg_isready -U leamout -d leamout >/dev/null 2>&1; do sleep 1; done
# Only the migration service is managed by this invocation: PostgreSQL is
# already healthy, and the migration exit status must stop the suite on failure.
$COMPOSE up --build --no-deps --exit-code-from migrate migrate
$COMPOSE exec -T postgres psql -v ON_ERROR_STOP=1 -U leamout -d leamout <tests/cloud-managed/bootstrap.sql >/dev/null
$COMPOSE up -d --build server worker opensips

ready=0
for _ in $(seq 1 90); do
    if python3 -c 'import urllib.request; assert urllib.request.urlopen("http://127.0.0.1:8080/readyz", timeout=2).status == 204; assert urllib.request.urlopen("http://127.0.0.1:18090/__state", timeout=2).status == 200; assert urllib.request.urlopen("http://127.0.0.1:18091", timeout=2).status == 200' >/dev/null 2>&1 \
        && freeswitch_sip_ready \
        && $COMPOSE exec -T opensips /usr/local/bin/leamout-opensips-drain status >/dev/null 2>&1; then
        ready=1
        break
    fi
    sleep 1
done
if [ "$ready" -ne 1 ]; then
    echo "cloud-managed acceptance topology did not become ready" >&2
    show_freeswitch_sip_status >&2
    exit 1
fi

$COMPOSE exec -T freeswitch fs_cli -H 127.0.0.1 -P 8021 \
    -p "$FREESWITCH_ESL_PASSWORD" -x "console loglevel debug" >/dev/null
$COMPOSE exec -T freeswitch fs_cli -H 127.0.0.1 -P 8021 \
    -p "$FREESWITCH_ESL_PASSWORD" -x "sofia global siptrace on" >/dev/null
python3 tests/cloud-managed/acceptance.py
