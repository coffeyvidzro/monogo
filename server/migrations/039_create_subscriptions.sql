-- Subscription plans are the monthly platform fee and are separate from
-- prepaid managed-carrier usage.

CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    currency CHAR(3) NOT NULL,
    interval TEXT NOT NULL DEFAULT 'month',
    amount_micros BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_subscription_plans_code
        CHECK (code ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT chk_subscription_plans_name
        CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    CONSTRAINT chk_subscription_plans_currency
        CHECK (currency = 'USD'),
    CONSTRAINT chk_subscription_plans_interval
        CHECK (interval = 'month'),
    CONSTRAINT chk_subscription_plans_amount
        CHECK (amount_micros >= 0),
    CONSTRAINT chk_subscription_plans_status
        CHECK (status IN ('active', 'archived'))
);

CREATE TRIGGER set_subscription_plans_updated_at
BEFORE UPDATE ON subscription_plans
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'pending',
    currency CHAR(3) NOT NULL,
    amount_micros BIGINT NOT NULL,
    interval TEXT NOT NULL DEFAULT 'month',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
    started_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_subscriptions_status
        CHECK (status IN ('pending', 'active', 'past_due', 'cancelled')),
    CONSTRAINT chk_subscriptions_currency
        CHECK (currency = 'USD'),
    CONSTRAINT chk_subscriptions_amount
        CHECK (amount_micros >= 0),
    CONSTRAINT chk_subscriptions_interval
        CHECK (interval = 'month'),
    CONSTRAINT chk_subscriptions_period
        CHECK (
            (current_period_start IS NULL AND current_period_end IS NULL)
            OR
            (
                current_period_start IS NOT NULL
                AND current_period_end IS NOT NULL
                AND current_period_end > current_period_start
            )
        ),
    CONSTRAINT chk_subscriptions_started
        CHECK (
            status NOT IN ('active', 'past_due')
            OR (
                started_at IS NOT NULL
                AND current_period_start IS NOT NULL
                AND current_period_end IS NOT NULL
            )
        ),
    CONSTRAINT chk_subscriptions_cancelled
        CHECK (
            (status = 'cancelled' AND cancelled_at IS NOT NULL)
            OR
            (status <> 'cancelled' AND cancelled_at IS NULL)
        ),
    CONSTRAINT chk_subscriptions_cancel_at_period_end
        CHECK (
            NOT cancel_at_period_end
            OR status IN ('active', 'past_due')
        )
);

CREATE UNIQUE INDEX uq_subscriptions_current_organization
    ON subscriptions (organization_id)
    WHERE status IN ('pending', 'active', 'past_due');

CREATE INDEX idx_subscriptions_plan_status
    ON subscriptions (plan_id, status, created_at DESC);

CREATE TRIGGER set_subscriptions_updated_at
BEFORE UPDATE ON subscriptions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
