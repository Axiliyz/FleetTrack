ALTER TABLE telemetry
ADD COLUMN trip_id BIGINT REFERENCES trips(id),
ADD COLUMN distance_km NUMERIC(10, 3) NOT NULL DEFAULT 0,
ADD COLUMN speed_kmh NUMERIC(6, 2) NOT NULL DEFAULT 0;

ALTER TABLE trips
ADD COLUMN distance_km NUMERIC(10, 3) NOT NULL DEFAULT 0;

CREATE INDEX idx_telemetry_vehicle_device_timestamp
ON telemetry (vehicle_id, device_timestamp DESC);

CREATE INDEX idx_telemetry_trip_id
ON telemetry(trip_id);
