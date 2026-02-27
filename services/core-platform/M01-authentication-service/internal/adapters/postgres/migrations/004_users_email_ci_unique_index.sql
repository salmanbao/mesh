CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower_unique
    ON users ((LOWER(email)));

