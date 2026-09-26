# Monogo

Monogo is a cloud communications control and media plane. The current product
focus is **Cloud + BYOC** (customers connect their own SIP carrier) and
**Cloud + Managed** (Monogo operates the upstream carrier connection).

The first supported production path is Cloud + BYOC. See the
[Cloud + BYOC operations guide](docs/cloud-byoc.md) for provisioning, network,
authentication, health-check, and acceptance-test procedures.
Recording deployment, retries, playback, deletion, and recovery are covered in
the [recording storage guide](docs/recording-storage.md).
The provider-neutral package boundaries for live AI audio are documented in
the [realtime media-plane guide](docs/media-plane.md).

## Services

- `server`: HTTP control-plane API.
- `worker`: asynchronous jobs, event consumers, reconciliation, and SIP endpoint health checks.
- `media`: low-latency realtime audio and AI provider orchestration.
- `opensips`: public SIP edge and carrier authentication.
- `freeswitch`: call application runtime.
- `rtpengine`: carrier/media boundary.
- PostgreSQL, Redis, and NATS: durable state, admission state, and messaging.

## Validation

```sh
cd server
go test ./...
go vet ./...
go build ./...

cd ..
docker compose --env-file .env.example -f deploy/compose.yaml config --quiet
```

Copy `.env.example` to a protected environment file and replace every example
secret before starting the Compose deployment.
