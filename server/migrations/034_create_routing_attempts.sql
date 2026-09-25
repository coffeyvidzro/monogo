-- Every actual carrier attempt is retained separately from the immutable route
-- plan. At most three attempts are allowed for one decision.
CREATE TABLE routing_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    routing_decision_id UUID NOT NULL REFERENCES routing_decisions(id) ON DELETE RESTRICT,
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE RESTRICT,
    attempt INTEGER NOT NULL,
    carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE RESTRICT,
    trunk_id UUID NOT NULL REFERENCES trunks(id) ON DELETE RESTRICT,
    trunk_endpoint_id UUID NOT NULL REFERENCES trunk_endpoints(id) ON DELETE RESTRICT,
    outcome TEXT NOT NULL,
    failure_class TEXT,
    sip_status INTEGER,
    duration_milliseconds BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (routing_decision_id, attempt),
    CHECK (attempt BETWEEN 1 AND 3),
    CHECK (outcome IN ('succeeded', 'retryable_failure', 'terminal_failure')),
    CHECK (failure_class IS NULL OR failure_class IN ('validation', 'transport', 'timeout', 'sip', 'capacity', 'internal')),
    CHECK (sip_status IS NULL OR sip_status BETWEEN 100 AND 699),
    CHECK (duration_milliseconds >= 0),
    CHECK ((outcome = 'succeeded' AND failure_class IS NULL AND sip_status IS NULL)
           OR (outcome <> 'succeeded' AND failure_class IS NOT NULL))
);

CREATE INDEX idx_routing_attempts_call
    ON routing_attempts (call_id, attempt);
CREATE INDEX idx_routing_attempts_outcome_created
    ON routing_attempts (outcome, created_at DESC);
