-- Preserve the current active emergency mapping while a replacement is validated.
-- Permit at most one active mapping and at most one in-flight replacement.
DROP INDEX uq_emergency_registrations_current;

CREATE UNIQUE INDEX uq_emergency_registrations_active
    ON emergency_registrations (phone_number_id)
    WHERE status = 'active';

CREATE UNIQUE INDEX uq_emergency_registrations_pending
    ON emergency_registrations (phone_number_id)
    WHERE status IN ('pending', 'validating');
