-- name: CreateCarrierRate :one
INSERT INTO carrier_rates (
    carrier_connection_id,
    destination_prefix,
    rate_micros,
    billing_currency,
    effective_at,
    expires_at
) VALUES (
    sqlc.arg(carrier_connection_id),
    sqlc.arg(destination_prefix),
    sqlc.arg(rate_micros),
    sqlc.arg(billing_currency),
    sqlc.arg(effective_at),
    sqlc.narg(expires_at)
)
RETURNING *;

-- name: UpsertCarrierRouteMetrics :one
INSERT INTO carrier_route_metrics (
    trunk_endpoint_id,
    asr_basis_points,
    aloc_milliseconds,
    latency_milliseconds,
    packet_loss_basis_points,
    sample_count,
    observed_at
) VALUES (
    sqlc.arg(trunk_endpoint_id),
    sqlc.arg(asr_basis_points),
    sqlc.arg(aloc_milliseconds),
    sqlc.arg(latency_milliseconds),
    sqlc.arg(packet_loss_basis_points),
    sqlc.arg(sample_count),
    sqlc.arg(observed_at)
)
ON CONFLICT (trunk_endpoint_id) DO UPDATE SET
    asr_basis_points = EXCLUDED.asr_basis_points,
    aloc_milliseconds = EXCLUDED.aloc_milliseconds,
    latency_milliseconds = EXCLUDED.latency_milliseconds,
    packet_loss_basis_points = EXCLUDED.packet_loss_basis_points,
    sample_count = EXCLUDED.sample_count,
    observed_at = EXCLUDED.observed_at,
    updated_at = now()
WHERE carrier_route_metrics.observed_at <= EXCLUDED.observed_at
RETURNING *;

-- name: ListManagedRouteCandidates :many
SELECT
    cc.id AS carrier_connection_id,
    cc.max_cps,
    cc.max_concurrent_calls,
    cc.max_daily_minutes,
    t.id AS trunk_id,
    te.id AS trunk_endpoint_id,
    te.host,
    te.port,
    te.transport,
    te.health_status,
    rate.rate_micros,
    metrics.asr_basis_points,
    metrics.aloc_milliseconds,
    metrics.latency_milliseconds,
    metrics.packet_loss_basis_points,
    metrics.observed_at
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
JOIN trunk_endpoints AS te ON te.trunk_id = t.id
JOIN carrier_route_metrics AS metrics ON metrics.trunk_endpoint_id = te.id
JOIN LATERAL (
    SELECT cr.rate_micros
    FROM carrier_rates AS cr
    WHERE cr.carrier_connection_id = cc.id
      AND cr.billing_currency = 'USD'
      AND sqlc.arg(destination_digits)::TEXT LIKE cr.destination_prefix || '%'
      AND cr.effective_at <= sqlc.arg(resolved_at)
      AND (cr.expires_at IS NULL OR cr.expires_at > sqlc.arg(resolved_at))
    ORDER BY length(cr.destination_prefix) DESC, cr.effective_at DESC
    LIMIT 1
) AS rate ON true
WHERE t.organization_id IS NULL
  AND t.provisioning_mode = 'managed'
  AND t.status = 'active'
  AND t.direction IN ('outbound', 'bidirectional')
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
  AND cc.status = 'active'
  AND te.organization_id IS NULL
  AND te.enabled = true
  AND te.direction IN ('outbound', 'bidirectional');

-- name: CreateRoutingDecision :one
INSERT INTO routing_decisions (
    organization_id,
    destination,
    selected_carrier_connection_id,
    selected_trunk_id,
    selected_trunk_endpoint_id,
    candidate_count
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(destination),
    sqlc.arg(selected_carrier_connection_id),
    sqlc.arg(selected_trunk_id),
    sqlc.arg(selected_trunk_endpoint_id),
    sqlc.arg(candidate_count)
)
RETURNING *;

-- name: CreateRoutingDecisionCandidate :exec
INSERT INTO routing_decision_candidates (
    routing_decision_id,
    rank,
    carrier_connection_id,
    trunk_id,
    trunk_endpoint_id,
    rate_micros,
    asr_basis_points,
    aloc_milliseconds,
    latency_milliseconds,
    packet_loss_basis_points,
    metrics_observed_at,
    score_micros
) VALUES (
    sqlc.arg(routing_decision_id),
    sqlc.arg(rank),
    sqlc.arg(carrier_connection_id),
    sqlc.arg(trunk_id),
    sqlc.arg(trunk_endpoint_id),
    sqlc.arg(rate_micros),
    sqlc.arg(asr_basis_points),
    sqlc.arg(aloc_milliseconds),
    sqlc.arg(latency_milliseconds),
    sqlc.arg(packet_loss_basis_points),
    sqlc.arg(metrics_observed_at),
    sqlc.arg(score_micros)
);

-- name: CreateRoutingAttempt :one
INSERT INTO routing_attempts (
    routing_decision_id,
    call_id,
    attempt,
    carrier_connection_id,
    trunk_id,
    trunk_endpoint_id,
    outcome,
    failure_class,
    sip_status,
    duration_milliseconds
) VALUES (
    sqlc.arg(routing_decision_id),
    sqlc.arg(call_id),
    sqlc.arg(attempt),
    sqlc.arg(carrier_connection_id),
    sqlc.arg(trunk_id),
    sqlc.arg(trunk_endpoint_id),
    sqlc.arg(outcome),
    sqlc.narg(failure_class),
    sqlc.narg(sip_status),
    sqlc.arg(duration_milliseconds)
)
RETURNING *;

-- name: SetRoutingDecisionSelectedRoute :execrows
UPDATE routing_decisions
SET selected_carrier_connection_id = sqlc.arg(carrier_connection_id),
    selected_trunk_id = sqlc.arg(trunk_id),
    selected_trunk_endpoint_id = sqlc.arg(trunk_endpoint_id)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id);
