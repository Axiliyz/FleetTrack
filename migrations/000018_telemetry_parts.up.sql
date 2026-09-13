DROP INDEX IF EXISTS idx_telemetry_device_time;
DROP INDEX IF EXISTS idx_telemetry_vehicle_created;
DROP INDEX IF EXISTS idx_telemetry_vehicle_device_timestamp;
DROP INDEX IF EXISTS idx_telemetry_trip_id;
DROP INDEX IF EXISTS idx_telemetry_vehicle_time;
DROP INDEX IF EXISTS idx_telemetry_vehicle;

ALTER TABLE telemetry RENAME TO telemetry_old;

CREATE TABLE telemetry (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    device_id BIGINT NOT NULL REFERENCES devices(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    latitude NUMERIC(9, 4),
    longitude NUMERIC(9, 4),
    fuel NUMERIC(5, 2),

    received_at TIMESTAMPTZ NOT NULL,
    device_timestamp TIMESTAMPTZ NOT NULL,
    trip_id BIGINT NOT NULL REFERENCES trips(id),
    distance_km NUMERIC(10, 3) NOT NULL DEFAULT 0,
    speed_kmh NUMERIC(6, 2) NOT NULL DEFAULT 0,
    PRIMARY KEY (id, received_at)
) PARTITION BY RANGE (received_at);

CREATE TABLE telemetry_2026_06 PARTITION OF telemetry
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE telemetry_2026_07 PARTITION OF telemetry
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE TABLE telemetry_2026_08 PARTITION OF telemetry
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');

CREATE TABLE telemetry_2026_09 PARTITION OF telemetry
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');

CREATE TABLE telemetry_2026_10 PARTITION OF telemetry
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');

INSERT INTO telemetry (
    organization_id, device_id, vehicle_id,
    latitude, longitude, fuel,
    received_at, device_timestamp, trip_id,
    distance_km, speed_kmh
)
SELECT
    organization_id, device_id, vehicle_id,
    latitude, longitude, fuel,
    received_at, device_timestamp, trip_id,
    distance_km, speed_kmh
FROM telemetry_old;

CREATE INDEX idx_telemetry_device_time
    ON telemetry (device_id, received_at DESC);

CREATE INDEX idx_telemetry_vehicle_created
    ON telemetry (vehicle_id, received_at DESC);

CREATE INDEX idx_telemetry_vehicle_device_timestamp
    ON telemetry (vehicle_id, device_timestamp DESC);

CREATE INDEX idx_telemetry_trip_id
    ON telemetry (trip_id);

DROP TABLE telemetry_old;