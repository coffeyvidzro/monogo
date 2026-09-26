#!/usr/bin/env python3
import json
import subprocess
import time
import urllib.request
import uuid


COMPOSE = ["docker", "compose", "-f", "tests/media-v1/compose.yaml"]
CONTROL_TOKEN = "media-v1-control-token-at-least-32-bytes"


def fs(command: str) -> str:
    result = subprocess.run(
        COMPOSE + ["exec", "-T", "freeswitch", "fs_cli", "-H", "127.0.0.1", "-P", "8021",
                   "-p", "media-v1-esl-password", "-x", command],
        check=True, capture_output=True, text=True,
    )
    return result.stdout.strip()


def create_session(channel_id: uuid.UUID) -> str:
    body = json.dumps({
        "id": str(uuid.uuid4()),
        "organization_id": str(uuid.uuid4()),
        "call_id": str(uuid.uuid4()),
        "channel_id": str(channel_id),
        "engine": "echo",
        "input_format": {"sample_rate_hz": 16000, "channels": 1},
        "output_format": {"sample_rate_hz": 16000, "channels": 1},
    }).encode()
    request = urllib.request.Request(
        "http://127.0.0.1:18090/internal/v1/sessions", data=body, method="POST",
        headers={"Authorization": f"Bearer {CONTROL_TOKEN}", "Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=5) as response:
        return json.load(response)["websocket_url"]


def main() -> None:
    channel_id = uuid.uuid4()
    websocket_url = create_session(channel_id)
    response = fs(f"originate {{origination_uuid={channel_id}}}loopback/9196/leamout &park()")
    if "+OK" not in response:
        raise RuntimeError(f"originate failed: {response}")
    try:
        response = fs(f"uuid_audio_fork {channel_id} start {websocket_url} mono 16k")
        if "+OK" not in response:
            raise RuntimeError(f"audio fork failed: {response}")
        fs(f"uuid_broadcast {channel_id} tone_stream://%(2000,0,440) aleg")
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            status = json.loads(fs("audio_fork status"))
            if status.get("sent_bytes", 0) > 0 and status.get("playback_bytes_played", 0) > 0:
                print(json.dumps(status, indent=2, sort_keys=True))
                return
            time.sleep(0.25)
        raise RuntimeError(f"audio did not complete a round trip: {status}")
    finally:
        fs(f"uuid_audio_fork {channel_id} stop")
        fs(f"uuid_kill {channel_id}")


if __name__ == "__main__":
    main()
