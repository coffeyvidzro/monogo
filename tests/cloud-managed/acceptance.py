#!/usr/bin/env python3
import json
import hashlib
import os
import subprocess
import time
import urllib.error
import urllib.request

API = os.getenv("CLOUD_MANAGED_API_BASE", "http://127.0.0.1:8080")
PROVIDER = os.getenv("CLOUD_MANAGED_PROVIDER_STATE", "http://127.0.0.1:18090/__state")
WHOLESALE = os.getenv("CLOUD_MANAGED_WHOLESALE", "http://127.0.0.1:18091")
TOKEN = os.getenv(
    "CLOUD_MANAGED_TOKEN", "lm_org_v1smoke0_v1smoke0abcdefghijklmnopqrstuvwx"
)
TOKEN_B = os.getenv(
    "CLOUD_MANAGED_TOKEN_B", "lm_org_v1smoke1_v1smoke1abcdefghijklmnopqrstuvwx"
)
ESL_PASSWORD = os.environ["FREESWITCH_ESL_PASSWORD"]
DID = "+15551236001"
COMPOSE = [
    "docker",
    "compose",
    "-f",
    "deploy/compose.yaml",
    "-f",
    "tests/cloud-managed/compose.yaml",
]


class Failure(RuntimeError):
    pass


def topology():
    result = subprocess.run(
        COMPOSE + ["ps", "--status", "running", "--services"],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    if result.returncode:
        raise Failure(result.stdout)
    required = {
        "postgres",
        "redis",
        "nats",
        "rtpengine",
        "freeswitch",
        "opensips",
        "server",
        "worker",
        "cloud-managed-provider",
        "cloud-managed-wholesale",
    }
    missing = required - set(result.stdout.splitlines())
    if missing:
        raise Failure(
            "cloud-managed topology is missing: " + ", ".join(sorted(missing))
        )


def api(method, path, payload=None, expected=(200,), token=TOKEN):
    data = None if payload is None else json.dumps(payload).encode()
    headers = {"Accept": "application/json", "Authorization": f"Bearer {token}"}
    if data is not None:
        headers["Content-Type"] = "application/json"
        headers["Idempotency-Key"] = (
            "cloud-managed-" + hashlib.sha256((method + path).encode()).hexdigest()
        )
    request = urllib.request.Request(
        API + path, data=data, headers=headers, method=method
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            status, raw = response.status, response.read()
    except urllib.error.HTTPError as error:
        status, raw = error.code, error.read()
    if status not in expected:
        raise Failure(f"{method} {path}: HTTP {status}: {raw.decode(errors='replace')}")
    body = json.loads(raw) if raw else None
    return (
        body["data"] if isinstance(body, dict) and body.get("success") is True else body
    )


def rejected_api(method, path, payload=None, token=TOKEN):
    data = None if payload is None else json.dumps(payload).encode()
    headers = {"Accept": "application/json", "Authorization": f"Bearer {token}"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(
        API + path, data=data, headers=headers, method=method
    )
    try:
        urllib.request.urlopen(request, timeout=10)
    except urllib.error.HTTPError as error:
        if 400 <= error.code < 500:
            return error.code
        raise Failure(f"expected policy denial, got HTTP {error.code}") from error
    raise Failure(f"{method} {path} unexpectedly succeeded")


def sql(statement):
    result = subprocess.run(
        COMPOSE
        + [
            "exec",
            "-T",
            "postgres",
            "psql",
            "-U",
            "leamout",
            "-d",
            "leamout",
            "-Atc",
            statement,
        ],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    if result.returncode:
        raise Failure(result.stdout)
    return result.stdout.strip()


def fs_cli(command):
    result = subprocess.run(
        COMPOSE
        + [
            "exec",
            "-T",
            "freeswitch",
            "fs_cli",
            "-H",
            "127.0.0.1",
            "-P",
            "8021",
            "-p",
            ESL_PASSWORD,
            "-x",
            command,
        ],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    if result.returncode:
        raise Failure(result.stdout)
    return result.stdout.strip()


def wait_for(description, probe, timeout=35):
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            value = probe()
            if value:
                return value
        except Exception as error:
            last = error
        time.sleep(0.25)
    raise Failure(f"timed out waiting for {description}: {last}")


def main():
    topology()
    print("PASS Leamout-operated Cloud + Managed runtime topology is healthy")
    available = api("GET", "/v1/numbers/available?country_code=US&contains=5551236")[
        "numbers"
    ]
    if len(available) != 1 or available[0]["number"] != DID:
        raise Failure(f"unexpected managed inventory: {available}")
    selection = available[0]["selection_id"]
    if not selection.startswith("sel_"):
        raise Failure("inventory did not return an opaque selection handle")
    print("PASS provider inventory is exposed as an opaque managed-number selection")

    number = api(
        "POST", "/v1/numbers/", {"type": "managed", "selection_id": selection}, (201,)
    )
    if number["status"] != "provisioning" or number["type"] != "managed":
        raise Failure(f"managed purchase did not persist provisioning intent: {number}")
    if number.get("carrier_connection_id") is not None:
        raise Failure("customer response exposed the platform carrier connection")

    def active_number():
        current = api("GET", f"/v1/numbers/{number['id']}")
        return current if current["status"] == "active" else None

    active = wait_for("provider operation completion", active_number)
    if active["number"] != DID or not active["voice_enabled"]:
        raise Failure(f"activated managed number is invalid: {active}")
    if (
        "provider_id" in active
        or "provider_resource_id" in active
        or active.get("carrier_connection_id") is not None
    ):
        raise Failure("customer response leaked internal provider or carrier state")

    provider = json.load(urllib.request.urlopen(PROVIDER, timeout=5))
    if provider["order_posts"] != 1 or provider["routing_patches"] != 1:
        raise Failure(
            f"provider order/routing workflow was not exactly-once: {provider}"
        )
    if provider["voice_in_trunk_id"] != "voice-in-cloud-1":
        raise Failure(f"DID was not routed to cloud managed ingress: {provider}")
    print("PASS managed DID purchase completed and converged provider routing")
    print(
        "PASS active customer response hides provider and platform routing identifiers"
    )

    application = api(
        "POST",
        "/v1/voice-applications/",
        {"name": "Cloud managed voice", "caller_id": DID},
        (201,),
    )
    api(
        "POST",
        f"/v1/voice-applications/{application['id']}/bindings",
        {"phone_number_id": active["id"]},
        (201,),
    )

    before = {call["id"] for call in api("GET", "/v1/calls/?limit=100")["calls"]}
    request = urllib.request.Request(
        WHOLESALE + "/originate", data=b"{}", method="POST"
    )
    inbound_result = json.load(urllib.request.urlopen(request, timeout=12))
    if 180 not in inbound_result["statuses"]:
        raise Failure(
            f"managed inbound DID did not reach cloud FreeSWITCH: {inbound_result}"
        )

    def inbound_call():
        calls = api("GET", "/v1/calls/?limit=100")["calls"]
        return next(
            (
                call
                for call in calls
                if call["id"] not in before
                and call["direction"] == "inbound"
                and call["to"] == DID
            ),
            None,
        )

    inbound = wait_for("managed inbound call persistence", inbound_call)
    if inbound.get("application_id") != application["id"]:
        raise Failure(
            f"managed inbound call resolved the wrong cloud application: {inbound}"
        )
    print(
        "PASS managed DID reached the local Cloud OpenSIPS and FreeSWITCH application"
    )

    wholesale_before = json.load(urllib.request.urlopen(WHOLESALE, timeout=5))[
        "outbound_invites"
    ]
    try:
        outbound = api(
            "POST",
            "/v1/calls/",
            {
                "application_id": application["id"],
                "from": DID,
                "to": "+15551236099",
            },
            (201,),
        )
    except Failure as error:
        wholesale_state = json.load(urllib.request.urlopen(WHOLESALE, timeout=5))
        channels = subprocess.run(
            COMPOSE
            + [
                "exec",
                "-T",
                "freeswitch",
                "fs_cli",
                "-H",
                "127.0.0.1",
                "-P",
                "8021",
                "-p",
                ESL_PASSWORD,
                "-x",
                "show channels",
            ],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        ).stdout.strip()
        raise Failure(
            f"{error}; wholesale={wholesale_state}; FreeSWITCH channels={channels}"
        ) from error
    if outbound["direction"] != "outbound" or outbound["state"] not in {
        "answered",
        "active",
    }:
        raise Failure(f"trunkless managed outbound call did not connect: {outbound}")

    def wholesale_call():
        current = json.load(urllib.request.urlopen(WHOLESALE, timeout=5))
        return current if current["outbound_invites"] == wholesale_before + 1 else None

    wholesale = wait_for("managed wholesale INVITE", wholesale_call)
    if "+15551236099" not in wholesale["last_destination"]:
        raise Failure(f"managed outbound reached wrong destination: {wholesale}")
    if wholesale["internal_route_header_seen"]:
        raise Failure("internal managed route header leaked to wholesale")
    print("PASS trunkless Cloud call selected the managed default wholesale route")

    provider_state = sql(
        "SELECT pnp.slug || ',' || ccp.slug "
        "FROM phone_numbers pn "
        "JOIN carrier_providers pnp ON pnp.id=pn.provider_id "
        "JOIN calls c ON c.id='" + outbound["id"] + "'::uuid "
        "JOIN carrier_connections cc ON cc.id=c.carrier_connection_id "
        "JOIN carrier_providers ccp ON ccp.id=cc.provider_id "
        "WHERE pn.id='" + active["id"] + "'::uuid"
    ).split(",")
    if provider_state != ["didww", "commpeak"]:
        raise Failure(
            f"adapter identity leaked into SIP carrier identity: {provider_state}"
        )
    wire_call_id = fs_cli(f"uuid_getvar {outbound['sip_call_id']} sip_call_id")
    if wholesale["last_call_id"] != wire_call_id:
        raise Failure("wholesale SIP Call-ID does not match the persisted managed call")
    print(
        "PASS DIDWW number ownership remains independent from CommPeak SIP termination"
    )

    rejected_api("GET", f"/v1/numbers/{active['id']}", token=TOKEN_B)
    rejected_api("GET", f"/v1/calls/{outbound['id']}", token=TOKEN_B)
    wholesale_count = json.load(urllib.request.urlopen(WHOLESALE, timeout=5))[
        "outbound_invites"
    ]
    rejected_api(
        "POST", "/v1/calls/", {"from": DID, "to": "+15551236100"}, token=TOKEN_B
    )
    if (
        json.load(urllib.request.urlopen(WHOLESALE, timeout=5))["outbound_invites"]
        != wholesale_count
    ):
        raise Failure("tenant B caller-ID denial reached the wholesale carrier")
    print(
        "PASS tenant B cannot read tenant A resources or use tenant A managed caller ID"
    )

    rejected_api(
        "POST",
        "/v1/calls/",
        {
            "trunk_id": "00000000-0000-0000-0000-000000006099",
            "from": DID,
            "to": "+15551236101",
        },
    )
    if (
        json.load(urllib.request.urlopen(WHOLESALE, timeout=5))["outbound_invites"]
        != wholesale_count
    ):
        raise Failure("failed explicit trunk request fell back to managed wholesale")

    sql(
        "UPDATE trunks SET status='disabled' WHERE id='00000000-0000-0000-0000-000000006021'"
    )
    try:
        rejected_api("POST", "/v1/calls/", {"from": DID, "to": "+15551236102"})
    finally:
        sql(
            "UPDATE trunks SET status='active' WHERE id='00000000-0000-0000-0000-000000006021'"
        )
    if (
        json.load(urllib.request.urlopen(WHOLESALE, timeout=5))["outbound_invites"]
        != wholesale_count
    ):
        raise Failure("missing managed default route fell through to a tenant route")
    print("PASS explicit-trunk failure and missing managed route both fail closed")

    sql(
        "UPDATE phone_numbers SET status='disabled' WHERE id='"
        + active["id"]
        + "'::uuid"
    )
    try:
        request = urllib.request.Request(
            WHOLESALE + "/originate", data=b"{}", method="POST"
        )
        denied_inbound = json.load(urllib.request.urlopen(request, timeout=12))
    finally:
        sql(
            "UPDATE phone_numbers SET status='active' WHERE id='"
            + active["id"]
            + "'::uuid"
        )
    if not denied_inbound["statuses"] or denied_inbound["statuses"][-1] != 404:
        raise Failure(f"inactive managed DID did not fail closed: {denied_inbound}")
    print("PASS inactive managed DID is denied before the Cloud media runtime")

    sql(
        "UPDATE organizations SET status='disabled' WHERE id='00000000-0000-0000-0000-000000006001'"
    )
    try:
        rejected_api("POST", "/v1/calls/", {"from": DID, "to": "+15551236103"})
        request = urllib.request.Request(
            WHOLESALE + "/originate", data=b"{}", method="POST"
        )
        disabled_org_inbound = json.load(urllib.request.urlopen(request, timeout=12))
    finally:
        sql(
            "UPDATE organizations SET status='active' WHERE id='00000000-0000-0000-0000-000000006001'"
        )
    if (
        not disabled_org_inbound["statuses"]
        or disabled_org_inbound["statuses"][-1] != 404
    ):
        raise Failure(
            f"disabled organization received managed inbound call: {disabled_org_inbound}"
        )
    if (
        json.load(urllib.request.urlopen(WHOLESALE, timeout=5))["outbound_invites"]
        != wholesale_count
    ):
        raise Failure("disabled organization reached managed wholesale")
    print("PASS disabled organization is denied on managed inbound and outbound paths")


if __name__ == "__main__":
    main()
