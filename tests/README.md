# Acceptance tests

These acceptance suites exercise implemented functionality in the current
Leamout runtime. Cloud + Managed number purchasing and provider provisioning
are not implemented and are not acceptance requirements.

## BYOC

- `byoc-v1`: customer-owned carrier onboarding and SIP routing.

## Core voice architecture

- `voice-v1`: programmable voice across OpenSIPS, FreeSWITCH, RTPengine, and
  the control plane.
- `webrtc-v1`: browser calling and TURN/ICE media relay through the core voice
  stack.
- `graceful-drain`: draining active calls across core signaling and media
  services.

The architecture suites exercise existing runtime capabilities; they do not
establish availability of a Cloud + Managed product model.
