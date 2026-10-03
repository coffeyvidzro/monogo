# Contributing to Monogo

Thanks for contributing to Monogo, the core runtime implementation behind Leamout.

Leamout is a carrier-grade platform for autonomous voice agents. Changes should preserve the separation between the control plane, Agent Runtime, Telephony Runtime, and realtime media path described in `README.md` and the architecture documentation.

## Before you start

For substantial features, architecture changes, or behavior that affects public APIs, telephony, realtime media, security, tenancy, provider contracts, or deployment, open an issue or discussion first when practical.

Small fixes, tests, documentation improvements, and focused refactors can usually go directly to a pull request.

Security vulnerabilities must not be reported publicly. Follow `SECURITY.md` instead.

## Development setup

### Server

The Go server currently targets Go 1.26.6.

From the repository root:

```sh
cd server
go mod download
go test ./...
go vet ./...
go build ./...
```

Useful Make targets are also available from the repository root:

```sh
make test
make test-race
make vet
make build
make sqlc
make compose-validate
```

If your change affects SQL queries or generated database code, run:

```sh
cd server
sqlc generate
```

Commit generated changes when they are part of the source tree.

### Clients

The client workspace uses Bun and Turbo.

Requirements:

- Node.js 24 or newer;
- Bun 1.3.14.

From `clients/`:

```sh
bun install
bun run lint
bun run build
```

Formatting is handled by Biome:

```sh
bun run format
```

If your change only affects one app, use the relevant workspace filter where appropriate.

### Deployment configuration

Validate Docker Compose changes with:

```sh
docker compose --env-file .env.example -f deploy/compose.yaml config --quiet
```

Do not commit real credentials or production environment files.

## Project boundaries

Please keep changes aligned with the current product architecture.

Leamout is focused on autonomous voice-agent infrastructure. The runtime includes:

- SIP and media infrastructure;
- programmable call control;
- realtime voice-agent sessions;
- turn control and interruption handling;
- composable STT, LLM, and TTS orchestration;
- integrated realtime-model support;
- secure tool execution;
- routing and human handoff;
- events, webhooks, diagnostics, and observability.

The project is BYOC-only. Customer carrier relationships, numbers, SIP trunks, PBXs, SBCs, and carrier billing remain customer-owned.

General-purpose messaging, managed-carrier commerce, telecom resale, telecom wallet charging, and payment adapters are outside the current product boundary unless the architecture is deliberately changed first.

## Architecture expectations

Avoid collapsing architectural boundaries for convenience.

### Control plane

The control plane owns durable configuration and management. It should not become part of the live audio hot path.

### Agent Runtime

The Agent Runtime owns conversational execution, tool calls, turn state, interruption handling, human handoff, and per-call agent behavior.

### Telephony Runtime

OpenSIPS, FreeSWITCH, RTPengine, Coturn, SIP trunks, calls, recordings, and routing primitives belong to the telephony boundary.

### Media Runtime

Latency-sensitive audio handling belongs in the realtime path. Do not route raw audio through NATS JetStream or persist audio frames to PostgreSQL as part of normal runtime operation.

## Go conventions

Keep Go code simple and explicit.

- Run `gofmt` on changed Go files.
- Keep package boundaries aligned with product domains.
- Prefer concrete types unless an abstraction is clearly justified.
- Keep HTTP handlers thin and move business behavior into services or runtime components.
- Use repository/database layers for persistence rather than scattering SQL through services.
- Preserve organization scoping on all tenant-owned data access.
- Wrap errors with useful operational context without leaking secrets.
- Avoid introducing process-global provider credentials or tenant-specific configuration.
- Keep realtime paths bounded and cancellation-aware.

When changing database access, use the project's sqlc workflow where applicable.

## Tests

Every behavior change should include appropriate tests when practical.

At minimum, server changes should pass:

```sh
cd server
go test ./...
go vet ./...
go build ./...
```

For concurrency-sensitive or runtime code, also run:

```sh
go test -race ./...
```

Changes to telephony, BYOC behavior, WebRTC, graceful draining, or the end-to-end voice-agent path may also be covered by dedicated acceptance workflows in `.github/workflows/`.

Add regression tests for bug fixes whenever the failure can be reproduced reliably.

## Linting and formatting

The repository uses CI checks for formatting, linting, vulnerability scanning, builds, tests, Docker validation, and acceptance paths.

Before opening a pull request, make sure changed files are formatted and avoid suppressing a linter merely to make CI green unless the suppression is justified and narrowly scoped.

For Go changes, run the configured `golangci-lint` locally when available.

For client changes, run the client lint and build commands from `clients/`.

## Security and secrets

Never commit:

- API keys;
- SIP credentials;
- database passwords;
- access tokens;
- private keys;
- webhook secrets;
- customer recordings or production call data;
- production `.env` files.

Do not place secrets in logs, events, provider snapshots, test fixtures, screenshots, or pull-request descriptions.

Security-sensitive changes should consider tenant isolation, credential scope, authorization, replay protection, tool execution, SIP abuse, media-session isolation, and failure behavior.

See `SECURITY.md` for vulnerability reporting.

## Database changes

Database migrations should be focused, reviewable, and forward-moving.

When modifying schema or sqlc queries:

1. add or update the migration;
2. update the corresponding sqlc query files;
3. run `sqlc generate`;
4. update repository/service code;
5. add or update tests;
6. verify generated code is committed when required.

Avoid unrelated schema cleanup in the same pull request unless it is required for the change.

## Pull requests

Keep pull requests focused enough to review confidently.

A good pull request should include:

- what changed;
- why the change is needed;
- important architectural or security implications;
- tests and validation performed;
- migration or deployment considerations, when applicable.

Prefer small, coherent commits and descriptive commit messages such as:

```text
feat(media): add bounded realtime control queue
fix(providers): scope credentials by organization
refactor(telephony): simplify SIP trunk routing
test(agent): cover tool execution timeout
docs: clarify BYOC architecture
```

Do not mix large formatting sweeps with functional changes unless the formatting change is the purpose of the pull request.

## Documentation

Update documentation when a change alters:

- public APIs;
- runtime architecture;
- provider contracts;
- environment or deployment requirements;
- security assumptions;
- SIP or media behavior;
- operational procedures.

The canonical architecture documents live under `docs/`.

## Review principles

Reviews should prioritize:

- correctness;
- security and tenant isolation;
- realtime reliability and bounded resource usage;
- architectural consistency;
- operational clarity;
- testability;
- maintainability.

A change that passes tests but weakens runtime isolation, leaks secrets, moves control-plane work into the audio hot path, or introduces unnecessary complexity should be reconsidered before merge.
