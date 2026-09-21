#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../../.." && pwd)
CERT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/leamout-byoc-v1.XXXXXX")

export FREESWITCH_ESL_PASSWORD="${FREESWITCH_ESL_PASSWORD:-byoc-v1-esl-secret}"
# The acceptance test rotates the synthetic carrier credential at runtime.
# Stage all mounted carrier fixtures in the disposable directory so a local run
# never rewrites tracked files in tests/acceptance/byoc-v1.
export BYOC_V1_SUITE_DIR="$CERT_DIR"
export BYOC_V1_CERT_DIR="$CERT_DIR"
export CARRIER_CREDENTIAL_ENCRYPTION_KEY="${CARRIER_CREDENTIAL_ENCRYPTION_KEY:-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA}"
export TURN_REALM="${TURN_REALM:-byoc-v1.local}"
export TURN_AUTH_SECRET="${TURN_AUTH_SECRET:-byoc-v1-turn-secret-0123456789abcdef}"
export TURN_EXTERNAL_IP="${TURN_EXTERNAL_IP:-127.0.0.1}"
export TURN_PUBLIC_URLS="${TURN_PUBLIC_URLS:-turn:127.0.0.1:3478}"
export RTPENGINE_PUBLIC_IP="${RTPENGINE_PUBLIC_IP:-172.31.0.10}"
COMPOSE="docker compose -f deploy/self-hosted/compose.yaml -f tests/acceptance/byoc-v1/compose.yaml"

cleanup() {
    status=$?
    trap - EXIT INT TERM
    if [ "$status" -ne 0 ]; then
        (cd "$REPO_ROOT" && $COMPOSE ps -a) || true
        (cd "$REPO_ROOT" && $COMPOSE logs --no-color --tail=300 server worker opensips freeswitch rtpengine byoc-v1-carrier postgres) || true
    fi
    if [ "${BYOC_V1_KEEP_STACK:-0}" != "1" ]; then
        (cd "$REPO_ROOT" && $COMPOSE down -v --remove-orphans) >/dev/null 2>&1 || true
        rm -rf "$CERT_DIR"
    else
        printf '%s\n' "BYOC v1 stack retained; runtime fixtures and certificates: $CERT_DIR"
    fi
    exit "$status"
}
trap cleanup EXIT INT TERM

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "openssl is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }

cp "$SCRIPT_DIR/carrier-directory.xml" "$CERT_DIR/carrier-directory.xml"
cp "$SCRIPT_DIR/carrier-sip-profile.xml" "$CERT_DIR/carrier-sip-profile.xml"
cp "$SCRIPT_DIR/carrier-dialplan.xml" "$CERT_DIR/carrier-dialplan.xml"

openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$CERT_DIR/privkey.pem" \
    -out "$CERT_DIR/fullchain.pem" \
    -subj '/CN=byoc-v1.local' -days 1 >/dev/null 2>&1
cp "$CERT_DIR/fullchain.pem" "$CERT_DIR/carrier-ca.pem"

cd "$REPO_ROOT"
$COMPOSE config --quiet
$COMPOSE up -d --build postgres redis nats rtpengine freeswitch byoc-v1-carrier
until $COMPOSE exec -T postgres pg_isready -U leamout -d leamout >/dev/null 2>&1; do sleep 1; done
$COMPOSE up --build migrate
$COMPOSE exec -T postgres psql -v ON_ERROR_STOP=1 -U leamout -d leamout <tests/acceptance/byoc-v1/bootstrap.sql >/dev/null
$COMPOSE up -d --build opensips server worker

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

required_services="postgres redis nats rtpengine freeswitch opensips server worker byoc-v1-carrier"
stack_ready=0
for _ in $(seq 1 45); do
    running="$($COMPOSE ps --status running --services)"
    missing=0
    for service in $required_services; do
        if ! printf '%s\n' "$running" | grep -qx "$service"; then
            missing=1
            break
        fi
    done
    if [ "$missing" -eq 0 ]; then
        stack_ready=1
        break
    fi
    sleep 1
done
[ "$stack_ready" -eq 1 ] || {
    echo "BYOC stack did not become fully running within 45 seconds" >&2
    $COMPOSE ps -a >&2 || true
    exit 1
}

python3 tests/acceptance/byoc-v1/acceptance.py
