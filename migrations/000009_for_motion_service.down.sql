DROP INDEX IF EXISTS idx_telemetry_vehicle_device_timestamp;
DROP INDEX IF EXISTS idx_telemetry_trip_id;

ALTER TABLE telemetry
DROP COLUMN IF EXISTS trip_id,
DROP COLUMN IF EXISTS distance_km,
DROP COLUMN IF EXISTS speed_kmh;

ALTER TABLE trips
DROP COLUMN IF EXISTS distance_km;