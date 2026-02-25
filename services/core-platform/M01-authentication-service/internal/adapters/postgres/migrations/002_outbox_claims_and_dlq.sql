ALTER TABLE auth_outbox
    ADD COLUMN IF NOT EXISTS claim_token VARCHAR(128),
    ADD COLUMN IF NOT EXISTS claim_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS dead_lettered_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_auth_outbox_claim_until
    ON auth_outbox (claim_until)
    WHERE published_at IS NULL AND dead_lettered_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_auth_outbox_dead_lettered_at
    ON auth_outbox (dead_lettered_at);
