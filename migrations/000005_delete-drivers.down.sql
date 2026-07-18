CREATE TABLE drivers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    name VARCHAR(35),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

ALTER TABLE trips ADD COLUMN driver_id BIGINT REFERENCES drivers(id);

CREATE INDEX idx_trip_driver
ON trips(driver_id);
