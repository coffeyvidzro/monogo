# Server agent instructions

These instructions apply to files under `server/` and extend the repository-wide rules in the root `AGENTS.md`.

## Server role

The Go server codebase owns Leamout's control-plane services, agent/runtime coordination, telephony application logic, realtime media services, persistence, security, integrations, workers, and supporting infrastructure.

Keep product-domain boundaries explicit even when packages live in one Go module.

## Go style

Prefer simple, concrete Go code.

- Prefer concrete types over interfaces unless polymorphism, substitution, or testing genuinely requires an interface.
- Keep interfaces narrow when they are necessary.
- Keep HTTP handlers thin.
- Put validation close to the domain boundary.
- Put durable persistence behind repository/database code.
- Keep service/runtime behavior out of transport handlers.
- Use `context.Context` for cancellation and request/session lifetimes.
- Propagate useful operational context in errors without leaking secrets.
- Keep error strings compatible with Go lint conventions.
- Run `gofmt` on changed Go files.

Do not introduce abstractions only to make code look layered.

## Package ownership

Place behavior in the package that owns the domain concept.

Do not create generic `utils`, `helpers`, or catch-all infrastructure packages when the behavior belongs to a domain package.

Keep provider adapters separate from provider-neutral runtime contracts.

Avoid circular dependencies and cross-package reach-through into unexported implementation details.

## Database and sqlc

Use sqlc for database queries where the project already follows that pattern.

Expected flow:

```text
migration
  -> query file
  -> sqlc generate
  -> repository
  -> service/runtime
  -> tests
```

Do not add ad hoc SQL inside handlers or services when the query belongs in sqlc.

Do not manually edit generated sqlc output.

Preserve tenant isolation in every organization-owned query. `organization_id` checks are part of the security boundary, not optional filtering.

For schema changes:

- keep migrations focused;
- avoid destructive rewrites unless the task explicitly requires them;
- update queries and generated code together;
- update repository/service code in the same change;
- add status or ownership predicates where the domain requires them.

## Realtime and concurrency

Realtime paths must be bounded and cancellation-aware.

Do not introduce:

- unbounded channels;
- unbounded queues;
- goroutines without a clear shutdown path;
- blocking database calls in per-frame media processing;
- synchronous durable event persistence in the live audio hot path;
- large avoidable allocations per audio frame;
- retry loops without timeout, cancellation, and backoff bounds.

Use explicit queue limits, frame limits, provider deadlines, attach/session timeouts, and graceful drain behavior where appropriate.

For concurrency-sensitive changes, run the race detector.

## Agent runtime

The Agent Runtime owns conversational execution, not provider-specific transport details.

Keep:

- turn state;
- interruption/barge-in handling;
- tool execution;
- playback cancellation;
- conversation persistence;
- human handoff;
- session diagnostics

inside the appropriate agent/runtime boundary.

Tool execution must remain authorized, schema-validated, timeout-bounded, cancellation-aware, and auditable.

Do not give model output unrestricted database, shell, network, or infrastructure access.

## Provider integrations

Provider packages translate external protocols into Leamout contracts.

Do not make core runtime code depend directly on Deepgram, Groq, Cartesia, OpenAI, or future provider wire formats unless the feature is explicitly provider-specific.

Credentials must be organization-scoped and encrypted at rest.

Never:

- log decrypted credentials;
- put provider secrets in snapshots or events;
- fall back to process-global tenant credentials;
- disable TLS verification;
- swallow terminal provider errors as retryable failures.

Provider config validation should reject unsupported fields rather than silently accepting unknown configuration.

## Telephony

Treat SIP trunks as Leamout's connectivity primitive.

Keep responsibilities clear:

- OpenSIPS: SIP edge, authentication, routing, policy;
- FreeSWITCH: B2BUA and call/media execution;
- RTPengine: RTP anchoring/media boundary;
- Coturn: STUN/TURN for WebRTC.

Do not reintroduce managed-carrier or carrier-commerce concepts into runtime domain models.

Telephony changes must preserve call ownership, organization scoping, routing safety, admission limits, and deterministic lifecycle cleanup.

## Security

Assume all external input is untrusted.

Validate and authorize:

- organization and resource ownership;
- SIP authentication and source constraints;
- webhook authenticity/replay resistance;
- tool calls;
- provider configuration;
- media/session attachment tokens;
- privileged internal operations.

Never weaken auth or tenant boundaries to satisfy tests.

Do not expose internal-only mutation surfaces as public HTTP endpoints without an explicit requirement.

## Tests

Behavior changes should include focused tests when practical.

At minimum, relevant server changes should pass:

```sh
go test ./...
go vet ./...
go build ./...
```

For concurrency-sensitive code:

```sh
go test -race ./...
```

Run targeted package tests while iterating, then broader validation before completion.

Changes touching BYOC, WebRTC, graceful drain, media, or end-to-end voice-agent behavior may also require the matching acceptance workflow under `.github/workflows/`.

Do not remove or weaken a regression test merely to make a change pass unless the test is demonstrably invalid and the reason is documented.

## Lint and generated code

Respect `.golangci.yml`.

Fix lint findings at the source instead of adding broad suppressions.

Do not commit formatting-only churn outside the requested scope.

Regenerate code when its source changes and commit generated output when the repository tracks it.

## Completion checklist

Before finishing server work:

- run `gofmt` on changed Go files;
- run relevant tests;
- run `go vet ./...` and `go build ./...` when practical;
- run race tests for concurrency-sensitive changes;
- regenerate sqlc output when queries/schema changed;
- verify organization scoping;
- verify no secrets enter logs, events, snapshots, or fixtures;
- update docs when architecture or public behavior changed.
