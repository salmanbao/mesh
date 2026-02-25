ALTER TABLE oauth_connections
    ADD COLUMN IF NOT EXISTS issuer VARCHAR(255),
    ADD COLUMN IF NOT EXISTS subject VARCHAR(255),
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS email_at_link_time VARCHAR(255);

UPDATE oauth_connections
SET
    issuer = COALESCE(NULLIF(issuer, ''), provider),
    subject = COALESCE(NULLIF(subject, ''), provider_user_id),
    last_login_at = COALESCE(last_login_at, linked_at),
    email_at_link_time = COALESCE(NULLIF(email_at_link_time, ''), email)
WHERE issuer IS NULL OR subject IS NULL OR last_login_at IS NULL OR email_at_link_time IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_connections_issuer_subject
    ON oauth_connections (issuer, subject);
