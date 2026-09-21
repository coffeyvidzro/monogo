# BYOC v1 acceptance gate

This suite validates carrier onboarding and SIP routing independently from programmable media controls.

It verifies the production-seeded generic provider, organization-scoped carrier connections, encrypted/redacted credential management, live source-IP authorization changes, trunk and endpoint provisioning, DID ownership, real inbound and outbound SIP calls, route attribution, disable behavior, and restart persistence.

Run from the repository root:

```sh
sh tests/acceptance/byoc-v1/run.sh
```

Set `BYOC_V1_KEEP_STACK=1` to retain containers after a run.
