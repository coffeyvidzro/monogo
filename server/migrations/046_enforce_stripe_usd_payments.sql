-- Stripe card payments in USD are the only v1 funding rail.

ALTER TABLE subscription_plans
    DROP CONSTRAINT chk_subscription_plans_currency,
    ADD CONSTRAINT chk_subscription_plans_currency CHECK (currency = 'USD');

ALTER TABLE subscriptions
    DROP CONSTRAINT chk_subscriptions_currency,
    ADD CONSTRAINT chk_subscriptions_currency CHECK (currency = 'USD');

ALTER TABLE voice_rates
    DROP CONSTRAINT chk_voice_rates_currency,
    ADD CONSTRAINT chk_voice_rates_currency CHECK (currency = 'USD');

ALTER TABLE product_rates
    DROP CONSTRAINT chk_product_rates_currency,
    ADD CONSTRAINT chk_product_rates_currency CHECK (currency = 'USD');

ALTER TABLE checkouts
    DROP CONSTRAINT chk_checkouts_currency,
    DROP CONSTRAINT chk_checkouts_payment_binding,
    ADD CONSTRAINT chk_checkouts_currency CHECK (currency = 'USD'),
    ADD CONSTRAINT chk_checkouts_payment_binding CHECK (
        (provider IS NULL AND payment_method IS NULL)
        OR (provider = 'stripe' AND payment_method = 'card')
    );

ALTER TABLE payments
    DROP CONSTRAINT chk_payments_provider,
    DROP CONSTRAINT chk_payments_payment_method,
    DROP CONSTRAINT chk_payments_currency,
    ADD CONSTRAINT chk_payments_provider CHECK (provider = 'stripe'),
    ADD CONSTRAINT chk_payments_payment_method CHECK (payment_method = 'card'),
    ADD CONSTRAINT chk_payments_currency CHECK (currency = 'USD');

ALTER TABLE payment_provider_events
    DROP CONSTRAINT chk_payment_provider_events_provider,
    ADD CONSTRAINT chk_payment_provider_events_provider CHECK (provider = 'stripe');
