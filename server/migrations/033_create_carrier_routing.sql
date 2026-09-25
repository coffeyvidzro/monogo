-- Rates use the platform billing currency and integer micros so route choices
-- are reproducible and never depend on floating-point money arithmetic.
CREATE TABLE carrier_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE CASCADE,
    destination_prefix TEXT NOT NULL,
    rate_micros BIGINT NOT NULL,
    billing_currency CHAR(3) NOT NULL DEFAULT 'USD',
    effective_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (destination_prefix ~ '^[1-9][0-9]{0,14}$'),
    CHECK (rate_micros >= 0),
    CHECK (billing_currency ~ '^[A-Z]{3}$'),
    CHECK (expires_at IS NULL OR expires_at > effective_at),
    UNIQUE (carrier_connection_id, destination_prefix, effective_at)
);

CREATE INDEX idx_carrier_rates_lookup
    ON carrier_rates (carrier_connection_id, destination_prefix, effective_at DESC);

-- Collectors upsert one current aggregate per endpoint. Basis points represent
-- percentages from 0.00% through 100.00%; durations remain integer millis.
CREATE TABLE carrier_route_metrics (
    trunk_endpoint_id UUID PRIMARY KEY REFERENCES trunk_endpoints(id) ON DELETE CASCADE,
    asr_basis_points INTEGER NOT NULL,
    aloc_milliseconds BIGINT NOT NULL,
    latency_milliseconds INTEGER NOT NULL,
    packet_loss_basis_points INTEGER NOT NULL,
    sample_count BIGINT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (asr_basis_points BETWEEN 0 AND 10000),
    CHECK (aloc_milliseconds >= 0),
    CHECK (latency_milliseconds >= 0),
    CHECK (packet_loss_basis_points BETWEEN 0 AND 10000),
    CHECK (sample_count > 0)
);

CREATE INDEX idx_carrier_route_metrics_observed
    ON carrier_route_metrics (observed_at DESC);

CREATE TRIGGER set_carrier_route_metrics_updated_at
BEFORE UPDATE ON carrier_route_metrics
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- A decision and its immutable candidate snapshots explain exactly why one
-- carrier won without consulting rate or metric rows that may later change.
CREATE TABLE routing_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    destination TEXT NOT NULL,
    selected_carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE RESTRICT,
    selected_trunk_id UUID NOT NULL REFERENCES trunks(id) ON DELETE RESTRICT,
    selected_trunk_endpoint_id UUID NOT NULL REFERENCES trunk_endpoints(id) ON DELETE RESTRICT,
    candidate_count INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, organization_id),
    CHECK (destination ~ '^\+[1-9][0-9]{6,14}$'),
    CHECK (candidate_count > 0)
);

CREATE INDEX idx_routing_decisions_organization_created
    ON routing_decisions (organization_id, created_at DESC);

ALTER TABLE calls
    ADD COLUMN routing_decision_id UUID,
    ADD CONSTRAINT fk_calls_routing_decision_organization
        FOREIGN KEY (routing_decision_id, organization_id)
        REFERENCES routing_decisions(id, organization_id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX uq_calls_routing_decision
    ON calls (routing_decision_id)
    WHERE routing_decision_id IS NOT NULL;

CREATE TABLE routing_decision_candidates (
    routing_decision_id UUID NOT NULL REFERENCES routing_decisions(id) ON DELETE CASCADE,
    rank INTEGER NOT NULL,
    carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE RESTRICT,
    trunk_id UUID NOT NULL REFERENCES trunks(id) ON DELETE RESTRICT,
    trunk_endpoint_id UUID NOT NULL REFERENCES trunk_endpoints(id) ON DELETE RESTRICT,
    rate_micros BIGINT NOT NULL,
    asr_basis_points INTEGER NOT NULL,
    aloc_milliseconds BIGINT NOT NULL,
    latency_milliseconds INTEGER NOT NULL,
    packet_loss_basis_points INTEGER NOT NULL,
    metrics_observed_at TIMESTAMPTZ NOT NULL,
    score_micros BIGINT NOT NULL,
    PRIMARY KEY (routing_decision_id, rank),
    UNIQUE (routing_decision_id, trunk_endpoint_id),
    CHECK (rank > 0),
    CHECK (rate_micros >= 0),
    CHECK (asr_basis_points BETWEEN 0 AND 10000),
    CHECK (aloc_milliseconds >= 0),
    CHECK (latency_milliseconds >= 0),
    CHECK (packet_loss_basis_points BETWEEN 0 AND 10000),
    CHECK (score_micros >= 0)
);
