-- Пользователей и правила удаляем мягко: на них ссылаются алерты (acknowledged_by, rule_id)
-- и история не должна терять автора или причину срабатывания
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE alert_rules ADD COLUMN deleted_at TIMESTAMPTZ;

-- почта сравнивается без учёта регистра: Sanya@ и sanya@ - один человек.
-- Удалённый пользователь адрес не держит, его можно завести заново
UPDATE users SET email = LOWER(email);
ALTER TABLE users DROP CONSTRAINT users_email_key;
CREATE UNIQUE INDEX uq_users_email ON users (LOWER(email)) WHERE deleted_at IS NULL;

DROP INDEX idx_users_driver_id;
CREATE UNIQUE INDEX idx_users_driver_id ON users (driver_id)
    WHERE driver_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_alert_rules_org_active ON alert_rules (organization_id)
    WHERE deleted_at IS NULL AND enabled;

-- refresh-токены: все токены одной цепочки ротаций - одна семья. Повторно пришедший отозванный
-- токен значит, что его украли, и тогда отзывается вся семья
ALTER TABLE refresh_tokens ADD COLUMN family_id UUID NOT NULL DEFAULT gen_random_uuid();
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens (family_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens (expires_at);

ALTER TABLE refresh_tokens
    DROP CONSTRAINT refresh_tokens_user_id_fkey,
    ADD CONSTRAINT refresh_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
