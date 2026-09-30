-- Enforce the v1 commercial invariant:
-- Stripe card payments in USD only.
--
-- Fail instead of rewriting financial history when incompatible rows exist.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM subscription_plans WHERE currency <> 'USD'
    ) OR EXISTS (
        SELECT 1 FROM subscriptions WHERE currency <> 'USD'
    ) OR EXISTS (
        SELECT 1 FROM voice_rates WHERE currency <> 'USD'
    ) OR EXISTS (
        SELECT 1 FROM product_rates WHERE currency <> 'USD'
    ) OR EXISTS (
        SELECT 1 FROM number_renewals
        WHERE currency IS NOT NULL AND currency <> 'USD'
    ) OR EXISTS (
        SELECT 1 FROM checkouts WHERE currency <> 'USD'
    ) OR EXISTS (
        SELECT 1 FROM payments WHERE currency <> 'USD'
    ) THEN
        RAISE EXCEPTION
            'cannot enforce USD-only commercial model while non-USD rows exist'
            USING ERRCODE = '23514';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM checkouts
        WHERE provider IS NOT NULL
          AND (
              provider <> 'stripe'
              OR payment_method IS DISTINCT FROM 'card'
          )
    ) OR EXISTS (
        SELECT 1
        FROM payments
        WHERE provider <> 'stripe'
           OR payment_method <> 'card'
    ) OR EXISTS (
        SELECT 1
        FROM payment_provider_events
        WHERE provider <> 'stripe'
    ) THEN
        RAISE EXCEPTION
            'cannot enforce Stripe-only payments while non-Stripe payment rows exist'
            USING ERRCODE = '23514';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM checkouts
        WHERE next_action NOT IN ('none', 'wait')
    ) THEN
        RAISE EXCEPTION
            'cannot remove checkout continuation actions while legacy actions exist'
            USING ERRCODE = '23514';
    END IF;
END;
$$;

ALTER TABLE subscription_plans
    DROP CONSTRAINT chk_subscription_plans_currency,
    ADD CONSTRAINT chk_subscription_plans_currency
        CHECK (currency = 'USD');

ALTER TABLE subscriptions
    DROP CONSTRAINT chk_subscriptions_currency,
    ADD CONSTRAINT chk_subscriptions_currency
        CHECK (currency = 'USD');

ALTER TABLE voice_rates
    DROP CONSTRAINT chk_voice_rates_currency,
    ADD CONSTRAINT chk_voice_rates_currency
        CHECK (currency = 'USD');

ALTER TABLE product_rates
    DROP CONSTRAINT chk_product_rates_currency,
    ADD CONSTRAINT chk_product_rates_currency
        CHECK (currency = 'USD');

ALTER TABLE number_renewals
    DROP CONSTRAINT chk_number_renewals_currency,
    ADD CONSTRAINT chk_number_renewals_currency
        CHECK (currency IS NULL OR currency = 'USD');

ALTER TABLE checkouts
    DROP CONSTRAINT chk_checkouts_currency,
    DROP CONSTRAINT chk_checkouts_payment_binding,
    DROP CONSTRAINT chk_checkouts_action,
    ADD CONSTRAINT chk_checkouts_currency
        CHECK (currency = 'USD'),
    ADD CONSTRAINT chk_checkouts_payment_binding
        CHECK (
            (provider IS NULL AND payment_method IS NULL)
            OR (
                provider = 'stripe'
                AND payment_method = 'card'
            )
        ),
    ADD CONSTRAINT chk_checkouts_action
        CHECK (next_action IN ('none', 'wait'));

ALTER TABLE payments
    DROP CONSTRAINT chk_payments_provider,
    DROP CONSTRAINT chk_payments_payment_method,
    DROP CONSTRAINT chk_payments_currency,
    ADD CONSTRAINT chk_payments_provider
        CHECK (provider = 'stripe'),
    ADD CONSTRAINT chk_payments_payment_method
        CHECK (payment_method = 'card'),
    ADD CONSTRAINT chk_payments_currency
        CHECK (currency = 'USD');

ALTER TABLE payment_provider_events
    DROP CONSTRAINT chk_payment_provider_events_provider,
    ADD CONSTRAINT chk_payment_provider_events_provider
        CHECK (provider = 'stripe');
