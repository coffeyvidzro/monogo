#!/usr/bin/env python3

import base64
import hashlib
import hmac
import json
import os
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid
import xml.etree.ElementTree as ET

API_BASE = os.getenv("VOICE_V1_API_BASE", "http://127.0.0.1:8080")
TOKEN = os.getenv("VOICE_V1_TOKEN", "lm_org_v1smoke0_v1smoke0abcdefghijklmnopqrstuvwx")
ESL_PASSWORD = os.getenv("FREESWITCH_ESL_PASSWORD", "voice-v1-esl-secret")
DID = os.getenv("VOICE_V1_DID", "+15551234567")
CALLER = os.getenv("VOICE_V1_CALLER", "+15557654321")
ORG_ID = "00000000-0000-0000-0000-000000001001"
COMPOSE = [
    "docker",
    "compose",
    "-f",
    "deploy/compose.yaml",
    "-f",
    "tests/voice-v1/compose.yaml",
]

STATE = {}
RESULTS = {}


class AcceptanceError(RuntimeError):
    pass


def run(command, check=True):
    completed = subprocess.run(
        command,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        check=False,
    )
    if check and completed.returncode != 0:
        raise AcceptanceError(
            f"command failed ({completed.returncode}): {' '.join(command)}\n{completed.stdout}"
        )
    return completed.stdout.strip()


def compose(*args, check=True):
    return run(COMPOSE + list(args), check=check)


def fs_cli(service, command):
    return compose(
        "exec",
        "-T",
        service,
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
        "exec",
        "-T",
        "postgres",
        "psql",
        "-v",
        "ON_ERROR_STOP=1",
        "-U",
        "leamout",
        "-d",
        "leamout",
        "-Atc",
        sql,
    )


def api(method, path, payload=None, auth=True, expected=None):
    body = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        body = json.dumps(payload).encode()
        headers["Content-Type"] = "application/json"
    if auth:
        headers["Authorization"] = f"Bearer {TOKEN}"

    request = urllib.request.Request(
        API_BASE + path,
        data=body,
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            status = response.status
            raw = response.read()
    except urllib.error.HTTPError as error:
        status = error.code
        raw = error.read()
    except OSError as error:
        raise AcceptanceError(f"{method} {path}: {error}") from error

    if expected is not None and status not in expected:
        raise AcceptanceError(
            f"{method} {path}: HTTP {status}, body={raw.decode(errors='replace')}"
        )
    if not raw:
        return status, None
    try:
        parsed = json.loads(raw)
        if (
            isinstance(parsed, dict)
            and parsed.get("success") is True
            and "data" in parsed
        ):
            parsed = parsed["data"]
        return status, parsed
    except json.JSONDecodeError:
        return status, raw.decode(errors="replace")


def wait_for(description, probe, timeout=15, interval=0.25):
    deadline = time.monotonic() + timeout
    last_error = None
    while time.monotonic() < deadline:
        try:
            value = probe()
            if value:
                return value
        except Exception as error:
            last_error = error
        time.sleep(interval)
    suffix = f": {last_error}" if last_error else ""
    raise AcceptanceError(f"timed out waiting for {description}{suffix}")


def health_status(path):
    status, _ = api("GET", path, auth=False)
    return status


def wait_api_ready():
    wait_for("API liveness", lambda: health_status("/healthz") == 204, timeout=45)
    wait_for("API readiness", lambda: health_status("/readyz") == 204, timeout=45)


def get_call(call_id):
    _, call = api("GET", f"/v1/calls/{call_id}", expected={200})
    return call


def wait_call(call_id, predicate, description, timeout=15):
    def probe():
        call = get_call(call_id)
        return call if predicate(call) else False

    return wait_for(description, probe, timeout=timeout)


def list_calls():
    _, body = api("GET", "/v1/calls/?limit=100", expected={200})
    return body["calls"]


def list_recordings():
    _, body = api("GET", "/v1/recordings/?limit=100", expected={200})
    return body["recordings"]


def sink_events():
    cert_dir = os.environ.get("VOICE_V1_CERT_DIR")
    if not cert_dir:
        raise AcceptanceError("VOICE_V1_CERT_DIR is required to verify webhook TLS")
    context = ssl.create_default_context(cafile=os.path.join(cert_dir, "ca.crt"))
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    request = urllib.request.Request("https://127.0.0.1:18443/events")
    with urllib.request.urlopen(request, timeout=5, context=context) as response:
        return json.loads(response.read())["events"]


def verify_signature(event):
    secret = STATE["webhook_secret"]
    secret_bytes = base64.urlsafe_b64decode(secret + "=" * (-len(secret) % 4))
    timestamp = event["headers"]["X-Leamout-Timestamp"]
    body = event["body"]
    expected = (
        "v1="
        + hmac.new(
            secret_bytes,
            f"{timestamp}.{body}".encode(),
            hashlib.sha256,
        ).hexdigest()
    )
    actual = event["headers"]["X-Leamout-Signature"]
    if not hmac.compare_digest(actual, expected):
        raise AcceptanceError("webhook signature did not verify")


def record(number, name, function):
    try:
        detail = function() or ""
        RESULTS[number] = ("PASS", name, detail)
        print(f"PASS {number:02d} {name}{': ' + detail if detail else ''}")
        return True
    except Exception as error:
        RESULTS[number] = ("FAIL", name, str(error))
        print(f"FAIL {number:02d} {name}: {error}")
        return False


def deploy():
    wait_api_ready()
    running = set(compose("ps", "--status", "running", "--services").splitlines())
    required = {
        "postgres",
        "redis",
        "nats",
        "server",
        "worker",
        "opensips",
        "rtpengine",
        "freeswitch",
        "voice-v1-carrier",
        "voice-v1-webhook",
    }
    missing = sorted(required - running)
    if missing:
        raise AcceptanceError("services not running: " + ", ".join(missing))
    return "full control/media stack is running"


def configure_provider():
    _, providers = api("GET", "/v1/carrier-providers/", expected={200})
    provider = next(
        (
            item
            for item in providers["carrier_providers"]
            if item["slug"] == "generic-sip"
        ),
        None,
    )
    if not provider:
        raise AcceptanceError("built-in generic SIP provider is not visible")

    _, connection = api(
        "POST",
        "/v1/carrier-connections/",
        {
            "provider_id": provider["id"],
            "name": "voice-v1-carrier",
            "inbound_enabled": True,
            "codecs": ["PCMU", "PCMA"],
        },
        expected={201},
    )
    STATE["connection_id"] = connection["id"]
    api(
        "POST",
        f"/v1/carrier-connections/{connection['id']}/source-ips",
        {"cidr": "172.30.0.50/32"},
        expected={201},
    )

    _, trunk = api(
        "POST",
        "/v1/trunks/",
        {
            "type": "byoc",
            "carrier_connection_id": connection["id"],
            "name": "voice-v1-trunk",
            "direction": "bidirectional",
        },
        expected={201},
    )
    STATE["trunk_id"] = trunk["id"]
    if trunk.get("type") != "byoc":
        raise AcceptanceError("test trunk was not created as a BYOC trunk")
    if trunk.get("carrier_connection_id") != connection["id"]:
        raise AcceptanceError("test trunk was not created on the carrier connection")

    _, endpoint = api(
        "POST",
        f"/v1/trunks/{trunk['id']}/endpoints",
        {
            "host": "voice-v1-carrier",
            "port": 5060,
            "transport": "udp",
            "direction": "bidirectional",
        },
        expected={201},
    )
    if endpoint["host"] != "voice-v1-carrier":
        raise AcceptanceError("trunk endpoint was not persisted")
    return f"carrier {connection['id']} routes to the synthetic SIP peer"


def create_voice_application():
    _, number = api(
        "POST",
        "/v1/numbers/",
        {
            "type": "byoc",
            "number": DID,
            "country_code": "US",
            "carrier_connection_id": STATE["connection_id"],
            "voice_enabled": True,
        },
        expected={201},
    )
    STATE["number_id"] = number["id"]
    if number.get("type") != "byoc":
        raise AcceptanceError("test DID was not created as a BYOC number")
    if number.get("carrier_connection_id") != STATE["connection_id"]:
        raise AcceptanceError("test DID was not created on the carrier connection")

    # Current outbound routing requires the caller identity itself to be an
    # owned, voice-enabled number on the same carrier connection as the trunk.
    _, caller_number = api(
        "POST",
        "/v1/numbers/",
        {
            "type": "byoc",
            "number": CALLER,
            "country_code": "US",
            "carrier_connection_id": STATE["connection_id"],
            "voice_enabled": True,
        },
        expected={201},
    )
    STATE["caller_number_id"] = caller_number["id"]
    if caller_number.get("type") != "byoc":
        raise AcceptanceError("caller identity was not created as a BYOC number")
    if caller_number.get("carrier_connection_id") != STATE["connection_id"]:
        raise AcceptanceError(
            "caller identity was not created on the carrier connection"
        )

    _, application = api(
        "POST",
        "/v1/voice-applications/",
        {"name": "voice-v1-acceptance", "caller_id": CALLER},
        expected={201},
    )
    STATE["application_id"] = application["id"]
    _, binding = api(
        "POST",
        f"/v1/voice-applications/{application['id']}/bindings",
        {"phone_number_id": number["id"]},
        expected={201},
    )
    if binding.get("phone_number_id") != number["id"]:
        raise AcceptanceError("DID binding was not persisted")
    return f"voice application {application['id']} is bound to {DID}"


def configure_webhook():
    _, created = api(
        "POST",
        "/v1/webhooks/",
        {
            "url": "https://voice-v1-webhook:8443/events",
            "subscribed_events": [
                "call.initiated",
                "call.ringing",
                "call.answered",
                "call.held",
                "call.resumed",
                "call.completed",
                "recording.started",
                "recording.completed",
            ],
        },
        expected={201},
    )
    STATE["webhook_id"] = created["webhook"]["id"]
    STATE["webhook_secret"] = created["signing_secret"]
    _, test_result = api(
        "POST",
        f"/v1/webhooks/{STATE['webhook_id']}/test",
        expected={200},
    )
    if test_result["response_status"] != 204:
        raise AcceptanceError(f"webhook test returned {test_result['response_status']}")


def inbound_call():
    existing = {item["id"] for item in list_calls()}
    carrier_uuid = str(uuid.uuid4())
    output = fs_cli(
        "voice-v1-carrier",
        "bgapi originate "
        f"{{origination_uuid={carrier_uuid},origination_caller_id_number={CALLER}}}"
        f"sofia/internal/{DID}@opensips:5060 &park()",
    )
    if "+OK Job-UUID:" not in output:
        raise AcceptanceError(f"synthetic carrier originate was not queued: {output}")

    def probe():
        return next(
            (
                item
                for item in list_calls()
                if item["id"] not in existing
                and item["direction"] == "inbound"
                and item["to"] == DID
            ),
            False,
        )

    inbound = wait_for("inbound call persistence", probe, timeout=15)
    STATE["inbound_call_id"] = inbound["id"]
    STATE["inbound_carrier_uuid"] = carrier_uuid
    return f"carrier ingress created live inbound call {inbound['id']}"


def answer_inbound():
    call_id = STATE["inbound_call_id"]
    try:
        _, call = api("POST", f"/v1/calls/{call_id}/answer", expected={200})
        if call["state"] not in {"answered", "active"}:
            raise AcceptanceError(f"answer state is {call['state']}")

        _, ended = api("POST", f"/v1/calls/{call_id}/hangup", expected={200})
        if ended["state"] not in {"completed", "cancelled"}:
            raise AcceptanceError(f"inbound hangup state is {ended['state']}")
        STATE["terminal_call_id"] = call_id
        STATE["terminal_state"] = ended["state"]
        return f"inbound answer and hangup persisted {ended['state']}"
    finally:
        carrier_uuid = STATE.get("inbound_carrier_uuid")
        if carrier_uuid:
            fs_cli("voice-v1-carrier", f"uuid_kill {carrier_uuid}")


def outbound_call():
    _, call = api(
        "POST",
        "/v1/calls/",
        {
            "application_id": STATE["application_id"],
            "trunk_id": STATE["trunk_id"],
            "from": CALLER,
            "to": DID,
        },
        expected={201},
    )
    if call["direction"] != "outbound" or not call.get("sip_call_id"):
        raise AcceptanceError("outbound call has no media identity")
    if call["state"] not in {"answered", "active"}:
        raise AcceptanceError(f"answered outbound call persisted as {call['state']}")
    STATE["call_id"] = call["id"]
    STATE["sip_call_id"] = call["sip_call_id"]
    return f"outbound call {call['id']} reached an answered carrier leg"


def hold_resume():
    call_id = STATE["call_id"]
    api("POST", f"/v1/calls/{call_id}/hold", expected={200})
    wait_call(call_id, lambda call: call["media_state"] == "held", "held media state")
    api("POST", f"/v1/calls/{call_id}/unhold", expected={200})
    wait_call(
        call_id, lambda call: call["media_state"] == "active", "active media state"
    )
    return "media_state changed active -> held -> active"


def play_audio():
    call_id = STATE["call_id"]
    path = "tone_stream://%(30000,0,440)"
    api(
        "POST",
        f"/v1/calls/{call_id}/play",
        {"path": path},
        expected={200},
    )
    time.sleep(0.2)
    if (
        "true"
        not in fs_cli("freeswitch", f"uuid_exists {STATE['sip_call_id']}").lower()
    ):
        raise AcceptanceError("media channel disappeared while playback was active")
    api("POST", f"/v1/calls/{call_id}/stop", expected={200})
    if (
        "true"
        not in fs_cli("freeswitch", f"uuid_exists {STATE['sip_call_id']}").lower()
    ):
        raise AcceptanceError("media channel disappeared during playback")
    return "playback and stop executed on a live FreeSWITCH channel"


def record_audio():
    call_id = STATE["call_id"]
    path = "/var/lib/freeswitch/recordings/voice-v1-acceptance.wav"
    api(
        "POST",
        f"/v1/calls/{call_id}/record",
        {"action": "start", "path": path},
        expected={200},
    )

    def started():
        return next(
            (
                item
                for item in list_recordings()
                if item["call_id"] == call_id and item["status"] == "recording"
            ),
            False,
        )

    recording = wait_for("recording.started persistence", started)
    api(
        "POST",
        f"/v1/calls/{call_id}/record",
        {"action": "stop", "path": path},
        expected={200},
    )

    def completed():
        return next(
            (
                item
                for item in list_recordings()
                if item["id"] == recording["id"] and item["status"] == "completed"
            ),
            False,
        )

    wait_for("recording.completed persistence", completed)
    return f"recording {recording['id']} completed through lifecycle events"


def transfer():
    call_id = STATE["call_id"]
    api(
        "POST",
        f"/v1/calls/{call_id}/transfer",
        {"destination": "9196", "dialplan": "XML", "context": "leamout"},
        expected={200},
    )
    time.sleep(0.25)
    if (
        "true"
        not in fs_cli("freeswitch", f"uuid_exists {STATE['sip_call_id']}").lower()
    ):
        raise AcceptanceError("transferred channel is no longer live")
    return "live call transferred to the local 9196 dialplan"


def hangup_outbound():
    _, call = api("POST", f"/v1/calls/{STATE['call_id']}/hangup", expected={200})
    if call["state"] not in {"completed", "cancelled"}:
        raise AcceptanceError(f"hangup state is {call['state']}")
    STATE["terminal_call_id"] = STATE["call_id"]
    STATE["terminal_state"] = call["state"]
    return f"outbound cleanup persisted {call['state']}"


def conference():
    name = "voice-v1-" + uuid.uuid4().hex[:8]
    _, item = api(
        "POST",
        "/v1/conferences/",
        {"application_id": STATE["application_id"], "name": name},
        expected={201},
    )
    if item["state"] != "active":
        raise AcceptanceError("conference API did not create active state")

    fs_cli(
        "freeswitch",
        "bgapi originate "
        f"{{origination_caller_id_number={CALLER}}}"
        f"loopback/9196/leamout &conference({name}@default)",
    )
    wait_for(
        "conference media room",
        lambda: name if name in fs_cli("freeswitch", "conference list") else False,
        timeout=15,
    )

    try:
        api("POST", f"/v1/conferences/{item['id']}/lock", expected={200})
        conference_xml = ET.fromstring(fs_cli("freeswitch", "conference xml_list"))
        locked = next(
            (
                conference.get("locked")
                for conference in conference_xml.iter("conference")
                if conference.get("name") == name
            ),
            None,
        )
        if locked != "true":
            raise AcceptanceError("conference lock is not observable in FreeSWITCH")
        api("POST", f"/v1/conferences/{item['id']}/unlock", expected={200})
        api("DELETE", f"/v1/conferences/{item['id']}", expected={200})
    finally:
        fs_cli("freeswitch", f"conference {name} kick all")

    return "conference lifecycle and controls are observable in FreeSWITCH"


def normalized_events():
    required = {
        "call.initiated",
        "call.answered",
        "call.held",
        "call.resumed",
        "call.completed",
    }

    def probe():
        found = {event["envelope"].get("type") for event in sink_events()}
        return found if required.issubset(found) else False

    found = wait_for("normalized call events", probe, timeout=20)
    return "observed " + ", ".join(sorted(required & found))


def query_call_state():
    call = get_call(STATE["terminal_call_id"])
    if call["state"] != STATE["terminal_state"]:
        raise AcceptanceError("queried state does not match terminal mutation")
    if call["organization_id"] != ORG_ID:
        raise AcceptanceError("queried call escaped acceptance organization")
    return f"durable call state is {call['state']}"


def webhooks():
    events = wait_for("webhook deliveries", lambda: sink_events() or False, timeout=20)
    call_events = [
        event
        for event in events
        if event["envelope"].get("type", "").startswith("call.")
    ]
    if not call_events:
        raise AcceptanceError("no call webhook deliveries received")
    for event in call_events:
        verify_signature(event)

    def delivered_attempt():
        _, current = api(
            "GET",
            f"/v1/webhooks/{STATE['webhook_id']}/deliveries?limit=100",
            expected={200},
        )
        return next(
            (item for item in current["deliveries"] if item["status"] == "succeeded"),
            False,
        )

    wait_for("persisted webhook delivery", delivered_attempt)
    return f"{len(call_events)} signed call deliveries verified"


def health():
    if health_status("/healthz") != 204 or health_status("/readyz") != 204:
        raise AcceptanceError("HTTP health endpoints are not healthy")
    status = fs_cli("freeswitch", "status")
    if "UP" not in status.upper():
        raise AcceptanceError("FreeSWITCH status is not UP")
    fs_cli("freeswitch", "show channels count")
    return "HTTP readiness and FreeSWITCH media status are healthy"


def restart_safety():
    before = get_call(STATE["call_id"])
    compose("restart", "worker")
    wait_for(
        "worker restart",
        lambda: (
            "worker" in compose("ps", "--status", "running", "--services").splitlines()
        ),
        timeout=30,
    )

    compose("restart", "server")
    wait_api_ready()
    if get_call(STATE["call_id"])["state"] != before["state"]:
        raise AcceptanceError("API/worker restart changed terminal call state")

    compose("stop", "freeswitch")
    wait_for(
        "readiness failure after FreeSWITCH stop",
        lambda: health_status("/readyz") == 503,
        timeout=15,
    )
    compose("start", "freeswitch")
    wait_for(
        "readiness recovery after FreeSWITCH start",
        lambda: health_status("/readyz") == 204,
        timeout=45,
    )

    if get_call(STATE["call_id"])["state"] != before["state"]:
        raise AcceptanceError("FreeSWITCH restart changed completed call state")
    return "API, worker, and FreeSWITCH restart with readiness/state recovery"


def print_summary():
    print("\nVoice v1 acceptance matrix")
    print("=" * 72)
    failed = 0
    for number in range(1, 17):
        status, name, detail = RESULTS.get(
            number,
            ("FAIL", "unexecuted acceptance item", "dependency prevented execution"),
        )
        if status != "PASS":
            failed += 1
        suffix = f" - {detail}" if detail else ""
        print(f"{number:02d}. {status:4} {name}{suffix}")
    return failed


def main():
    if not record(1, "Deploy Leamout", deploy):
        return print_summary() or 1
    if not record(2, "Configure a SIP endpoint/provider", configure_provider):
        return print_summary() or 1
    if not record(3, "Create a voice application", create_voice_application):
        return print_summary() or 1

    try:
        configure_webhook()
    except Exception as error:
        RESULTS[14] = ("FAIL", "Receive webhooks", f"webhook setup failed: {error}")
        print(f"FAIL 14 Receive webhooks: webhook setup failed: {error}")

    inbound_ok = record(4, "Receive an inbound call", inbound_call)
    if inbound_ok:
        record(6, "Answer/hang up", answer_inbound)

    if record(5, "Originate an outbound call", outbound_call):
        record(8, "Hold/resume", hold_resume)
        record(9, "Play audio", play_audio)
        record(10, "Record", record_audio)
        record(7, "Transfer", transfer)
        try:
            detail = hangup_outbound()
            print(f"PASS outbound cleanup: {detail}")
        except Exception as error:
            print(f"FAIL outbound cleanup: {error}")
            if RESULTS.get(6, ("FAIL",))[0] == "PASS":
                RESULTS[6] = (
                    "FAIL",
                    "Answer/hang up",
                    f"outbound hangup failed: {error}",
                )

    record(11, "Create/manage conferences", conference)
    record(12, "Receive normalized call events", normalized_events)
    record(13, "Query call state", query_call_state)
    if RESULTS.get(14, ("PASS",))[0] != "FAIL":
        record(14, "Receive webhooks", webhooks)
    record(15, "Inspect call/media health", health)
    record(16, "Restart components without corrupting state", restart_safety)

    failed = print_summary()
    if failed:
        print(
            f"\nVoice v1 acceptance FAILED: {failed} capability check(s) did not pass."
        )
        return 1
    print("\nVoice v1 acceptance PASSED: all 16 capabilities are complete.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
