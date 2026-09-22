#!/usr/bin/env python3
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid

API = os.getenv("GRACEFUL_DRAIN_API_BASE", "http://127.0.0.1:8080")
TOKEN = os.getenv(
    "GRACEFUL_DRAIN_TOKEN",
    "lm_org_v1smoke0_v1smoke0abcdefghijklmnopqrstuvwx",
)
ESL_PASSWORD = os.getenv("FREESWITCH_ESL_PASSWORD", "graceful-drain-esl-secret")
DID = "+15551234567"
CALLER = "+15557654321"
COMPOSE = [
    "docker",
    "compose",
    "-f",
    "deploy/compose.yaml",
    "-f",
    "tests/graceful-drain/compose.yaml",
]
STATE = {}


class Failure(RuntimeError):
    pass


def run(args, check=True):
    proc = subprocess.run(
        args,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    if check and proc.returncode:
        raise Failure(f"command failed: {' '.join(args)}\n{proc.stdout}")
    return proc.stdout.strip()


def compose(*args, check=True):
    return run(COMPOSE + list(args), check)


def compose_succeeds(*args):
    return (
        subprocess.run(
            COMPOSE + list(args),
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        ).returncode
        == 0
    )


def fs_args(service, command):
    return COMPOSE + [
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
    ]


def fs(service, command):
    return run(fs_args(service, command))


def api(method, path, payload=None, expected=(200,)):
    data = json.dumps(payload).encode() if payload is not None else None
    headers = {
        "Accept": "application/json",
        "Authorization": f"Bearer {TOKEN}",
    }
    if data is not None:
        headers["Content-Type"] = "application/json"

    request = urllib.request.Request(
        API + path,
        data=data,
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            status = response.status
            raw = response.read()
    except urllib.error.HTTPError as error:
        status = error.code
        raw = error.read()

    if status not in expected:
        raise Failure(
            f"{method} {path}: HTTP {status}, body={raw.decode(errors='replace')}"
        )
    if not raw:
        return None

    body = json.loads(raw)
    if isinstance(body, dict) and body.get("success") is True:
        return body.get("data")
    return body


def wait(description, probe, timeout=30):
    deadline = time.monotonic() + timeout
    last_error = None
    while time.monotonic() < deadline:
        try:
            value = probe()
            if value:
                return value
        except Failure:
            raise
        except Exception as error:  # noqa: BLE001 - retain transient probe diagnostics
            last_error = error
        time.sleep(0.25)
    detail = f": {last_error}" if last_error else ""
    raise Failure(f"timed out waiting for {description}{detail}")


def numeric(output, description):
    value = output.strip()
    if not value.isdigit():
        raise Failure(f"{description} is not numeric: {output!r}")
    return int(value)


def channel_count(service):
    output = fs(service, "show channels count")
    match = re.search(r"(?m)^\s*(\d+)\s+total\.\s*$", output)
    if not match:
        raise Failure(f"unable to parse {service} channel count: {output!r}")
    return int(match.group(1))


def channel_exists(service, channel_id):
    output = fs(service, f"uuid_exists {channel_id}").strip().lower()
    if output in ("true", "false"):
        return output == "true"
    raise Failure(f"unable to parse {service} uuid_exists response: {output!r}")


def opensips_dialogs():
    return numeric(
        compose(
            "exec",
            "-T",
            "opensips",
            "/usr/local/bin/leamout-opensips-drain",
            "dialogs",
        ),
        "OpenSIPS dialog count",
    )


def freeswitch_channels():
    return numeric(
        compose(
            "exec",
            "-T",
            "freeswitch",
            "/usr/local/bin/leamout-freeswitch-drain",
            "channels",
        ),
        "FreeSWITCH channel count",
    )


def rtpengine_media_sockets():
    output = compose(
        "exec",
        "-T",
        "rtpengine",
        "sh",
        "-lc",
        "netstat -anu 2>/dev/null | awk 'NR > 2 { n=split($4,a,\":\"); p=a[n]+0; if (p >= 23000 && p <= 32768) count++ } END { print count+0 }'",
    )
    return numeric(output, "RTPengine media socket count")


def provision():
    # Carrier-provider inventory is internal, seeded by migration 013.
    provider_id = compose(
        "exec", "-T", "postgres", "psql", "-U", "leamout", "-d", "leamout",
        "-Atc",
        "SELECT id::text FROM carrier_providers "
        "WHERE slug='generic-sip' AND adapter='sip' AND status='active'",
    ).strip()
    if not provider_id:
        raise Failure("migration-seeded generic SIP provider is unavailable")

    connection = api(
        "POST",
        "/v1/carrier-connections/",
        {
            "provider_id": provider_id,
            "name": "Graceful drain synthetic carrier",
            "inbound_enabled": True,
        },
        (201,),
    )
    STATE["connection"] = connection

    api(
        "POST",
        f"/v1/carrier-connections/{connection['id']}/source-ips",
        {"cidr": "172.30.0.50/32"},
        (201,),
    )

    number = api(
        "POST",
        "/v1/numbers/byoc",
        {
            "number": DID,
            "country_code": "US",
            "carrier_connection_id": connection["id"],
            "voice_enabled": True,
        },
        (201,),
    )
    STATE["number"] = number
    if number.get("type") != "byoc":
        raise Failure("test DID was not created as a BYOC number")
    if number.get("carrier_connection_id") != connection["id"]:
        raise Failure("test DID was not created on the carrier connection")

    app = api(
        "POST",
        "/v1/voice-applications/",
        {"name": "Graceful drain ingress", "caller_id": CALLER},
        (201,),
    )
    STATE["app"] = app
    api(
        "POST",
        f"/v1/voice-applications/{app['id']}/bindings",
        {"phone_number_id": number["id"]},
        (201,),
    )


def establish_call():
    before = {call["id"] for call in api("GET", "/v1/calls/?limit=100")["calls"]}

    originate_uuid = str(uuid.uuid4())
    originate_command = (
        "originate "
        f"{{origination_uuid={originate_uuid},origination_caller_id_number={CALLER},originate_timeout=15}}"
        f"sofia/internal/{DID}@opensips:5060 &park()"
    )
    originate = subprocess.Popen(
        fs_args("graceful-drain-carrier", originate_command),
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    STATE["originate"] = originate
    STATE["originate_uuid"] = originate_uuid

    def inbound_call():
        calls = api("GET", "/v1/calls/?limit=100")["calls"]
        call = next(
            (
                item
                for item in calls
                if item["id"] not in before and item.get("direction") == "inbound"
            ),
            None,
        )
        if call is not None:
            return call

        if originate.poll() is not None:
            output, _ = originate.communicate()
            raise Failure(
                "carrier originate ended before Leamout created an inbound call "
                f"(exit={originate.returncode}, uuid={originate_uuid}): {output.strip()!r}"
            )
        return None

    call = wait("inbound call record", inbound_call, 30)
    STATE["call"] = call
    if not call.get("sip_call_id"):
        raise Failure("inbound call is missing SIP Call-ID attribution")

    # The action endpoint returns an acknowledgement, not a CallResponse.
    # Poll the persisted call after executing the real FreeSWITCH answer action.
    api("POST", f"/v1/calls/{call['id']}/answer")

    def connected_call():
        current = api("GET", f"/v1/calls/{call['id']}")
        return current if current.get("state") in ("answered", "active") else None

    current = wait("answered call state", connected_call, 20)
    STATE["call"] = current

    wait("OpenSIPS dialog", lambda: opensips_dialogs() > 0, 15)
    wait("Leamout FreeSWITCH channel", lambda: freeswitch_channels() > 0, 15)
    wait(
        "synthetic carrier FreeSWITCH channel",
        lambda: channel_count("graceful-drain-carrier") > 0,
        15,
    )
    wait("RTPengine media sockets", lambda: rtpengine_media_sockets() >= 2, 15)

    print(
        "PASS established real SIP/media call "
        f"call={current['id']} sip_call_id={current['sip_call_id']} "
        f"opensips_dialogs={opensips_dialogs()} "
        f"freeswitch_channels={freeswitch_channels()} "
        f"carrier_channels={channel_count('graceful-drain-carrier')} "
        f"rtpengine_media_sockets={rtpengine_media_sockets()}"
    )


def restart_control_plane():
    call = STATE["call"]
    channel_id = call["sip_call_id"]

    compose("restart", "server", "worker")

    wait(
        "worker readiness after restart",
        lambda: compose_succeeds(
            "exec",
            "-T",
            "worker",
            "wget",
            "--spider",
            "-q",
            "http://127.0.0.1:8081/readyz",
        ),
        60,
    )

    def recovered_call():
        current = api("GET", f"/v1/calls/{call['id']}")
        return current if current.get("state") in ("answered", "active") else None

    STATE["call"] = wait("active call after control-plane restart", recovered_call, 60)
    if not channel_exists("freeswitch", channel_id):
        raise Failure("control-plane restart interrupted the active FreeSWITCH channel")
    if opensips_dialogs() == 0 or rtpengine_media_sockets() < 2:
        raise Failure("control-plane restart interrupted active SIP/media state")

    print("PASS server and worker restart preserved the active SIP/media session")


def cleanup_call():
    call = STATE.get("call")
    if not call:
        return

    channel_id = call["sip_call_id"]
    api("POST", f"/v1/calls/{call['id']}/hangup")
    time.sleep(0.5)
    updated = api("GET", f"/v1/calls/{call['id']}")
    print(
        "POST-HANGUP "
        f"api_state={updated.get('state')} "
        f"leamout_uuid_exists={channel_exists('freeswitch', channel_id)} "
        f"freeswitch_channels={freeswitch_channels()} "
        f"carrier_channels={channel_count('graceful-drain-carrier')} "
        f"opensips_dialogs={opensips_dialogs()} "
        f"rtpengine_media_sockets={rtpengine_media_sockets()}"
    )

    wait("Leamout FreeSWITCH channel cleanup", lambda: freeswitch_channels() == 0, 10)
    wait(
        "carrier channel cleanup",
        lambda: channel_count("graceful-drain-carrier") == 0,
        10,
    )
    wait("OpenSIPS dialog cleanup", lambda: opensips_dialogs() == 0, 20)
    wait("RTPengine media cleanup", lambda: rtpengine_media_sockets() == 0, 20)

    originate = STATE.get("originate")
    if originate is not None and originate.poll() is None:
        try:
            originate.communicate(timeout=5)
        except subprocess.TimeoutExpired:
            originate.terminate()
            originate.communicate(timeout=5)

    print("PASS call cleanup released SIP, application, and media state")


def main():
    provision()
    establish_call()
    restart_control_plane()
    cleanup_call()
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as error:  # noqa: BLE001 - acceptance runner needs one failure surface
        print(f"FAIL graceful drain call-path acceptance: {error}", file=sys.stderr)
        sys.exit(1)
