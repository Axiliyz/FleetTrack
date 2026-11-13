ALTER TABLE refresh_tokens
    DROP CONSTRAINT refresh_tokens_user_id_fkey,
    ADD CONSTRAINT refresh_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id);

DROP INDEX IF EXISTS idx_refresh_tokens_expires;
DROP INDEX IF EXISTS idx_refresh_tokens_family;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS family_id;

DROP INDEX IF EXISTS idx_alert_rules_org_active;

DROP INDEX IF EXISTS idx_users_driver_id;
CREATE UNIQUE INDEX idx_users_driver_id ON users (driver_id) WHERE driver_id IS NOT NULL;

DROP INDEX IF EXISTS uq_users_email;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE alert_rules DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
