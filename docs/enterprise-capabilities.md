# Enterprise capability boundaries

Enterprise is optional commercial packaging for Leamout Self-Hosted or
Leamout Cloud. It is not a deployment model, separate control plane, or runtime
architecture. Enterprise capabilities use the same organization, application,
Media Runtime, and Agent Runtime boundaries as every other Leamout deployment.

## Package ownership

Enterprise-capable product work belongs to the domain that owns the behavior:

```text
server/internal/
├── security/
│   ├── sso/             enterprise identity-provider connections
│   ├── scim/            directory provisioning for users and groups
│   └── authz/           roles, permissions, scopes, and authorization policy
└── platform/
    ├── entitlements/    organization capability availability
    ├── retention/       organization retention policy and cleanup jobs
    └── networking/      organization-level network access configuration
```

Packages are added when their behavior is implemented; empty scaffolding is
not architecture. A package adds only the files required by its active
responsibilities:

| File | Add it when |
| --- | --- |
| `model.go` | The module defines domain models, states, errors, commands, inputs, or outputs. |
| `repository.go` | The module owns durable persistence or database queries. |
| `service.go` | The module contains business rules, use cases, orchestration, or transaction boundaries. |
| `validation.go` | The module has reusable domain or input validation. |
| `handler.go` | The module exposes HTTP endpoints. |
| `routes.go` | The module registers HTTP routes, normally together with `handler.go`. |
| `consumer.go` | Events enter the module asynchronously. |
| `publisher.go` | Events leave the module asynchronously. |
| `jobs.go` | The module owns scheduled or recurring background work. |

Existing packages do not need mechanical file renames when their current files
already express these responsibilities clearly. In particular,
`security/authz` remains the shared authorization domain rather than an
Enterprise-only duplicate.

## Entitlements

`platform/entitlements` answers whether an organization may use an optional
capability. Initial capability identifiers are SSO, SCIM, advanced RBAC,
retention policies, and private networking.

An entitlement is organization scoped and deployment neutral:

```text
Organization
    |
    +-- capability entitlement
            |
            +-- security/sso
            +-- security/scim
            +-- security/authz
            +-- platform/retention
            +-- platform/networking
```

An entitlement is not a subscription, invoice, pricing tier, deployment mode,
or runtime-fleet assignment. Commercial systems may grant or revoke it, but
feature modules consume only the capability decision. Self-Hosted and Cloud
therefore use the same checks and domain model.

Entitlements default to disabled when no organization row exists. This avoids
accidentally enabling commercial or security-sensitive behavior. Capability
modules must still perform their own authorization and tenant-scope checks;
an entitlement never replaces authorization.

## PostgreSQL ownership

Enterprise-capability state remains ordinary organization-scoped application
state in PostgreSQL:

```text
PostgreSQL
├── Enterprise access
│   ├── entitlements
│   ├── sso_connections
│   ├── scim_tokens
│   └── scim_identities
├── Retention
│   └── retention_policies
└── Networking
    └── network_policies
```

- `entitlements` records which optional capabilities an organization may use.
- `sso_connections` stores organization SAML or OIDC configuration. Secret
  material is ciphertext; list queries intentionally omit it.
- `scim_tokens` stores only hashes of credentials presented by external SCIM
  clients. Plaintext bearer tokens are never persisted.
- `scim_identities` maps external directory identifiers to Leamout users in
  the same organization.
- `retention_policies` stores per-resource retention periods for an
  organization.
- `network_policies` stores organization-level source-network access rules.

All normal reads and writes are scoped by `organization_id`. Authentication by
token hash is the deliberate exception for locating a SCIM principal; the
resolved token still carries its owning organization into subsequent access.
These tables contain configuration and access state, not subscription billing
or a parallel Enterprise control plane.

## Capability modules

### `security/sso`

Owns organization identity-provider connections and SAML/OIDC login behavior.
Provider secrets and signing material must be encrypted and tenant scoped.

### `security/scim`

Owns directory provisioning for organization users and groups. SCIM bearer
credentials belong to this security boundary and must not be exposed by normal
API responses.

### `security/authz`

Owns roles, permissions, credential scopes, policies, and request guards. The
existing package is shared across all product packaging; advanced RBAC extends
it without creating a parallel Enterprise authorization stack.

### `platform/retention`

Owns organization retention policies and scheduled cleanup work. Retention
deletion must respect existing recording-storage ownership and tenant
isolation.

### `platform/networking`

Owns organization-level network access configuration. It must not introduce a
private runtime fleet, runtime registration, or deployment-mode branching.

Active policies are enforced after authentication and organization resolution.
A matching deny rule takes precedence over every allow rule. When at least one
active allow rule exists, unmatched addresses are denied; with deny-only or no
active rules, unmatched addresses are allowed. Disabled rules are ignored.

The direct socket peer is the request source unless it belongs to an
operator-configured `TRUSTED_PROXY_CIDRS` range. Only then may Leamout walk the
`X-Forwarded-For` chain from right to left to find the first untrusted client
address. Forwarding headers from untrusted peers are ignored.
