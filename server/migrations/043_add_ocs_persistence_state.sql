-- Persist the charge sequence and charge result snapshots required to prove
-- ordering and idempotency when Redis Stream entries are redelivered.

ALTER TABLE charges
ADD COLUMN ocs_sequence BIGINT NOT NULL DEFAULT 0,
ADD CONSTRAINT chk_charges_ocs_sequence CHECK (ocs_sequence >= 0);

UPDATE charges AS charge
SET ocs_sequence = sequence.maximum
FROM (
    SELECT charge_id, MAX(charge_sequence) AS maximum
    FROM wallet_events
    WHERE charge_id IS NOT NULL
    GROUP BY charge_id
) AS sequence
WHERE charge.id = sequence.charge_id;

ALTER TABLE wallet_events
ADD COLUMN charge_authorized_after_micros BIGINT,
ADD COLUMN charge_consumed_after_micros BIGINT,
ADD COLUMN charge_reserved_after_micros BIGINT,
ADD COLUMN charge_status TEXT,
ADD CONSTRAINT chk_wallet_events_charge_snapshot
CHECK (
    (
        charge_authorized_after_micros IS NULL
        AND charge_consumed_after_micros IS NULL
        AND charge_reserved_after_micros IS NULL
        AND charge_status IS NULL
    )
    OR
    (
        charge_id IS NOT NULL
        AND charge_authorized_after_micros IS NOT NULL
        AND charge_consumed_after_micros IS NOT NULL
        AND charge_reserved_after_micros IS NOT NULL
        AND charge_status IS NOT NULL
        AND charge_authorized_after_micros >= 0
        AND charge_consumed_after_micros >= 0
        AND charge_reserved_after_micros >= 0
        AND charge_consumed_after_micros + charge_reserved_after_micros
            <= charge_authorized_after_micros
        AND charge_status IN ('active', 'completed', 'failed', 'cancelled')
        AND (
            charge_status = 'active'
            OR charge_reserved_after_micros = 0
        )
    )
);
