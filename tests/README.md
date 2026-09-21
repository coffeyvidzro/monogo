# Acceptance tests

The acceptance suites cover both supported product models and the shared Leamout
Core voice architecture.

## Product-model suites

- `byoc-v1`: customer-owned carrier onboarding and SIP routing.
- `cloud-managed`: Leamout-managed number purchasing, routing, and calling.

## Leamout Core architecture suites

- `voice-v1`: programmable voice across OpenSIPS, FreeSWITCH, RTPengine, and the
  control plane.
- `webrtc-v1`: browser calling and TURN/ICE media relay through the core voice
  stack.
- `graceful-drain`: safe draining of active calls across core signaling and
  media services.

These architecture suites are not a separate product or deployment model. They
exercise infrastructure shared by BYOC and Cloud + Managed services.
