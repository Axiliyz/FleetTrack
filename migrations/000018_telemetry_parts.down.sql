ALTER TABLE telemetry RENAME TO telemetry_partitioned;

CREATE TABLE telemetry (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    device_id BIGINT NOT NULL REFERENCES devices(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    latitude NUMERIC(9, 4),
    longitude NUMERIC(9, 4),
    fuel NUMERIC(5, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    received_at TIMESTAMPTZ NOT NULL,
    device_timestamp TIMESTAMPTZ NOT NULL,
    trip_id BIGINT NOT NULL REFERENCES trips(id),
    distance_km NUMERIC(10, 3) NOT NULL DEFAULT 0,
    speed_kmh NUMERIC(6, 2) NOT NULL DEFAULT 0
);

INSERT INTO telemetry (
    organization_id, device_id, vehicle_id,
    latitude, longitude, fuel, created_at,
    received_at, device_timestamp, trip_id,
    distance_km, speed_kmh
)
SELECT
    organization_id, device_id, vehicle_id,
    latitude, longitude, fuel, created_at,
    received_at, device_timestamp, trip_id,
    distance_km, speed_kmh
FROM telemetry_partitioned;

CREATE INDEX idx_telemetry_device_time
    ON telemetry (device_id, received_at DESC);

CREATE INDEX idx_telemetry_vehicle_created
    ON telemetry (vehicle_id, received_at DESC);

CREATE INDEX idx_telemetry_vehicle_device_timestamp
    ON telemetry (vehicle_id, device_timestamp DESC);

CREATE INDEX idx_telemetry_trip_id
    ON telemetry (trip_id);

DROP TABLE telemetry_partitioned;