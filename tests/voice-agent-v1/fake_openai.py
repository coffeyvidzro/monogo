#!/usr/bin/env python3

import argparse
import base64
import hashlib
import json
import math
import ssl
import struct
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

STATE = {
    "connections": 0,
    "session_updates": 0,
    "audio_appends": 0,
    "responses": 0,
    "tool_calls": 0,
    "tool_results": 0,
    "last_tool_result": None,
    "last_session": None,
}
LOCK = threading.Lock()


def tone_pcm():
    samples = 2400
    values = bytearray()
    for index in range(samples):
        value = int(7000 * math.sin(2 * math.pi * 440 * index / 24000))
        values.extend(struct.pack("<h", value))
    return bytes(values)


AUDIO = base64.b64encode(tone_pcm()).decode()


def websocket_accept(key):
    digest = hashlib.sha1(
        (key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()
    ).digest()
    return base64.b64encode(digest).decode()


def read_exact(stream, count):
    data = stream.read(count)
    if len(data) != count:
        raise EOFError("short websocket frame")
    return data


def read_frame(stream):
    head = read_exact(stream, 2)
    opcode = head[0] & 0x0F
    masked = bool(head[1] & 0x80)
    length = head[1] & 0x7F
    if length == 126:
        length = struct.unpack("!H", read_exact(stream, 2))[0]
    elif length == 127:
        length = struct.unpack("!Q", read_exact(stream, 8))[0]
    mask = read_exact(stream, 4) if masked else None
    payload = bytearray(read_exact(stream, length))
    if mask:
        for index in range(length):
            payload[index] ^= mask[index % 4]
    return opcode, bytes(payload)


def write_frame(stream, opcode, payload):
    payload = bytes(payload)
    first = 0x80 | opcode
    if len(payload) < 126:
        head = bytes([first, len(payload)])
    elif len(payload) <= 65535:
        head = bytes([first, 126]) + struct.pack("!H", len(payload))
    else:
        head = bytes([first, 127]) + struct.pack("!Q", len(payload))
    stream.write(head + payload)
    stream.flush()


def write_json(stream, value):
    write_frame(stream, 0x1, json.dumps(value, separators=(",", ":")).encode())


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        return

    def do_GET(self):
        if self.path == "/healthz":
            body = b"ok"
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return

        if self.path == "/state":
            with LOCK:
                body = json.dumps(STATE).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return

        if not self.path.startswith("/v1/realtime"):
            self.send_error(404)
            return

        if self.headers.get("Upgrade", "").lower() != "websocket":
            self.send_error(400)
            return
        if self.headers.get("Authorization") != "Bearer tenant-provider-test-token":
            self.send_error(401)
            return

        key = self.headers.get("Sec-WebSocket-Key")
        if not key:
            self.send_error(400)
            return

        self.send_response(101, "Switching Protocols")
        self.send_header("Upgrade", "websocket")
        self.send_header("Connection", "Upgrade")
        self.send_header("Sec-WebSocket-Accept", websocket_accept(key))
        self.end_headers()

        with LOCK:
            STATE["connections"] += 1

        sent_tool_call = False
        sent_final_response = False
        while True:
            try:
                opcode, payload = read_frame(self.rfile)
            except (EOFError, OSError):
                return

            if opcode == 0x8:
                write_frame(self.wfile, 0x8, b"")
                return
            if opcode == 0x9:
                write_frame(self.wfile, 0xA, payload)
                continue
            if opcode != 0x1:
                continue

            try:
                event = json.loads(payload)
            except json.JSONDecodeError:
                continue

            if event.get("type") == "session.update":
                with LOCK:
                    STATE["session_updates"] += 1
                    STATE["last_session"] = event.get("session")
                continue

            event_type = event.get("type")
            if event_type == "input_audio_buffer.append":
                with LOCK:
                    STATE["audio_appends"] += 1

                if sent_tool_call:
                    continue
                sent_tool_call = True
                write_json(
                    self.wfile,
                    {
                        "type": "input_audio_buffer.speech_started",
                        "event_id": "evt-speech-start",
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "input_audio_buffer.speech_stopped",
                        "event_id": "evt-speech-stop",
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "conversation.item.input_audio_transcription.completed",
                        "event_id": "evt-user-transcript",
                        "transcript": "Please send digit five.",
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "response.created",
                        "event_id": "evt-tool-response-start",
                        "response_id": "resp-tool",
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "response.function_call_arguments.done",
                        "event_id": "evt-tool-call",
                        "response_id": "resp-tool",
                        "call_id": "tool-call-1",
                        "name": "send_dtmf",
                        "arguments": json.dumps({"digits": "5"}),
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "response.done",
                        "event_id": "evt-tool-response-done",
                        "response_id": "resp-tool",
                        "response": {
                            "id": "resp-tool",
                            "status": "completed",
                            "usage": {},
                        },
                    },
                )
                with LOCK:
                    STATE["tool_calls"] += 1
                    STATE["responses"] += 1
                continue

            if event_type == "conversation.item.create":
                item = event.get("item") or {}
                if item.get("type") != "function_call_output":
                    continue
                with LOCK:
                    STATE["tool_results"] += 1
                    STATE["last_tool_result"] = item
                continue

            if event_type == "response.create":
                if sent_final_response:
                    continue
                sent_final_response = True
                write_json(
                    self.wfile,
                    {
                        "type": "response.created",
                        "event_id": "evt-final-response-start",
                        "response_id": "resp-final",
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "response.output_audio_transcript.delta",
                        "event_id": "evt-assistant-transcript",
                        "response_id": "resp-final",
                        "delta": "I sent digit five.",
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "response.audio.delta",
                        "event_id": "evt-audio",
                        "response_id": "resp-final",
                        "delta": AUDIO,
                    },
                )
                write_json(
                    self.wfile,
                    {
                        "type": "response.done",
                        "event_id": "evt-final-response-done",
                        "response_id": "resp-final",
                        "response": {
                            "id": "resp-final",
                            "status": "completed",
                            "usage": {},
                        },
                    },
                )
                with LOCK:
                    STATE["responses"] += 1
                continue


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cert", required=True)
    parser.add_argument("--key", required=True)
    parser.add_argument("--port", type=int, default=8444)
    args = parser.parse_args()

    server = ThreadingHTTPServer(("0.0.0.0", args.port), Handler)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    context.load_cert_chain(args.cert, args.key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
