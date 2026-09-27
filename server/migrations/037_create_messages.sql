CREATE TABLE IF NOT EXISTS messaging_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK (scope IN ('platform', 'organization')),
    channel TEXT NOT NULL CHECK (channel IN ('sms', 'whatsapp')),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
    encrypted_secret TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_messaging_connections_id_organization UNIQUE (id, organization_id),
    CONSTRAINT chk_messaging_connections_scope CHECK (
        (scope = 'platform' AND organization_id IS NULL)
        OR (scope = 'organization' AND organization_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_messaging_connections_routing
    ON messaging_connections (scope, organization_id, channel, created_at) WHERE status = 'active';

CREATE TRIGGER set_messaging_connections_updated_at
BEFORE UPDATE ON messaging_connections
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    messaging_connection_id UUID REFERENCES messaging_connections(id) ON DELETE RESTRICT,

    channel TEXT NOT NULL,
    direction TEXT NOT NULL,
    status TEXT NOT NULL,

    from_address TEXT NOT NULL,
    to_address TEXT NOT NULL,
    body TEXT,
    media JSONB NOT NULL DEFAULT '[]'::jsonb,

    provider_message_id TEXT,
    idempotency_key TEXT,
    request_hash TEXT,

    failure_code TEXT,
    failure_message TEXT,

    queued_at TIMESTAMPTZ,
    submitting_at TIMESTAMPTZ,
    submission_unknown_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_messages_id_organization UNIQUE (id, organization_id),

    CONSTRAINT chk_messages_channel CHECK (
        channel IN ('sms', 'whatsapp')
    ),
    CONSTRAINT chk_messages_direction CHECK (
        direction IN ('inbound', 'outbound')
    ),
    CONSTRAINT chk_messages_status CHECK (
        status IN (
            'queued',
            'submitting',
            'submission_unknown',
            'submitted',
            'sent',
            'delivered',
            'undelivered',
            'received',
            'failed'
        )
    ),

    CONSTRAINT chk_messages_from_address CHECK (
        length(btrim(from_address)) BETWEEN 1 AND 255
    ),
    CONSTRAINT chk_messages_to_address CHECK (
        length(btrim(to_address)) BETWEEN 1 AND 255
    ),
    CONSTRAINT chk_messages_body CHECK (
        body IS NULL OR length(body) <= 10000
    ),
    CONSTRAINT chk_messages_media_array CHECK (
        jsonb_typeof(media) = 'array' AND jsonb_array_length(media) <= 10
    ),
    CONSTRAINT chk_messages_payload CHECK (
        (body IS NOT NULL AND length(btrim(body)) > 0)
        OR jsonb_array_length(media) > 0
    ),

    CONSTRAINT chk_messages_outbound_idempotency CHECK (
        (
            direction = 'outbound'
            AND idempotency_key IS NOT NULL
            AND length(btrim(idempotency_key)) BETWEEN 1 AND 255
            AND request_hash IS NOT NULL
            AND length(request_hash) = 64
        )
        OR (
            direction = 'inbound'
            AND idempotency_key IS NULL
            AND request_hash IS NULL
        )
    ),

    CONSTRAINT chk_messages_provider_identity CHECK (
        provider_message_id IS NULL
        OR (
            messaging_connection_id IS NOT NULL
            AND length(btrim(provider_message_id)) > 0
        )
    ),

    CONSTRAINT chk_messages_direction_status CHECK (
        (direction = 'inbound' AND status = 'received')
        OR
        (
            direction = 'outbound'
            AND status IN (
                'queued',
                'submitting',
                'submission_unknown',
                'submitted',
                'sent',
                'delivered',
                'undelivered',
                'failed'
            )
        )
    ),

    CONSTRAINT chk_messages_lifecycle_timestamps CHECK (
        (direction <> 'outbound' OR queued_at IS NOT NULL)
        AND (direction <> 'inbound' OR received_at IS NOT NULL)
        AND (status <> 'submitting' OR submitting_at IS NOT NULL)
        AND (status <> 'submission_unknown' OR submission_unknown_at IS NOT NULL)
        AND (status <> 'submitted' OR submitted_at IS NOT NULL)
        AND (status <> 'sent' OR sent_at IS NOT NULL)
        AND (status <> 'delivered' OR delivered_at IS NOT NULL)
        AND (status NOT IN ('undelivered', 'failed') OR failed_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_messages_organization_created
    ON messages (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_organization_status
    ON messages (organization_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_organization_direction
    ON messages (organization_id, direction, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_organization_channel
    ON messages (organization_id, channel, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_messaging_connection
    ON messages (messaging_connection_id, created_at DESC)
    WHERE messaging_connection_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_messages_organization_idempotency
    ON messages (organization_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_messages_provider_message
    ON messages (messaging_connection_id, provider_message_id)
    WHERE messaging_connection_id IS NOT NULL
      AND provider_message_id IS NOT NULL;

CREATE TRIGGER set_messages_updated_at
BEFORE UPDATE ON messages
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
