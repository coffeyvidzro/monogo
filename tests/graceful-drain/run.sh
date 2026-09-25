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
# Synthetic credentials for disposable acceptance stacks; no provider calls
# are made by these suites. Never reuse these values for real deployments.
export DIDWW_API_KEY="${DIDWW_API_KEY:-acceptance-didww-not-live}"
export COMMPEAK_API_AUTHORIZATION="${COMMPEAK_API_AUTHORIZATION:-acceptance-commpeak-not-live}"
export STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-sk_test_acceptance_not_live}"
export STRIPE_WEBHOOK_SECRET="${STRIPE_WEBHOOK_SECRET:-whsec_acceptance_not_live}"
export PAYSTACK_SECRET_KEY="${PAYSTACK_SECRET_KEY:-sk_test_acceptance_not_live}"


export GRACEFUL_DRAIN_SUITE_DIR="$SCRIPT_DIR"
export FREESWITCH_ESL_PASSWORD="${FREESWITCH_ESL_PASSWORD:-graceful-drain-esl-secret}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA}"
export RTPENGINE_PUBLIC_IP="${RTPENGINE_PUBLIC_IP:-172.31.0.10}"
export TURN_AUTH_SECRET="${TURN_AUTH_SECRET:-graceful-drain-turn-secret-0123456789abcdef}"
export TURN_EXTERNAL_IP="${TURN_EXTERNAL_IP:-127.0.0.1}"
export TURN_PUBLIC_URLS="${TURN_PUBLIC_URLS:-turn:127.0.0.1:3478}"
export TURN_REALM="${TURN_REALM:-graceful-drain.local}"

COMPOSE="docker compose -f deploy/compose.yaml -f tests/graceful-drain/compose.yaml -f tests/acceptance-minio.yaml"
COMPOSE_CONFIG_TMP=""

fs_diag() {
    service=$1
    echo "--- $service active channels ---" >&2
    (cd "$REPO_ROOT" && $COMPOSE exec -T "$service" \
        fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" \
        -x 'show channels') >&2 || true
    echo "--- $service calls ---" >&2
    (cd "$REPO_ROOT" && $COMPOSE exec -T "$service" \
        fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" \
        -x 'show calls') >&2 || true
}

cleanup() {
    status=$?
    trap - EXIT INT TERM
    if [ -n "$COMPOSE_CONFIG_TMP" ]; then
        rm -f "$COMPOSE_CONFIG_TMP"
    fi
    if [ "$status" -ne 0 ]; then
        (cd "$REPO_ROOT" && $COMPOSE ps -a) || true
        fs_diag graceful-drain-carrier
        fs_diag freeswitch
        echo "--- OpenSIPS drain/dialog state ---" >&2
        (cd "$REPO_ROOT" && $COMPOSE exec -T opensips /usr/local/bin/leamout-opensips-drain status) >&2 || true
        (cd "$REPO_ROOT" && $COMPOSE exec -T opensips /usr/local/bin/leamout-opensips-drain dialogs) >&2 || true
        echo "--- RTPengine media sockets ---" >&2
        (cd "$REPO_ROOT" && $COMPOSE exec -T rtpengine sh -lc \
            "netstat -anu 2>/dev/null | awk 'NR > 2 { n=split(\$4,a,\":\"); p=a[n]+0; if (p >= 23000 && p <= 32768) print \$0 }'") >&2 || true
        (cd "$REPO_ROOT" && $COMPOSE exec -T graceful-drain-carrier fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" -x 'sofia status profile internal') || true
        (cd "$REPO_ROOT" && $COMPOSE exec -T freeswitch fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" -x 'sofia status profile internal') || true
        (cd "$REPO_ROOT" && $COMPOSE logs --no-color --tail=400 server worker opensips freeswitch rtpengine graceful-drain-carrier postgres migrate) || true
    fi
    if [ "${GRACEFUL_DRAIN_KEEP_STACK:-0}" != "1" ]; then
        (cd "$REPO_ROOT" && $COMPOSE down -v --remove-orphans) >/dev/null 2>&1 || true
    fi
    exit "$status"
}
trap cleanup EXIT INT TERM

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "openssl is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }

mkdir -p "$SCRIPT_DIR/certs"
openssl req \
    -x509 \
    -newkey rsa:2048 \
    -nodes \
    -keyout "$SCRIPT_DIR/certs/privkey.pem" \
    -out "$SCRIPT_DIR/certs/fullchain.pem" \
    -subj '/CN=graceful-drain.local' \
    -days 1 \
    >/dev/null 2>&1
cp "$SCRIPT_DIR/certs/fullchain.pem" "$SCRIPT_DIR/certs/carrier-ca.pem"

for fixture in fullchain.pem privkey.pem carrier-ca.pem; do
    test -s "$SCRIPT_DIR/certs/$fixture" || {
        echo "missing generated TLS fixture: $SCRIPT_DIR/certs/$fixture" >&2
        exit 1
    }
done

cd "$REPO_ROOT"
$COMPOSE config --quiet
COMPOSE_CONFIG_TMP=$(mktemp)
$COMPOSE config --format json > "$COMPOSE_CONFIG_TMP"
python3 - "$COMPOSE_CONFIG_TMP" "$SCRIPT_DIR/certs" <<'PY'
import json
import os
import sys

config_path, cert_dir = sys.argv[1:]
with open(config_path, encoding="utf-8") as handle:
    config = json.load(handle)

volumes = config["services"]["opensips"].get("volumes", [])
resolved = {
    item.get("target"): os.path.realpath(item.get("source", ""))
    for item in volumes
    if isinstance(item, dict)
}
expected = {
    "/etc/opensips/tls/fullchain.pem": os.path.realpath(
        os.path.join(cert_dir, "fullchain.pem")
    ),
    "/etc/opensips/tls/privkey.pem": os.path.realpath(
        os.path.join(cert_dir, "privkey.pem")
    ),
    "/etc/opensips/tls/carrier-ca.pem": os.path.realpath(
        os.path.join(cert_dir, "carrier-ca.pem")
    ),
}

missing = {
    target: {"expected": source, "resolved": resolved.get(target)}
    for target, source in expected.items()
    if resolved.get(target) != source
}
if missing:
    raise SystemExit(
        "resolved OpenSIPS TLS mounts do not use acceptance fixtures: "
        + json.dumps(missing, sort_keys=True)
    )
PY
rm -f "$COMPOSE_CONFIG_TMP"
COMPOSE_CONFIG_TMP=""

$COMPOSE up -d --build postgres redis nats rtpengine freeswitch graceful-drain-carrier
until $COMPOSE exec -T postgres pg_isready -U leamout -d leamout >/dev/null 2>&1; do
    sleep 1
done

# Only the migration service is managed by this invocation: PostgreSQL is
# already healthy, and the migration exit status must stop the suite on failure.
$COMPOSE up --build --no-deps --exit-code-from migrate migrate
$COMPOSE exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U leamout -d leamout \
    < tests/graceful-drain/bootstrap.sql \
    >/dev/null

$COMPOSE up -d --build opensips server worker

sip_ready=0
for _ in $(seq 1 60); do
    if $COMPOSE exec -T graceful-drain-carrier \
            fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" \
            -x 'sofia status profile internal' >/dev/null 2>&1 \
        && $COMPOSE exec -T freeswitch \
            fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" \
            -x 'sofia status profile internal' >/dev/null 2>&1 \
        && $COMPOSE exec -T opensips \
            /usr/local/bin/leamout-opensips-drain status >/dev/null 2>&1; then
        sip_ready=1
        break
    fi
    sleep 1
done

if [ "$sip_ready" -ne 1 ]; then
    echo "SIP services did not become ready within 60 seconds" >&2
    exit 1
fi

# Verify the effective Record-Route address before originating a call.
$COMPOSE exec -T opensips grep -Fx \
    'advertised_address = "opensips"' /etc/opensips/opensips.cfg >/dev/null || {
    echo "OpenSIPS must advertise the Docker-reachable hostname opensips" >&2
    exit 1
}

# Acceptance-only wire diagnostics. On failure these traces tell us whether a
# hangup leaves FreeSWITCH, reaches OpenSIPS, or bypasses the proxy entirely.
$COMPOSE exec -T graceful-drain-carrier \
    fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" \
    -x 'sofia global siptrace on' >/dev/null
$COMPOSE exec -T freeswitch \
    fs_cli -H 127.0.0.1 -P 8021 -p "$FREESWITCH_ESL_PASSWORD" \
    -x 'sofia global siptrace on' >/dev/null

ready=0
for _ in $(seq 1 90); do
    if python3 - <<'PY'
import urllib.request
try:
    with urllib.request.urlopen('http://127.0.0.1:8080/readyz', timeout=2) as response:
        raise SystemExit(0 if response.status == 200 else 1)
except Exception:
    raise SystemExit(1)
PY
    then
        ready=1
        break
    fi
    sleep 1
done

if [ "$ready" -ne 1 ]; then
    echo "API did not become ready within 90 seconds" >&2
    exit 1
fi

# API readiness only proves the server can reach FreeSWITCH. Inbound call
# persistence is driven by the worker's ESL event subscription, so do not
# originate the acceptance call until that subscription is active. Without
# this gate a fast runner can send the INVITE in the same second the worker
# subscribes and lose the initial CHANNEL_CREATE event.
worker_ready=0
for _ in $(seq 1 60); do
    if $COMPOSE exec -T worker \
        wget --spider -q http://127.0.0.1:8081/readyz; then
        worker_ready=1
        break
    fi
    sleep 1
done

if [ "$worker_ready" -ne 1 ]; then
    echo "worker did not subscribe to FreeSWITCH lifecycle events within 60 seconds" >&2
    exit 1
fi

python3 tests/graceful-drain/acceptance.py
