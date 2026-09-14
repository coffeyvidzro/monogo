CREATE TABLE IF NOT EXISTS voice_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    ring_timeout_seconds INTEGER NOT NULL DEFAULT 30,
    caller_id TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    voice_url TEXT,
    callback_url TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_voice_applications_organization_name UNIQUE (organization_id, name),
    CONSTRAINT chk_voice_applications_name CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_voice_applications_ring_timeout CHECK (ring_timeout_seconds BETWEEN 1 AND 300),
    CONSTRAINT chk_voice_applications_caller_id CHECK (
        caller_id IS NULL OR length(trim(caller_id)) > 0
    ),
    CONSTRAINT chk_voice_applications_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_voice_applications_voice_url CHECK (
        voice_url IS NULL OR voice_url ~ '^https?://'
    ),
    CONSTRAINT chk_voice_applications_callback_url CHECK (
        callback_url IS NULL OR callback_url ~ '^https?://'
    )
);

CREATE INDEX IF NOT EXISTS idx_voice_applications_organization_status
    ON voice_applications (organization_id, status);

CREATE TABLE IF NOT EXISTS voice_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voice_application_id UUID NOT NULL
        REFERENCES voice_applications(id) ON DELETE CASCADE,

    -- Exactly one target may be bound.
    phone_number_id UUID REFERENCES phone_numbers(id) ON DELETE CASCADE,
    sip_domain_id UUID REFERENCES sip_domains(id) ON DELETE CASCADE,
    subscriber_id UUID REFERENCES subscribers(id) ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_voice_bindings_single_target CHECK (
        (phone_number_id IS NOT NULL)::int
        + (sip_domain_id IS NOT NULL)::int
        + (subscriber_id IS NOT NULL)::int = 1
    )
);

-- A DID may route to only one voice application. Applications may still own
-- many DIDs through separate bindings.
CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_bindings_phone_number
    ON voice_bindings (phone_number_id)
    WHERE phone_number_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_bindings_application_sip_domain
    ON voice_bindings (voice_application_id, sip_domain_id)
    WHERE sip_domain_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_bindings_application_subscriber
    ON voice_bindings (voice_application_id, subscriber_id)
    WHERE subscriber_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_voice_bindings_sip_domain
    ON voice_bindings (sip_domain_id)
    WHERE sip_domain_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_voice_bindings_subscriber
    ON voice_bindings (subscriber_id)
    WHERE subscriber_id IS NOT NULL;

CREATE TRIGGER set_voice_applications_updated_at
BEFORE UPDATE ON voice_applications
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
