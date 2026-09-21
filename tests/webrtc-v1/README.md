# WebRTC v1 forced-TURN acceptance

This gate proves the self-hosted browser voice path against the real Compose stack.
It provisions a SIP domain and subscriber through the public API, requests short-lived
ICE credentials from `/v1/realtime/ice-credentials`, registers Chromium to OpenSIPS
over WSS, and calls the FreeSWITCH `9196` echo service through RTPengine.

The browser configures `iceTransportPolicy: "relay"` and the test fails unless the
nominated local ICE candidate is a TURN relay and inbound audio bytes are received.
Host and server-reflexive fallback therefore cannot make the test pass.

## Run locally

Install the browser dependencies once:

```sh
npm install --prefix tests/acceptance/webrtc-v1
cd tests/acceptance/webrtc-v1 && npx playwright install chromium && cd ../../..
```

Then run:

```sh
sh tests/acceptance/webrtc-v1/run.sh
```

The runner creates disposable TLS material, selects a small collision-free UDP relay
range for Coturn, starts the required Docker services, applies migrations, provisions
an isolated acceptance organization, and tears the stack down on exit. Set
`WEBRTC_V1_TURN_MIN_PORT` and `WEBRTC_V1_TURN_MAX_PORT` together to use an explicit
relay range instead.

Set `WEBRTC_V1_KEEP_STACK=1` to keep a failed stack available for inspection.
