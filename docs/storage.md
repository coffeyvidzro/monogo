# Recording Storage

Leamout supports managed recording storage and customer-controlled S3-compatible storage without changing the recording API.

## Deployment model

```text
Leamout Cloud
├── Default: Leamout-managed recording storage
│   └── shared MinIO / S3-compatible object storage
│       └── organization prefix isolation
│
└── Optional: BYOS (Bring Your Own Storage)
    └── customer's own S3-compatible storage

Self-Hosted
├── Default: bundled local MinIO
└── Optional: external S3-compatible storage
```

The multi-tenant isolation rules in this document primarily protect Leamout Cloud's default managed storage path, where unrelated organizations share the same object-storage infrastructure.

BYOS remains organization-owned and credential-isolated through the configured storage integration. Self-hosted deployments normally run inside a single customer's trust boundary, while retaining organization namespacing for consistency.

## Deployment endpoint policy

Leamout distinguishes Cloud and Self-Hosted storage networking with `DEPLOYMENT_MODE`.

```text
DEPLOYMENT_MODE=cloud
    → public HTTPS S3-compatible endpoints only

DEPLOYMENT_MODE=self-hosted
    → private/internal HTTP or HTTPS S3-compatible endpoints allowed
```

`cloud` is the secure default.

Cloud keeps public-network endpoint validation and outbound DNS/IP filtering so a tenant cannot turn the storage integration feature into an SSRF path into Leamout infrastructure.

Self-hosted operators may intentionally point Leamout at infrastructure such as:

```text
http://10.0.0.25:9000
https://minio.internal.example
http://localhost:9000
```

That relaxed network policy only applies when the deployment is explicitly configured as self-hosted.

## Managed storage namespace

Objects written to Leamout-managed storage use a server-generated organization namespace:

```text
organizations/{organization_id}/recordings/{yyyy}/{mm}/{dd}/{recording_id}.{ext}
```

Example:

```text
organizations/0c5e2f9d-7ab0-44f5-9761-c6808a70db7f/
└── recordings/
    └── 2026/
        └── 10/
            └── 02/
                └── 8a42d704-50a4-4514-b244-d25c332c6b48.wav
```

The client does not choose this object key. Leamout derives it from trusted recording state.

## Isolation boundary

Managed storage access follows this chain:

```text
authenticated organization
        ↓
tenant-scoped recording lookup
        ↓
recording.organization_id
        ↓
server-generated / persisted storage key
        ↓
validate organizations/{organization_id}/recordings/ prefix
        ↓
managed object store
```

Playback and deletion must reject a managed-storage key that does not belong to the recording organization's namespace, even if malformed or cross-tenant metadata reaches the storage layer.

The organization prefix is defense in depth. It does not replace tenant-scoped database authorization.

## Database ownership

Every recording belongs to an organization.

Every customer storage integration also belongs to an organization.

Recordings use a tenant-scoped composite foreign key so a recording cannot reference a storage integration owned by another organization:

```text
recordings (
    organization_id,
    storage_integration_id
)
        ↓
storage_integrations (
    organization_id,
    id
)
```

This relationship is enforced by the database, not only application code.

## Leamout-managed storage

When a recording has no custom `storage_integration_id`, Leamout uses the configured managed object store.

For Leamout Cloud this is shared infrastructure, so Leamout enforces organization namespacing before upload, playback URL generation, and deletion.

Customers do not receive credentials to the shared managed object store.

Leamout does not expose a generic API that accepts arbitrary object-store keys or buckets for managed storage operations.

## BYOS

An organization may configure its own S3-compatible recording destination.

The storage integration contains the customer's endpoint, region, bucket, access key, encrypted secret, and addressing mode.

```text
organization
    ↓
storage integration
    ↓
customer credentials
    ↓
customer S3-compatible bucket
```

A recording persists the exact `storage_integration_id` that owns its object so historical playback and deletion continue to resolve the original destination after the organization's active integration changes or is disabled.

Leamout does not silently fall back to managed storage when a configured BYOS operation fails.

## Upload destination pinning

The selected destination is persisted before the object is written.

```text
recording ready for upload
        ↓
resolve active organization storage
        ↓
persist:
  storage_integration_id
  storage_key
  storage_provider
  storage_bucket
        ↓
PUT object
        ↓
complete recording
```

This prevents retry drift.

For example, if Integration A is active when an upload starts, the recording is pinned to A before the object write. If the object write succeeds but the final completion update fails, and the organization later activates Integration B, the retry still uses Integration A and the original object key.

For Leamout-managed storage, a pinned recording has no `storage_integration_id`; the persisted managed key/provider/bucket identify that the destination has already been selected. The worker must not re-resolve a newly active BYOS integration for that retry.

## Self-hosted storage

Self-hosted Leamout uses bundled local MinIO by default.

The same organization-prefixed object layout is retained for predictable storage semantics:

```text
organizations/{organization_id}/recordings/...
```

Operators may replace the bundled destination with their own external S3-compatible storage integration, including private-network MinIO, Ceph, or another compatible object store when `DEPLOYMENT_MODE=self-hosted`.

## Security invariants

Recording storage must preserve these invariants:

- organization identity comes from trusted application state, not an object key supplied by a caller;
- managed object keys are generated by Leamout;
- managed uploads cannot write outside the recording organization's prefix;
- managed playback cannot presign an object outside the recording organization's prefix;
- managed deletion cannot delete an object outside the recording organization's prefix;
- database queries remain tenant-scoped;
- a recording cannot reference another organization's storage integration;
- BYOS secrets remain encrypted at rest and are never returned by the API;
- disabled BYOS integrations remain resolvable for historical recordings;
- configured BYOS failures never silently redirect data into Leamout-managed storage;
- Cloud storage endpoints remain public HTTPS only;
- private/internal storage endpoints are accepted only for explicitly self-hosted deployments;
- an upload destination is pinned before object write and reused on every retry.

## Provider neutrality

The product abstraction is **Leamout-managed storage**, not AWS S3.

Leamout Cloud may use MinIO or another S3-compatible object store behind that abstraction. The multi-tenant contract depends on organization namespacing and trusted server-side storage resolution rather than AWS-specific STS or IAM APIs.

This keeps the storage architecture portable while allowing object-store-specific hardening to be added later without changing the public recording model.
