#!/usr/bin/env python3
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid

API = os.getenv("BYOC_V1_API_BASE", "http://127.0.0.1:8080")
TOKEN = os.getenv("BYOC_V1_TOKEN", "lm_org_v1smoke0_v1smoke0abcdefghijklmnopqrstuvwx")
TOKEN_B = os.getenv(
    "BYOC_V1_TOKEN_B", "lm_org_v1smoke1_v1smoke1abcdefghijklmnopqrstuvwx"
)
ESL_PASSWORD = os.getenv("FREESWITCH_ESL_PASSWORD", "byoc-v1-esl-secret")
SUITE_DIR = os.getenv("BYOC_V1_SUITE_DIR", os.path.dirname(os.path.abspath(__file__)))
DID, CALLER = "+15551234567", "+15557654321"
COMPOSE = [
    "docker",
    "compose",
    "-f",
    "deploy/compose.yaml",
    "-f",
    "tests/byoc-v1/compose.yaml",
]
S, RESULTS = {}, []


class Failure(RuntimeError):
    pass


def run(args, check=True):
    p = subprocess.run(
        args, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT
    )
    if check and p.returncode:
        raise Failure(f"command failed: {' '.join(args)}\n{p.stdout}")
    return p.stdout.strip()


def compose(*args, check=True):
    return run(COMPOSE + list(args), check)


def fs(command):
    return compose(
        "exec",
        "-T",
        "byoc-v1-carrier",
        "fs_cli",
        "-H",
        "127.0.0.1",
        "-P",
        "8021",
        "-p",
        ESL_PASSWORD,
        "-x",
        command,
    )


def psql(sql):
    return compose(
        "exec", "-T", "postgres", "psql", "-U", "leamout", "-d", "leamout", "-Atc", sql
    )


def api(method, path, payload=None, expected=(200,), token=TOKEN):
    data = json.dumps(payload).encode() if payload is not None else None
    headers = {"Accept": "application/json", "Authorization": f"Bearer {token}"}
    if data:
        headers["Content-Type"] = "application/json"
    try:
        with urllib.request.urlopen(
            urllib.request.Request(
                API + path, data=data, headers=headers, method=method
            ),
            timeout=15,
        ) as r:
            status, raw = r.status, r.read()
    except urllib.error.HTTPError as e:
        status, raw = e.code, e.read()
    if status not in expected:
        raise Failure(
            f"{method} {path}: HTTP {status}, body={raw.decode(errors='replace')}"
        )
    if not raw:
        return None
    body = json.loads(raw)
    return body["data"] if body.get("success") is True else body


def wait(description, probe, timeout=20):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        try:
            value = probe()
            if value:
                return value
        except Exception:
            pass
        time.sleep(0.25)
    raise Failure(f"timed out waiting for {description}")


def check(name, fn):
    try:
        detail = fn() or ""
        RESULTS.append(True)
        print(f"PASS {len(RESULTS):02d} {name}: {detail}")
    except Exception as e:
        RESULTS.append(False)
        print(f"FAIL {len(RESULTS):02d} {name}: {e}")


def deploy():
    wait(
        "API readiness",
        lambda: urllib.request.urlopen(API + "/readyz", timeout=2).status == 200,
        45,
    )
    required = {
        "postgres",
        "redis",
        "nats",
        "rtpengine",
        "freeswitch",
        "opensips",
        "server",
        "worker",
        "byoc-v1-carrier",
    }
    missing = required - set(
        compose("ps", "--status", "running", "--services").splitlines()
    )
    if missing:
        raise Failure("missing services: " + ", ".join(sorted(missing)))
    return "signaling and control stack is running"


def provider():
    # Carrier providers are internal platform inventory, not a public API.
    # Migration 013 seeds this provider; verify the real row is usable.
    provider_id = psql(
        "SELECT id::text FROM carrier_providers "
        "WHERE slug='generic-sip' AND adapter='sip' AND status='active'"
    )
    if not provider_id:
        raise Failure("migration-seeded generic SIP provider is missing or inactive")
    S["provider"] = {"id": provider_id}
    return f"migration-seeded generic SIP provider {provider_id} is active"


def assert_digest_runtime(secret_name, expected_ha1):
    connection_id = S["connection"]["id"]
    stored = psql(
        f"SELECT auth_secret_ciphertext FROM carrier_connections WHERE id='{connection_id}'"
    )
    if not stored or secret_name in stored:
        raise Failure("credential was not encrypted")
    runtime = psql(
        f"SELECT username||':'||realm||':'||ha1_md5 FROM carrier_digest_credentials WHERE carrier_connection_id='{connection_id}' AND direction='outbound'"
    )
    expected = f"byoc-user:carrier.example:{expected_ha1}"
    if runtime != expected:
        raise Failure(f"realm-bound runtime HA1 mismatch: {runtime!r}")
    opensips_password = psql(
        f"SELECT password FROM opensips_outbound_carrier_credentials WHERE carrier_connection_id='{connection_id}'"
    )
    if opensips_password != "0x" + expected_ha1:
        raise Failure("OpenSIPS outbound HA1 view was not updated")


def connection_and_auth():
    item = api(
        "POST",
        "/v1/carrier-connections/",
        {
            "provider_id": S["provider"]["id"],
            "name": "BYOC synthetic carrier",
            "inbound_enabled": True,
        },
        (201,),
    )
    S["connection"] = item
    api(
        "PUT",
        f"/v1/carrier-connections/{item['id']}/outbound-auth",
        {
            "method": "digest",
            "username": "byoc-user",
            "realm": "carrier.example",
            "secret": "first-secret",
        },
        (204,),
    )
    # PUT acknowledges credential storage with no response body. Verify the
    # actual persisted carrier configuration via GET instead.
    updated = api("GET", f"/v1/carrier-connections/{item['id']}")
    if not updated["has_outbound_credentials"]:
        raise Failure("outbound credentials were not marked present")
    assert_digest_runtime("first-secret", "367072b7e49f70c083774e6dc9d06af8")
    return "first digest credential is encrypted and active in OpenSIPS runtime"


def trunk():
    item = api(
        "POST",
        "/v1/trunks/",
        {
            "type": "byoc",
            "carrier_connection_id": S["connection"]["id"],
            "name": "BYOC trunk",
            "direction": "bidirectional",
        },
        (201,),
    )
    S["trunk"] = item
    if item.get("type") != "byoc":
        raise Failure("trunk was not created as BYOC")
    if item.get("carrier_connection_id") != S["connection"]["id"]:
        raise Failure("trunk was not created on the carrier connection")
    endpoint = api(
        "POST",
        f"/v1/trunks/{item['id']}/endpoints",
        {
            "host": "byoc-v1-carrier",
            "port": 5060,
            "transport": "udp",
            "direction": "bidirectional",
            "priority": 10,
        },
        (201,),
    )
    S["endpoint"] = endpoint
    return f"trunk {item['id']} routes to endpoint {endpoint['id']}"


def number_and_app():
    number = api(
        "POST",
        "/v1/numbers/",
        {
            "number": DID,
            "country_code": "US",
            "carrier_connection_id": S["connection"]["id"],
            "voice_enabled": True,
        },
        (201,),
    )
    S["number"] = number
    if number.get("type") != "byoc":
        raise Failure("DID was not created as BYOC")
    if number.get("carrier_connection_id") != S["connection"]["id"]:
        raise Failure("DID ownership was not assigned at creation")

    caller = api(
        "POST",
        "/v1/numbers/",
        {
            "number": CALLER,
            "country_code": "US",
            "carrier_connection_id": S["connection"]["id"],
            "voice_enabled": True,
        },
        (201,),
    )
    S["caller_number"] = caller
    if caller.get("type") != "byoc":
        raise Failure("caller identity was not created as BYOC")
    if caller.get("carrier_connection_id") != S["connection"]["id"]:
        raise Failure("caller identity ownership was not assigned at creation")

    app = api(
        "POST",
        "/v1/voice-applications/",
        {"name": "BYOC ingress", "caller_id": CALLER},
        (201,),
    )
    S["app"] = app
    api(
        "POST",
        f"/v1/voice-applications/{app['id']}/bindings",
        {"phone_number_id": number["id"]},
        (201,),
    )
    return "DID and caller identity ownership plus application binding use public APIs"


def reject_cross_org_did_ownership():
    foreign_connection = api(
        "POST",
        "/v1/carrier-connections/",
        {
            "provider_id": S["provider"]["id"],
            "name": "BYOC foreign tenant carrier",
            "inbound_enabled": True,
        },
        (201,),
        TOKEN_B,
    )
    api("GET", f"/v1/numbers/{S['number']['id']}", expected=(404,), token=TOKEN_B)
    api(
        "PATCH",
        f"/v1/numbers/{S['number']['id']}/carrier-connection",
        {"carrier_connection_id": foreign_connection["id"]},
        (404,),
        TOKEN_B,
    )
    owned = api("GET", f"/v1/numbers/{S['number']['id']}")
    if owned.get("carrier_connection_id") != S["connection"]["id"]:
        raise Failure("cross-organization request changed DID ownership")
    return "tenant B cannot read or reassign tenant A DID ownership"


def outbound(label):
    call = api(
        "POST",
        "/v1/calls/",
        {
            "application_id": S["app"]["id"],
            "trunk_id": S["trunk"]["id"],
            "from_uri": CALLER,
            "to_uri": DID,
        },
        (201,),
    )
    S[f"outbound_{label}"] = call
    expected = {
        "carrier_connection_id": S["connection"]["id"],
        "trunk_id": S["trunk"]["id"],
        "trunk_endpoint_id": S["endpoint"]["id"],
    }
    for field, value in expected.items():
        if call.get(field) != value:
            raise Failure(f"{field}={call.get(field)!r}, want {value}")
    # Origination is queued. The real SIP Call-ID is persisted later by the
    # FreeSWITCH lifecycle consumer, not necessarily in the POST response.
    def persisted_with_sip_id():
        current = api("GET", f"/v1/calls/{call['id']}")
        return current if current.get("sip_call_id") else None

    persisted = wait(f"{label} outbound SIP Call-ID", persisted_with_sip_id)
    for field, value in expected.items():
        if persisted.get(field) != value:
            raise Failure(f"persisted {field}={persisted.get(field)!r}, want {value}")
    return f"{label} authenticated call {call['id']} persisted connection/trunk/endpoint attribution"


def first_authenticated_outbound():
    return outbound("first-secret")


def write_carrier_secret(secret):
    path = os.path.join(SUITE_DIR, "carrier-directory.xml")
    content = f'''<include>\n  <domain name="carrier.example">\n    <groups>\n      <group name="default">\n        <users>\n          <user id="byoc-user">\n            <params>\n              <param name="password" value="{secret}"/>\n            </params>\n          </user>\n        </users>\n      </group>\n    </groups>\n  </domain>\n</include>\n'''
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    output = fs("reloadxml")
    if "+OK" not in output:
        raise Failure("carrier did not reload rotated directory credential: " + output)


def rotate_and_authenticated_outbound():
    opensips_before = compose("ps", "-q", "opensips")
    if not opensips_before:
        raise Failure("OpenSIPS container id is unavailable before rotation")
    api(
        "PUT",
        f"/v1/carrier-connections/{S['connection']['id']}/outbound-auth",
        {
            "method": "digest",
            "username": "byoc-user",
            "realm": "carrier.example",
            "secret": "rotated-secret",
        },
        (204,),
    )
    updated = api("GET", f"/v1/carrier-connections/{S['connection']['id']}")
    if not updated["has_outbound_credentials"]:
        raise Failure("rotated outbound credentials were not marked present")
    assert_digest_runtime("rotated-secret", "9fcc44d55f26bac30b97201af8e5654d")
    write_carrier_secret("rotated-secret")
    opensips_after = compose("ps", "-q", "opensips")
    if opensips_after != opensips_before:
        raise Failure("OpenSIPS restarted during credential rotation")
    detail = outbound("rotated-secret")
    return detail + "; OpenSIPS container remained unchanged"


def rejected_before_allowlist():
    output = fs(
        f"originate {{origination_caller_id_number={CALLER}}}sofia/internal/{DID}@opensips:5060 &park()"
    )
    if "+OK" in output:
        raise Failure("untrusted carrier source was accepted")
    return "unknown carrier source is rejected"


def add_source_and_inbound():
    source = api(
        "POST",
        f"/v1/carrier-connections/{S['connection']['id']}/source-ips",
        {"cidr": "172.30.0.50/32"},
        (201,),
    )
    carrier_uuid = str(uuid.uuid4())
    try:
        before = {x["id"] for x in api("GET", "/v1/calls/?limit=100")["calls"]}
        output = fs(
            "bgapi originate "
            f"{{origination_uuid={carrier_uuid},origination_caller_id_number={CALLER}}}"
            f"sofia/internal/{DID}@opensips:5060 &park()"
        )
        if "+OK Job-UUID:" not in output:
            raise Failure("allowlisted carrier originate was not queued: " + output)
        call = wait(
            "inbound call",
            lambda: next(
                (
                    x
                    for x in api("GET", "/v1/calls/?limit=100")["calls"]
                    if x["id"] not in before and x["direction"] == "inbound"
                ),
                None,
            ),
        )
        S["inbound"] = call
    finally:
        fs(f"uuid_kill {carrier_uuid}")
        api(
            "DELETE",
            f"/v1/carrier-connections/{S['connection']['id']}/source-ips/{source['id']}",
            expected=(204,),
        )

    rejected = fs(
        f"originate {{origination_caller_id_number={CALLER}}}sofia/internal/{DID}@opensips:5060 &park()"
    )
    if "+OK" in rejected:
        raise Failure("removed carrier source remained authorized")
    return "source-IP addition and removal took effect without OpenSIPS restart"


def disable_rejects_routes():
    api(
        "PATCH",
        f"/v1/carrier-connections/{S['connection']['id']}",
        {"status": "disabled"},
    )
    api(
        "POST",
        "/v1/calls/",
        {"trunk_id": S["trunk"]["id"], "from_uri": CALLER, "to_uri": DID},
        (404,),
    )
    return "disabled carrier connection excludes outbound routes"


def restart_persistence():
    compose("restart", "opensips", "server")
    wait(
        "API recovery",
        lambda: urllib.request.urlopen(API + "/readyz", timeout=2).status == 200,
        45,
    )
    item = api("GET", f"/v1/carrier-connections/{S['connection']['id']}")
    if item["status"] != "disabled":
        raise Failure("carrier state changed after restart")
    return "carrier configuration survives OpenSIPS and API restart"


def main():
    tests = [
        ("Deploy BYOC stack", deploy),
        ("Discover generic SIP provider", provider),
        ("Activate first outbound digest credential", connection_and_auth),
        ("Provision trunk endpoint", trunk),
        ("Assign DID ownership", number_and_app),
        ("Reject cross-org DID ownership", reject_cross_org_did_ownership),
        ("Authenticate outbound with first secret", first_authenticated_outbound),
        (
            "Rotate digest auth without OpenSIPS restart",
            rotate_and_authenticated_outbound,
        ),
        ("Reject unknown source", rejected_before_allowlist),
        ("Apply source IP live", add_source_and_inbound),
        ("Disable carrier routing", disable_rejects_routes),
        ("Recover configuration", restart_persistence),
    ]
    for name, fn in tests:
        check(name, fn)
    failed = len([x for x in RESULTS if not x])
    print(
        f"\nBYOC v1 acceptance {'FAILED' if failed else 'PASSED'}: {failed} failure(s)."
    )
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
