CREATE TABLE IF NOT EXISTS alert_rules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    type VARCHAR(30) NOT NULL CHECK (type IN ('SPEED_EXCEEDED', 'DEVICE_OFFLINE', 'LOW_FUEL')),
    name VARCHAR(40) NOT NULL,
    threshold NUMERIC(6, 2) NOT NULL,
    severity VARCHAR(30) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS alerts (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    rule_id BIGINT NOT NULL REFERENCES alert_rules(id),
    type VARCHAR(30) NOT NULL CHECK (type IN ('SPEED_EXCEEDED', 'DEVICE_OFFLINE', 'LOW_FUEL')),
    severity VARCHAR(30) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    status VARCHAR(15) NOT NULL CHECK (status IN ('FIRED', 'ACKNOWLEDGED', 'RESOLVED')),
    message TEXT NOT NULL,
    value NUMERIC(10, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by BIGINT REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    resolved_by BIGINT REFERENCES users(id),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE alerts_2026_09 PARTITION OF alerts
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');

CREATE TABLE alerts_2026_10 PARTITION OF alerts
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');

CREATE TABLE IF NOT EXISTS user_notification_channels (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(30) NOT NULL CHECK (type IN ('EMAIL', 'TELEGRAM', 'WEBHOOK')),
    config JSONB NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    min_severity VARCHAR(30) NOT NULL CHECK (min_severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alert_notifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    alert_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL REFERENCES user_notification_channels(id),
    status VARCHAR(15) NOT NULL CHECK (status IN ('PENDING', 'SENT', 'FAILED')) DEFAULT 'PENDING',
    attempts INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_org ON alert_rules(organization_id);
CREATE INDEX idx_alerts_org_status_time ON alerts(organization_id, status, created_at DESC);
CREATE UNIQUE INDEX idx_alerts_active_unique ON alerts(vehicle_id, rule_id, created_at)
    WHERE status != 'RESOLVED';
CREATE INDEX idx_alerts_fired ON alerts(organization_id, created_at DESC)
    WHERE status = 'FIRED';
CREATE INDEX idx_alerts_rule_vehicle ON alerts(rule_id, vehicle_id, created_at DESC);
CREATE INDEX idx_alert_notifications_pending ON alert_notifications(next_retry_at)
    WHERE status = 'PENDING';
