#!/usr/bin/env python3
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

DID = "15551236001"
state = {"orders": {}, "order_posts": 0, "routing_patches": 0, "voice_in_trunk_id": None}


def resource(resource_id, resource_type, attributes, relationships=None):
    value = {"id": resource_id, "type": resource_type, "attributes": attributes}
    if relationships is not None:
        value["relationships"] = relationships
    return value


class Handler(BaseHTTPRequestHandler):
    def reply(self, status, payload=None):
        body = b"" if payload is None else json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/vnd.api+json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def authorized(self):
        return self.headers.get("Api-Key") == "cloud-managed-test-key"

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path == "/__state":
            self.reply(200, state)
            return
        if not self.authorized():
            self.reply(401, {"errors": [{"title": "unauthorized"}]})
            return
        query = parse_qs(parsed.query)
        if parsed.path == "/v3/countries":
            self.reply(200, {"data": [resource("country-us", "countries", {"iso": "US"})]})
        elif parsed.path == "/v3/available_dids":
            self.reply(200, {
                "data": [resource("available-1", "available_dids", {"number": DID}, {
                    "did_group": {"data": {"id": "group-1", "type": "did_groups"}}
                })],
                "included": [
                    resource("group-1", "did_groups", {}, {
                        "stock_keeping_units": {"data": [{"id": "sku-voice-1", "type": "stock_keeping_units"}]}
                    }),
                    resource("sku-voice-1", "stock_keeping_units", {"channels_included_count": 2}),
                ],
            })
        elif parsed.path == "/v3/orders":
            external = query.get("filter[external_reference_id]", [""])[0]
            order = state["orders"].get(external)
            self.reply(200, {"data": [] if order is None else [order]})
        elif parsed.path == "/v3/dids":
            data = []
            if state["orders"]:
                relationship = None
                if state["voice_in_trunk_id"]:
                    relationship = {"id": state["voice_in_trunk_id"], "type": "voice_in_trunks"}
                data = [resource("did-1", "dids", {"number": DID}, {
                    "voice_in_trunk": {"data": relationship}
                })]
            self.reply(200, {"data": data})
        else:
            self.reply(404, {"errors": [{"title": "not found"}]})

    def do_POST(self):
        if not self.authorized() or self.path != "/v3/orders":
            self.reply(401 if not self.authorized() else 404)
            return
        payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
        external = payload["data"]["attributes"]["external_reference_id"]
        order = resource("order-1", "orders", {
            "reference": "CLOUD-ACCEPTANCE-1",
            "external_reference_id": external,
            "amount": "1.00",
            "status": "completed",
            "description": "Cloud managed acceptance DID",
            "created_at": "2026-09-07T00:00:00Z",
        })
        state["orders"][external] = order
        state["order_posts"] += 1
        self.reply(201, {"data": order})

    def do_PATCH(self):
        parsed = urlparse(self.path)
        if not self.authorized() or parsed.path != "/v3/dids/did-1":
            self.reply(401 if not self.authorized() else 404)
            return
        payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
        trunk = payload["data"]["relationships"]["voice_in_trunk"]["data"]["id"]
        state["voice_in_trunk_id"] = trunk
        state["routing_patches"] += 1
        relationships = {
            "voice_in_trunk": {
                "data": {"id": trunk, "type": "voice_in_trunks"}
            }
        }
        self.reply(200, {
            "data": resource("did-1", "dids", {"number": DID}, relationships)
        })

    def log_message(self, *_):
        pass


ThreadingHTTPServer(("0.0.0.0", 8089), Handler).serve_forever()
