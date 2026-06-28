CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    registration_id UUID NOT NULL REFERENCES registrations (id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    razorpay_order_id TEXT NOT NULL UNIQUE, -- created before payment
    razorpay_payment_id TEXT UNIQUE, -- filled after success
    razorpay_signature TEXT, -- for verification
    amount_in_paise BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    status TEXT NOT NULL DEFAULT 'created' CHECK (
        status IN (
            'created',
            'paid',
            'failed',
            'refunded'
        )
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_registration_id ON payments (registration_id);

CREATE INDEX idx_payments_razorpay_order_id ON payments (razorpay_order_id);