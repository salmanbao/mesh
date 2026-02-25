CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS profiles (
    profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    username VARCHAR(30) UNIQUE,
    display_name VARCHAR(50) NOT NULL,
    bio TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    banner_url TEXT NOT NULL DEFAULT '',
    kyc_status VARCHAR(50) NOT NULL DEFAULT 'not_started',
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    is_unlisted BOOLEAN NOT NULL DEFAULT FALSE,
    hide_statistics BOOLEAN NOT NULL DEFAULT FALSE,
    analytics_opt_out BOOLEAN NOT NULL DEFAULT FALSE,
    last_username_change_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT profiles_username_lowercase CHECK (username IS NULL OR username = LOWER(username)),
    CONSTRAINT profiles_username_pattern CHECK (username IS NULL OR username ~ '^[a-z0-9_]{3,30}$'),
    CONSTRAINT profiles_display_name_length CHECK (LENGTH(display_name) BETWEEN 3 AND 50),
    CONSTRAINT profiles_bio_length CHECK (LENGTH(bio) <= 200)
);
CREATE INDEX IF NOT EXISTS idx_profiles_username ON profiles (username);
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles (user_id);
CREATE INDEX IF NOT EXISTS idx_profiles_kyc_status ON profiles (kyc_status);
CREATE INDEX IF NOT EXISTS idx_profiles_deleted_at ON profiles (deleted_at);

CREATE TABLE IF NOT EXISTS reserved_usernames (
    reserved_username_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(30) UNIQUE NOT NULL,
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT
);

INSERT INTO reserved_usernames (username, reason)
VALUES
  ('admin', 'System reserved'),
  ('support', 'System reserved'),
  ('viralforge', 'System reserved'),
  ('profile', 'System reserved'),
  ('api', 'System reserved'),
  ('www', 'System reserved'),
  ('app', 'System reserved'),
  ('help', 'System reserved'),
  ('settings', 'System reserved'),
  ('dashboard', 'System reserved')
ON CONFLICT (username) DO NOTHING;

CREATE TABLE IF NOT EXISTS social_links (
    social_link_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    platform VARCHAR(50) NOT NULL,
    handle VARCHAR(255),
    profile_url TEXT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    oauth_connection_id UUID,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_synced_at TIMESTAMPTZ,
    UNIQUE (user_id, platform)
);
CREATE INDEX IF NOT EXISTS idx_social_links_user_id ON social_links (user_id);
CREATE INDEX IF NOT EXISTS idx_social_links_platform ON social_links (platform);

CREATE TABLE IF NOT EXISTS payout_methods (
    payout_method_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    method_type VARCHAR(50) NOT NULL,
    identifier_encrypted BYTEA NOT NULL,
    verification_status VARCHAR(50) NOT NULL DEFAULT 'unverified',
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    UNIQUE (user_id, method_type)
);
CREATE INDEX IF NOT EXISTS idx_payout_methods_user_id ON payout_methods (user_id);

CREATE TABLE IF NOT EXISTS kyc_documents (
    kyc_document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    document_type VARCHAR(50) NOT NULL,
    file_key TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'uploaded',
    rejection_reason TEXT,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID
);
CREATE INDEX IF NOT EXISTS idx_kyc_documents_user_id ON kyc_documents (user_id);
CREATE INDEX IF NOT EXISTS idx_kyc_documents_status ON kyc_documents (status);

CREATE TABLE IF NOT EXISTS username_history (
    history_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    old_username VARCHAR(30) NOT NULL,
    new_username VARCHAR(30) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    redirect_expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_username_history_old_username ON username_history (old_username);
CREATE INDEX IF NOT EXISTS idx_username_history_user_changed_at ON username_history (user_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS profile_stats (
    stat_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    total_earnings_ytd NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    submission_count INTEGER NOT NULL DEFAULT 0,
    approval_rate NUMERIC(6,2) NOT NULL DEFAULT 0.00,
    follower_count INTEGER NOT NULL DEFAULT 0,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_profile_stats_user_id ON profile_stats (user_id);

CREATE TABLE IF NOT EXISTS profile_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    partition_key VARCHAR(255) NOT NULL,
    partition_key_path VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    schema_version VARCHAR(16) NOT NULL DEFAULT '1.0',
    trace_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    last_error_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_profile_outbox_unpublished ON profile_outbox (published_at, created_at);

CREATE TABLE IF NOT EXISTS profile_idempotency (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    request_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL,
    response_code INT,
    response_body JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_profile_idempotency_expires_at ON profile_idempotency (expires_at);

CREATE TABLE IF NOT EXISTS profile_event_dedup (
    event_id VARCHAR(255) PRIMARY KEY,
    event_type VARCHAR(120) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_profile_event_dedup_expires_at ON profile_event_dedup (expires_at);

