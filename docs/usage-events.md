# Usage observations (not billing)

Migration 031 defines tenant-scoped `meters` and immutable `usage_events`.
The supplied usage-events schema referred to `meters(id)`, which did not
exist in the parent branch. The minimal meter registry is included in 031
to satisfy that dependency without defining rates or billability.

Usage observations are internal records; they do not charge a prepaid wallet,
create an invoice, or establish a customer's contractual price.

## Invariants

- `(meter_id, organization_id)` is a composite foreign key so an organization
  cannot record usage against another organization's meter.
- `(organization_id, idempotency_key)` is unique. A repeated identical
  observation returns the existing record; a different observation with the
  same key raises a conflict. JSONB equality ignores property ordering.
- Quantities are strictly positive integers; source labels follow the
  lowercase snake-case convention. Dimensions must be a JSON object.
- Records are immutable: corrections require distinct new observations, not
  UPDATE/DELETE of an accepted event.
- `commercial.RegisterRoutes` currently registers **no HTTP endpoints**.
  Ingestion is internal-only until authorization, pricing and event provenance
  are specified. This stack creates no consumer, publisher or scheduled job.

## Integration notes

This PR is stacked on the managed-number branch (#69). It deliberately
does not depend on the acceptance-suite MinIO workaround in the sibling
stacked PR (#72). Regenerate sqlc and the Atlas hash before merging.
