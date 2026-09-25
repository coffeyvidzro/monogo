-- Only one payment may await or have completed a checkout.
CREATE UNIQUE INDEX uq_payments_active_checkout ON payments (checkout_id)
    WHERE status IN ('pending', 'succeeded');
