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

