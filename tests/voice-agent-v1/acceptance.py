#!/usr/bin/env python3

import json
import os
import ssl
import subprocess
import time
import urllib.error
import urllib.request
import uuid

API_BASE = os.getenv("VOICE_AGENT_V1_API_BASE", "http://127.0.0.1:8080")
TOKEN = os.getenv("VOICE_AGENT_V1_TOKEN", "lm_org_v1smoke0_v1smoke0abcdefghijklmnopqrstuvwx")
ESL_PASSWORD = os.getenv("FREESWITCH_ESL_PASSWORD", "voice-agent-v1-esl-secret")
DID = os.getenv("VOICE_AGENT_V1_DID", "+15551234601")
CALLER = os.getenv("VOICE_AGENT_V1_CALLER", "+15557654601")
ORG_ID = "00000000-0000-0000-0000-000000001101"
INITIAL_INSTRUCTIONS = "You are the Voice Agent v1 acceptance assistant."
COMPOSE = [
    "docker", "compose",
    "-f", "deploy/compose.yaml",
    "-f", "tests/voice-agent-v1/compose.yaml",
    "-f", "tests/acceptance-minio.yaml",
]
STATE = {}


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


def psql(sql):
    return compose(
        "exec", "-T", "postgres",
        "psql", "-v", "ON_ERROR_STOP=1",
        "-U", "leamout", "-d", "leamout", "-Atc", sql,
    )


def redis_cli(*args):
    return compose("exec", "-T", "redis", "redis-cli", "--raw", *args)


def fs_cli(service, command):
    return compose(
        "exec", "-T", service,
        "fs_cli", "-H", "127.0.0.1", "-P", "8021",
        "-p", ESL_PASSWORD, "-x", command,
    )


def api(method, path, payload=None, expected=None):
    body = None
    headers = {
        "Accept": "application/json",
        "Authorization": f"Bearer {TOKEN}",
    }
    if payload is not None:
        body = json.dumps(payload).encode()
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(
        API_BASE + path, data=body, headers=headers, method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            status = response.status
            raw = response.read()
    except urllib.error.HTTPError as error:
        status = error.code
        raw = error.read()
    if expected is not None and status not in expected:
        raise AcceptanceError(
            f"{method} {path}: HTTP {status}, body={raw.decode(errors='replace')}"
        )
    if not raw:
        return status, None
    parsed = json.loads(raw)
    if isinstance(parsed, dict) and parsed.get("success") is True and "data" in parsed:
        parsed = parsed["data"]
    return status, parsed


def wait_for(description, probe, timeout=30, interval=0.25):
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


def get_call(call_id):
    return api("GET", f"/v1/calls/{call_id}", expected={200})[1]


def list_calls():
    return api("GET", "/v1/calls/?limit=100", expected={200})[1]["calls"]


def fake_openai_state():
    cert_dir = os.environ["VOICE_AGENT_V1_CERT_DIR"]
    context = ssl.create_default_context(cafile=os.path.join(cert_dir, "ca.crt"))
    with urllib.request.urlopen(
        "https://127.0.0.1:18444/state",
        timeout=5,
        context=context,
    ) as response:
        return json.loads(response.read())


def setup_carrier():
    trunk = api(
        "POST", "/v1/trunks/",
        {
            "name": "voice-agent-v1-trunk",
            "direction": "bidirectional",
            "inbound_enabled": True,
            "codecs": ["PCMU", "PCMA"],
        },
        expected={201},
    )[1]
    STATE["trunk_id"] = trunk["id"]
    api(
        "POST",
        f"/v1/trunks/{trunk['id']}/source-ips",
        {"cidr": "172.30.0.50/32"},
        expected={201},
    )
    api(
        "POST",
        f"/v1/trunks/{trunk['id']}/endpoints",
        {
            "host": "voice-agent-v1-carrier",
            "port": 5060,
            "transport": "udp",
            "direction": "bidirectional",
        },
        expected={201},
    )


def setup_numbers():
    number = api(
        "POST", "/v1/numbers/",
        {
            "number": DID,
            "country_code": "US",
            "trunk_id": STATE["trunk_id"],
            "voice_enabled": True,
        },
        expected={201},
    )[1]
    STATE["number_id"] = number["id"]
    caller = api(
        "POST", "/v1/numbers/",
        {
            "number": CALLER,
            "country_code": "US",
            "trunk_id": STATE["trunk_id"],
            "voice_enabled": True,
        },
        expected={201},
    )[1]
    STATE["caller_number_id"] = caller["id"]
    if number.get("trunk_id") != STATE["trunk_id"]:
        raise AcceptanceError("Voice Agent DID was not created on the SIP trunk")
    if caller.get("trunk_id") != STATE["trunk_id"]:
        raise AcceptanceError("Voice Agent caller identity was not created on the SIP trunk")


def setup_voice_agent():
    agent = api(
        "POST", "/v1/voice-agents/",
        {
            "name": "voice-agent-v1",
            "engine": "integrated",
            "instructions": INITIAL_INSTRUCTIONS,
            "voice": "alloy",
            "language": "en",
            "engine_config": {},
        },
        expected={201},
    )[1]
    STATE["agent_id"] = agent["id"]

    provider_credential = api(
        "POST",
        "/v1/ai-provider-credentials/",
        {
            "provider": "openai",
            "name": "voice-agent-v1-openai",
            "secret": "tenant-provider-test-token",
        },
        expected={201},
    )[1]
    STATE["provider_credential_id"] = provider_credential["id"]
    if "secret" in provider_credential:
        raise AcceptanceError("provider credential secret leaked from create API")

    provider_binding = api(
        "PUT",
        f"/v1/voice-agents/{agent['id']}/providers/realtime",
        {
            "provider": "openai",
            "credential_id": provider_credential["id"],
            "config": {},
        },
        expected={200},
    )[1]
    if provider_binding["credential_id"] != provider_credential["id"]:
        raise AcceptanceError("Voice Agent provider binding was not persisted")

    webhook_tool = api(
        "POST",
        f"/v1/voice-agents/{agent['id']}/tools/",
        {
            "type": "webhook",
            "name": "lookup_customer",
            "description": "Look up a customer.",
            "parameters": {
                "type": "object",
                "properties": {"customer_id": {"type": "string"}},
                "required": ["customer_id"],
            },
            "endpoint_url": "https://example.com/voice-agent-tool",
            "timeout_ms": 1000,
        },
        expected={201},
    )[1]
    first_secret = webhook_tool.get("signing_secret", "")
    if len(first_secret) < 32:
        raise AcceptanceError("webhook tool did not return a signing secret")

    listed = api(
        "GET",
        f"/v1/voice-agents/{agent['id']}/tools/",
        expected={200},
    )[1]["tools"]
    created = next(item for item in listed if item["id"] == webhook_tool["id"])
    if "signing_secret" in created:
        raise AcceptanceError("tool signing secret leaked through list API")

    rotated = api(
        "POST",
        f"/v1/voice-agents/{agent['id']}/tools/{webhook_tool['id']}/rotate-signing-secret",
        expected={200},
    )[1]["signing_secret"]
    if rotated == first_secret or len(rotated) < 32:
        raise AcceptanceError("webhook signing secret rotation did not produce a new secret")

    builtin = api(
        "POST",
        f"/v1/voice-agents/{agent['id']}/tools/",
        {
            "type": "builtin",
            "name": "send_dtmf",
            "description": "Send DTMF digits on the current call.",
            "parameters": {
                "type": "object",
                "properties": {"digits": {"type": "string"}},
                "required": ["digits"],
            },
        },
        expected={201},
    )[1]
    STATE["tool_id"] = builtin["id"]

    verification = api(
        "POST",
        f"/v1/ai-integrations/{provider_credential['id']}/verify",
        expected={200},
    )[1]
    if verification.get("connection_state") != "ready":
        raise AcceptanceError(
            f"OpenAI integration verification state = {verification.get('connection_state')!r}"
        )

    readiness = api(
        "GET",
        f"/v1/voice-agents/{agent['id']}/readiness",
        expected={200},
    )[1]
    if not readiness.get("ready"):
        raise AcceptanceError(
            f"Voice Agent readiness failed: {json.dumps(readiness.get('issues') or [])}"
        )

    activated = api(
        "POST",
        f"/v1/voice-agents/{agent['id']}/activate",
        expected={200},
    )[1]
    if activated.get("active_revision") is None:
        raise AcceptanceError("Voice Agent activation did not persist an active revision")
    STATE["active_revision"] = activated["active_revision"]

    binding = api(
        "POST",
        f"/v1/voice-agents/{agent['id']}/bindings",
        {"phone_number_id": STATE["number_id"]},
        expected={201},
    )[1]
    if binding["phone_number_id"] != STATE["number_id"]:
        raise AcceptanceError("Voice Agent phone-number binding was not persisted")


def originate_call():
    call = api(
        "POST", "/v1/calls/",
        {
            "voice_agent_id": STATE["agent_id"],
            "trunk_id": STATE["trunk_id"],
            "from_uri": CALLER,
            "to_uri": DID,
        },
        expected={201},
    )[1]
    STATE["call_id"] = call["id"]
    if call.get("voice_agent_id") != STATE["agent_id"]:
        raise AcceptanceError("outbound call was not attributed to the Voice Agent")

    def answered():
        current = get_call(call["id"])
        return current if current["state"] in {"answered", "active"} else False

    current = wait_for("answered Voice Agent call", answered)
    channel_id = compose(
        "exec", "-T", "redis", "redis-cli", "--raw",
        "GET", f"telecom:calls:channel:{call['id']}",
    ).strip()
    try:
        uuid.UUID(channel_id)
    except ValueError as error:
        raise AcceptanceError(f"invalid FreeSWITCH channel id {channel_id!r}") from error
    STATE["channel_id"] = channel_id

    def carrier_channel():
        raw = fs_cli("voice-agent-v1-carrier", "show channels as json")
        payload = json.loads(raw)
        rows = payload.get("rows") or []
        channels = [row.get("uuid") for row in rows if row.get("uuid")]
        return channels[0] if len(channels) == 1 else False

    STATE["carrier_channel_id"] = wait_for(
        "synthetic carrier channel",
        carrier_channel,
        timeout=15,
    )
    return current


def wait_voice_agent_session():
    def probe():
        row = psql(
            "SELECT id::text || '|' || state || '|' || engine || '|' || instructions_snapshot "
            f"FROM voice_agent_sessions WHERE call_id='{STATE['call_id']}'"
        )
        return row if row else False

    row = wait_for("durable Voice Agent session", probe)
    session_id, state, engine, instructions = row.split("|", 3)
    if state != "active" or engine != "integrated":
        raise AcceptanceError(f"unexpected Voice Agent session: {row}")
    if instructions != INITIAL_INSTRUCTIONS:
        raise AcceptanceError("durable session instructions snapshot is incorrect")
    STATE["session_id"] = session_id

    count = psql(
        f"SELECT count(*) FROM voice_agent_sessions WHERE call_id='{STATE['call_id']}'"
    )
    if count != "1":
        raise AcceptanceError(f"Voice Agent session count = {count}, want 1")

    revision = psql(
        "SELECT configuration_revision::text FROM voice_agent_sessions "
        f"WHERE id='{session_id}'"
    )
    if revision != str(STATE["active_revision"]):
        raise AcceptanceError(
            f"session revision = {revision!r}, want active revision {STATE['active_revision']}"
        )

    marker = fs_cli(
        "freeswitch",
        f"uuid_getvar {STATE['channel_id']} leamout_voice_agent_session_id",
    ).strip()
    if marker != session_id:
        raise AcceptanceError(
            f"FreeSWITCH session marker = {marker!r}, want {session_id!r}"
        )


def verify_media_placement():
    owner_key = f"runtime:media:session:{STATE['session_id']}"

    def owner():
        value = redis_cli("GET", owner_key).strip()
        return value if value else False

    node_id = wait_for("Redis media session ownership", owner)
    STATE["media_node_id"] = node_id

    raw = redis_cli("GET", f"runtime:media:node:{node_id}")
    if not raw:
        raise AcceptanceError("media node registry record is missing")
    node = json.loads(raw)
    if node.get("draining") is True:
        raise AcceptanceError("Voice Agent session was placed on a draining media node")
    if int(node.get("capacity") or 0) < 1:
        raise AcceptanceError(f"invalid media node capacity: {node}")
    if not str(node.get("control_url") or "").startswith("http"):
        raise AcceptanceError(f"invalid media node control URL: {node}")


def verify_provider_session():
    def probe():
        state = fake_openai_state()
        return state if state["session_updates"] >= 1 else False

    state = wait_for("OpenAI Realtime session.update", probe)
    session = state.get("last_session") or {}
    if session.get("instructions") != INITIAL_INSTRUCTIONS:
        raise AcceptanceError("provider did not receive the durable instructions snapshot")
    audio = session.get("audio") or {}
    if (audio.get("input") or {}).get("format", {}).get("rate") != 24000:
        raise AcceptanceError("provider input format is not 24 kHz PCM")
    tools = session.get("tools") or []
    names = {tool.get("name") for tool in tools}
    if "send_dtmf" not in names:
        raise AcceptanceError(
            f"provider tool definitions missing send_dtmf: {sorted(names)}"
        )


def verify_provider_credential_isolation():
    listed = api(
        "GET",
        "/v1/ai-provider-credentials/",
        expected={200},
    )[1]["provider_credentials"]
    current = next(
        item for item in listed
        if item["id"] == STATE["provider_credential_id"]
    )
    if "secret" in current:
        raise AcceptanceError("provider credential secret leaked from list API")
    if current.get("connection_state") != "ready":
        raise AcceptanceError("verified provider credential did not remain ready")

    ciphertext = psql(
        "SELECT secret_ciphertext FROM ai_provider_credentials "
        f"WHERE id='{STATE['provider_credential_id']}'"
    )
    if not ciphertext or ciphertext == "tenant-provider-test-token":
        raise AcceptanceError("provider credential was not encrypted at rest")

    engine_snapshot = psql(
        "SELECT engine_config_snapshot::text FROM voice_agent_sessions "
        f"WHERE id='{STATE['session_id']}'"
    )
    if "tenant-provider-test-token" in engine_snapshot:
        raise AcceptanceError("provider credential leaked into engine snapshot")


def verify_snapshot_immutability():
    api(
        "PATCH",
        f"/v1/voice-agents/{STATE['agent_id']}",
        {"instructions": "This change must not affect the active call."},
        expected={200},
    )
    snapshot = psql(
        "SELECT instructions_snapshot FROM voice_agent_sessions "
        f"WHERE id='{STATE['session_id']}'"
    )
    if snapshot != INITIAL_INSTRUCTIONS:
        raise AcceptanceError("active durable session snapshot changed after agent update")


def live_media_state():
    call = get_call(STATE["call_id"])
    durable_state = psql(
        "SELECT state FROM voice_agent_sessions "
        f"WHERE id='{STATE['session_id']}'"
    )
    channel_exists = (
        fs_cli(
            "freeswitch",
            f"uuid_exists {STATE['channel_id']}",
        ).strip().lower()
        == "true"
    )
    carrier_exists = (
        fs_cli(
            "voice-agent-v1-carrier",
            f"uuid_exists {STATE['carrier_channel_id']}",
        ).strip().lower()
        == "true"
    )
    try:
        fork = json.loads(fs_cli("freeswitch", "audio_fork status"))
    except json.JSONDecodeError:
        fork = {}

    return {
        "call_state": call.get("state"),
        "call_media_state": call.get("media_state"),
        "durable_session_state": durable_state,
        "channel_exists": channel_exists,
        "carrier_channel_exists": carrier_exists,
        "audio_fork": fork,
    }


def verify_live_media_attachment():
    def attached():
        state = live_media_state()
        fork = state["audio_fork"]
        if (
            state["call_state"] in {"answered", "active"}
            and state["durable_session_state"] == "active"
            and state["channel_exists"]
            and state["carrier_channel_exists"]
            and fork.get("forks", 0) >= 1
            and fork.get("calls", 0) >= 1
        ):
            return state
        return False

    try:
        return wait_for(
            "live Voice Agent media fork",
            attached,
            timeout=5,
            interval=0.1,
        )
    except AcceptanceError as error:
        raise AcceptanceError(
            f"{error}; lifecycle={json.dumps(live_media_state(), sort_keys=True)}"
        ) from error


def verify_audio_roundtrip():
    before = verify_live_media_attachment()

    response = fs_cli(
        "voice-agent-v1-carrier",
        f"uuid_broadcast {STATE['carrier_channel_id']} tone_stream://%(2500,0,440) aleg",
    )
    if "-ERR" in response:
        raise AcceptanceError(f"synthetic carrier tone failed: {response}")

    deadline = time.monotonic() + 20
    provider = {}
    fork = {}
    while time.monotonic() < deadline:
        provider = fake_openai_state()
        try:
            fork = json.loads(fs_cli("freeswitch", "audio_fork status"))
        except (json.JSONDecodeError, AcceptanceError):
            fork = {}

        if (
            provider.get("audio_appends", 0) > 0
            and provider.get("tool_calls", 0) == 1
            and provider.get("tool_results", 0) == 1
            and provider.get("responses", 0) >= 2
            and fork.get("sent_bytes", 0) > 0
            and fork.get("playback_bytes_played", 0) > 0
        ):
            tool_result = provider.get("last_tool_result") or {}
            if tool_result.get("call_id") != "tool-call-1":
                raise AcceptanceError(
                    f"provider tool result call_id = {tool_result.get('call_id')!r}"
                )
            if json.loads(tool_result.get("output") or "{}") != {"ok": True}:
                raise AcceptanceError(
                    f"provider tool result output = {tool_result.get('output')!r}"
                )
            return
        time.sleep(0.25)

    raise AcceptanceError(
        "Voice Agent audio round trip did not complete: "
        f"before={json.dumps(before, sort_keys=True)} "
        f"after={json.dumps(live_media_state(), sort_keys=True)} "
        f"provider={json.dumps(provider, sort_keys=True)} "
        f"audio_fork={json.dumps(fork, sort_keys=True)}"
    )


def verify_durable_realtime_history():
    def tool_execution():
        row = psql(
            "SELECT state || '|' || tool_call_id || '|' || "
            "COALESCE(response_status::text, '') || '|' || "
            "convert_from(response_body, 'UTF8') "
            "FROM voice_agent_tool_executions "
            f"WHERE session_id='{STATE['session_id']}' "
            "ORDER BY created_at ASC LIMIT 1"
        )
        return row if row else False

    execution = wait_for("durable Voice Agent tool execution", tool_execution)
    state, tool_call_id, status, body = execution.split("|", 3)
    if state != "succeeded" or tool_call_id != "tool-call-1":
        raise AcceptanceError(f"unexpected tool execution: {execution}")
    if status != "200" or json.loads(body) != {"ok": True}:
        raise AcceptanceError(f"unexpected tool result: {execution}")

    def turns():
        raw = psql(
            "SELECT sequence::text || '|' || role || '|' || content || '|' || "
            "COALESCE(tool_name, '') || '|' || COALESCE(tool_call_id, '') "
            "FROM voice_agent_turns "
            f"WHERE session_id='{STATE['session_id']}' "
            "ORDER BY sequence ASC"
        )
        rows = [line for line in raw.splitlines() if line]
        return rows if len(rows) >= 3 else False

    rows = wait_for("durable Voice Agent conversation turns", turns)
    parsed = [row.split("|", 4) for row in rows]
    if [item[0] for item in parsed[:3]] != ["1", "2", "3"]:
        raise AcceptanceError(f"turn sequence is not monotonic: {rows}")
    if parsed[0][1] != "user" or parsed[0][2] != "Please send digit five.":
        raise AcceptanceError(f"unexpected user turn: {parsed[0]}")
    if parsed[1][1] != "tool" or parsed[1][3] != "send_dtmf" or parsed[1][4] != "tool-call-1":
        raise AcceptanceError(f"unexpected tool turn: {parsed[1]}")
    if parsed[2][1] != "assistant" or parsed[2][2] != "I sent digit five.":
        raise AcceptanceError(f"unexpected assistant turn: {parsed[2]}")


def verify_worker_restart_during_live_call():
    before = psql(
        "SELECT COALESCE(MAX(sequence), 0) FROM voice_agent_turns "
        f"WHERE session_id='{STATE['session_id']}'"
    )
    compose("restart", "worker")
    wait_for(
        "worker recovery readiness",
        lambda: compose("exec", "-T", "worker", "wget", "-qO-", "http://127.0.0.1:8081/readyz", check=False) == "ready",
    )
    cert_dir = os.environ["VOICE_AGENT_V1_CERT_DIR"]
    context = ssl.create_default_context(cafile=os.path.join(cert_dir, "ca.crt"))
    with urllib.request.urlopen("https://127.0.0.1:18444/next-turn", context=context, timeout=5) as response:
        response.read()
    wait_for("tool result after live worker restart", lambda: fake_openai_state().get("tool_results", 0) == 2)
    wait_for("assistant turn after live worker restart", lambda: psql(
        "SELECT COUNT(*) FROM voice_agent_turns "
        f"WHERE session_id='{STATE['session_id']}' AND role='assistant' AND sequence>{int(before)}"
    ) == "1")
    if psql(f"SELECT COUNT(*) FROM voice_agent_sessions WHERE call_id='{STATE['call_id']}'") != "1":
        raise AcceptanceError("worker restart created another durable session")
    if fake_openai_state().get("session_updates") != 1:
        raise AcceptanceError("worker restart replaced the live provider session")
    verify_live_media_attachment()


def hangup_and_verify_completion():
    api("POST", f"/v1/calls/{STATE['call_id']}/hangup", expected={200})

    def call_ended():
        call = get_call(STATE["call_id"])
        return call if call["state"] in {"completed", "cancelled"} else False

    wait_for("terminal call state", call_ended)

    def session_ended():
        state = psql(
            "SELECT state FROM voice_agent_sessions "
            f"WHERE id='{STATE['session_id']}'"
        )
        return state if state in {"completed", "cancelled", "failed"} else False

    session_state = wait_for("terminal Voice Agent session", session_ended)
    if session_state != "completed":
        raise AcceptanceError(
            f"Voice Agent session state = {session_state}, want completed"
        )

    summary = psql(
        "SELECT turn_count::text || '|' || interruption_count::text || '|' || "
        "COALESCE(first_response_latency_ms::text, '') || '|' || "
        "COALESCE(avg_turn_latency_ms::text, '') "
        "FROM voice_agent_sessions "
        f"WHERE id='{STATE['session_id']}'"
    )
    turn_count, interruption_count, first_latency, avg_latency = summary.split("|", 3)
    if turn_count != "2":
        raise AcceptanceError(f"turn_count = {turn_count}, want 2")
    if interruption_count != "0":
        raise AcceptanceError(
            f"interruption_count = {interruption_count}, want 0"
        )
    if first_latency == "" or int(first_latency) < 0:
        raise AcceptanceError(
            f"first_response_latency_ms = {first_latency!r}, want non-negative"
        )
    if avg_latency == "" or int(avg_latency) < 0:
        raise AcceptanceError(
            f"avg_turn_latency_ms = {avg_latency!r}, want non-negative"
        )

    owner_key = f"runtime:media:session:{STATE['session_id']}"

    def ownership_released():
        return redis_cli("GET", owner_key).strip() == ""

    wait_for("media session ownership release", ownership_released)


def main():
    setup_carrier()
    print("PASS 01 SIP trunk configured")
    setup_numbers()
    print("PASS 02 BYOC DID and caller identity configured")
    setup_voice_agent()
    print("PASS 03 provider verified, Voice Agent activated, DID bound, and tool secrets verified")
    originate_call()
    print("PASS 04 outbound Voice Agent call reached answered state")
    wait_voice_agent_session()
    print("PASS 05 active Voice Agent revision attached as one durable session")
    verify_media_placement()
    print("PASS 06 Redis placement assigned session ownership to a healthy media node")
    verify_provider_session()
    print("PASS 07 tenant-scoped provider credential reached realtime provider")
    verify_provider_credential_isolation()
    print("PASS 08 provider secret remained encrypted and outside durable snapshots")
    verify_snapshot_immutability()
    print("PASS 09 active call retained immutable durable agent snapshot")
    verify_audio_roundtrip()
    print("PASS 10 audio and realtime tool-result round trip completed")
    verify_durable_realtime_history()
    print("PASS 11 user, tool, and assistant turns persisted durably")
    verify_worker_restart_during_live_call()
    print("PASS 12 live worker restart preserved provider, tools, and conversation history")
    hangup_and_verify_completion()
    print("PASS 13 session completion persisted conversation summary metrics")
    print("Voice Agent v1 realtime release gate passed")


if __name__ == "__main__":
    main()