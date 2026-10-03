# Client agent instructions

These instructions apply to files under `clients/` and extend the repository-wide rules in the root `AGENTS.md`.

## Client role

The client workspace contains Leamout's web-facing applications and shared frontend packages.

Client code should present and operate the existing Leamout product model. Do not invent new backend capabilities, commercial models, or telephony concepts in UI code.

## Tooling

The workspace uses Bun, Turbo, TypeScript, and Biome.

Current baseline:

- Node.js 24 or newer;
- Bun 1.3.14;
- TypeScript 7;
- Biome for formatting/lint-related client workflows;
- Turbo for workspace task orchestration.

Use the workspace scripts rather than bypassing them with ad hoc commands when an existing script covers the task.

Common commands from `clients/`:

```sh
bun install
bun run lint
bun run build
bun run format
```

Use workspace filters for targeted validation when appropriate, then run broader checks before completion when the change is cross-cutting.

## Product consistency

UI copy and flows must match Leamout's current product boundary.

Leamout is a carrier-grade runtime and control plane for autonomous voice agents.

Do not present Leamout as:

- a managed carrier;
- a telecom reseller;
- an SMS or WhatsApp platform;
- a phone-number marketplace;
- a carrier-billing or prepaid-wallet product.

BYOC means customers retain their own carrier, SIP trunk, PBX, SBC, phone-number, and carrier-billing relationships.

If a UI concept conflicts with the backend architecture or `README.md`, do not paper over the mismatch in frontend code. Fix or clarify the product contract instead.

## Component design

Keep components focused and composable.

Prefer:

- explicit props;
- small presentational components;
- deterministic rendering;
- accessible semantic HTML;
- shared primitives for repeated visual patterns;
- data-driven rendering for repeated static structures.

Avoid:

- oversized page components with unrelated responsibilities;
- deeply coupled presentation and data-fetching logic;
- hidden global state for local UI behavior;
- premature component abstractions used only once;
- copying the same markup or validation logic across apps when a shared primitive already exists.

Do not create a shared package solely to avoid a few lines of duplication unless there is a stable cross-app contract.

## TypeScript

Keep types explicit at public component and data boundaries.

Do not use `any` to silence type errors when the underlying shape can be modeled.

Prefer discriminated unions or narrow domain types over strings with implicit meaning when the UI has multiple states.

Do not duplicate backend domain enums manually if the project already exposes a shared/generated contract that should be used instead.

Handle nullable and loading states explicitly rather than relying on unsafe assertions.

## Server/client boundaries

Do not move sensitive operations into the browser.

Never expose:

- provider API keys;
- SIP credentials;
- private tokens;
- internal service credentials;
- decrypted secrets;
- privileged internal API operations.

Treat all browser-visible configuration as public information.

Authorization must be enforced by the server even if the UI hides or disables an action.

Do not rely on client-side validation as the security boundary.

## Data fetching and mutations

Keep request behavior explicit.

- Surface loading, empty, success, and failure states where relevant.
- Do not silently swallow API errors.
- Avoid aggressive retry loops for mutations.
- Preserve idempotency semantics when the backend expects them.
- Do not fabricate backend state optimistically when failure would create misleading telephony or runtime status.

For realtime session or call state, distinguish authoritative server/runtime state from temporary UI state.

## Accessibility

Interactive elements must be keyboard accessible and use appropriate semantic elements.

Provide accessible names for icon-only controls.

Do not use clickable `div` elements where a button or link is appropriate.

Maintain visible focus behavior and sensible heading structure.

Decorative visuals should not create unnecessary screen-reader noise.

## Styling

Follow the existing app's design system and layout conventions.

Do not introduce an unrelated design language for a single feature.

Keep responsive behavior intentional across desktop and mobile sizes.

Avoid hard-coded visual values when an existing design token or shared primitive should be used.

Do not make large repository-wide styling or formatting sweeps as part of an unrelated feature.

## Marketing and product copy

Keep public-facing language precise.

Preferred concepts include:

- autonomous voice agents;
- carrier-grade runtime;
- control plane;
- BYOC;
- SIP-native connectivity;
- realtime orchestration;
- self-hosted and Leamout Cloud where applicable.

Do not make unsupported claims about latency, uptime, compliance, carrier coverage, pricing, or production readiness.

Do not describe planned functionality as generally available unless the task explicitly updates product status.

## Testing and validation

For client changes, run the checks relevant to the affected workspace.

Baseline from `clients/`:

```sh
bun run lint
bun run build
```

Run formatting when files require it:

```sh
bun run format
```

For a single application, use the relevant Turbo/Bun workspace filter when practical.

For UI changes, verify the important responsive and interaction states rather than checking only the default desktop render.

Do not claim a build, lint, or visual check passed unless it was actually run or CI confirms it.

## Scope discipline

Keep frontend changes within the requested surface.

Do not rewrite shared components, navigation, styling, or unrelated pages merely because they could be improved.

If backend support is missing for a requested UI feature, do not invent a fake client-only implementation. Call out the dependency or implement the necessary backend contract when it is part of the task.

## Completion checklist

Before finishing client work:

- confirm copy matches the current Leamout product boundary;
- confirm no secret or privileged operation is exposed to the browser;
- run relevant lint/build checks;
- format changed files using the repository tooling;
- verify loading, empty, error, and success states when applicable;
- verify accessibility for changed interactive controls;
- verify responsive behavior for meaningful UI changes;
- update documentation when public behavior or product terminology changed.
