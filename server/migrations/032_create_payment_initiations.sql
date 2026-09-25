-- An external provider request can be sent at most once by the synchronous
-- API. An ambiguous response must be reconciled, never blindly resubmitted.
CREATE TABLE payment_initiations (
    payment_id UUID PRIMARY KEY REFERENCES payments(id) ON DELETE RESTRICT,
    checkout_id UUID NOT NULL REFERENCES checkouts(id) ON DELETE RESTRICT,
    request_hash TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'submitting',
    provider_reference TEXT,
    client_secret TEXT,
    display_text TEXT,
    upstream_status TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_initiation_hash CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_initiation_state CHECK (state IN ('submitting', 'submitted', 'uncertain', 'resolved')),
    CONSTRAINT chk_initiation_reference CHECK (provider_reference IS NULL OR length(btrim(provider_reference)) > 0),
    CONSTRAINT chk_initiation_submitted CHECK (
        state <> 'submitted' OR (provider_reference IS NOT NULL AND upstream_status IS NOT NULL)
    )
);

-- Do not open two chargeable sessions for the same checkout, even if callers
-- supply different providers or attempt keys.
CREATE UNIQUE INDEX uq_payment_initiations_active_checkout
    ON payment_initiations (checkout_id)
    WHERE state IN ('submitting', 'submitted', 'uncertain');

CREATE TRIGGER set_payment_initiations_updated_at
BEFORE UPDATE ON payment_initiations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
