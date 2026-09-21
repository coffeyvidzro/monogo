#!/usr/bin/env python3

import argparse
import json
import ssl
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class EventStore:
    def __init__(self):
        self._lock = threading.Lock()
        self._events = []

    def append(self, event):
        with self._lock:
            self._events.append(event)

    def snapshot(self):
        with self._lock:
            return list(self._events)

    def reset(self):
        with self._lock:
            self._events.clear()


STORE = EventStore()


class Handler(BaseHTTPRequestHandler):
    server_version = "LeamoutVoiceV1Webhook/1.0"

    def log_message(self, fmt, *args):
        return

    def _json(self, status, value):
        body = json.dumps(value).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            self._json(200, {"status": "ok"})
            return
        if self.path == "/events":
            self._json(200, {"events": STORE.snapshot()})
            return
        self._json(404, {"error": "not found"})

    def do_POST(self):
        if self.path == "/reset":
            STORE.reset()
            self._json(204, {})
            return
        if self.path != "/events":
            self._json(404, {"error": "not found"})
            return

        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        try:
            envelope = json.loads(body)
        except json.JSONDecodeError:
            self._json(400, {"error": "invalid json"})
            return

        STORE.append({
            "headers": {
                "X-Leamout-Event": self.headers.get("X-Leamout-Event", ""),
                "X-Leamout-Event-ID": self.headers.get("X-Leamout-Event-ID", ""),
                "X-Leamout-Timestamp": self.headers.get("X-Leamout-Timestamp", ""),
                "X-Leamout-Signature": self.headers.get("X-Leamout-Signature", ""),
            },
            "body": body.decode("utf-8"),
            "envelope": envelope,
        })
        self._json(204, {})


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cert", required=True)
    parser.add_argument("--key", required=True)
    parser.add_argument("--port", type=int, default=8443)
    args = parser.parse_args()

    server = ThreadingHTTPServer(("0.0.0.0", args.port), Handler)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    context.load_cert_chain(args.cert, args.key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
