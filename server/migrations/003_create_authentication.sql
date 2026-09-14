CREATE TABLE IF NOT EXISTS auth_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier CITEXT NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    state TEXT NOT NULL DEFAULT 'started',
    selected_method TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_auth_transactions_state CHECK (
        state IN (
            'started',
            'otp_sent',
            'otp_verified',
            'password_required',
            'authenticated',
            'expired'
        )
    ),
    CONSTRAINT chk_auth_transactions_method CHECK (
        selected_method IS NULL
        OR selected_method IN ('otp', 'password')
    ),
    CONSTRAINT chk_auth_transactions_expiry CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS auth_transactions_identifier_idx
    ON auth_transactions (identifier);

CREATE INDEX IF NOT EXISTS auth_transactions_user_id_idx
    ON auth_transactions (user_id);

CREATE INDEX IF NOT EXISTS auth_transactions_active_idx
    ON auth_transactions (identifier, expires_at)
    WHERE state NOT IN ('authenticated', 'expired');


CREATE TABLE IF NOT EXISTS auth_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_transaction_id UUID
        REFERENCES auth_transactions(id)
        ON DELETE CASCADE,
    identifier CITEXT NOT NULL,
    secret_hash TEXT NOT NULL,
    purpose TEXT NOT NULL,
    state JSONB NOT NULL DEFAULT '{}'::jsonb,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_auth_challenges_purpose CHECK (
        purpose IN (
            'email_otp',
            'email_verification',
            'password_reset',
            'magic_link'
        )
    ),
    CONSTRAINT chk_auth_challenges_state CHECK (jsonb_typeof(state) = 'object'),
    CONSTRAINT chk_auth_challenges_attempts CHECK (attempts >= 0),
    CONSTRAINT chk_auth_challenges_max_attempts CHECK (max_attempts > 0),
    CONSTRAINT chk_auth_challenges_expiry CHECK (expires_at > created_at),
    CONSTRAINT chk_auth_challenges_consumed CHECK (
        consumed_at IS NULL
        OR consumed_at >= created_at
    )
);

CREATE INDEX IF NOT EXISTS auth_challenges_transaction_idx
    ON auth_challenges (auth_transaction_id);

CREATE INDEX IF NOT EXISTS auth_challenges_identifier_idx
    ON auth_challenges (identifier);

CREATE INDEX IF NOT EXISTS auth_challenges_active_idx
    ON auth_challenges (identifier, purpose, expires_at)
    WHERE consumed_at IS NULL;

CREATE INDEX IF NOT EXISTS auth_challenges_expires_idx
    ON auth_challenges (expires_at)
    WHERE consumed_at IS NULL;


CREATE TRIGGER trg_auth_transactions_set_updated_at
BEFORE UPDATE ON auth_transactions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
