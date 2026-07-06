CREATE TABLE driver_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    driver_id BIGINT NOT NULL REFERENCES drivers(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ
);

DROP INDEX IF EXISTS idx_telemetry_vehicle_created;
DROP INDEX IF EXISTS idx_telemetry_vehicle;
DROP INDEX IF EXISTS idx_trip_driver;
DROP INDEX IF EXISTS idx_trip_vehicle;
DROP INDEX IF EXISTS idx_driver_organization;
DROP INDEX IF EXISTS idx_vehicle_organization;

ALTER TABLE devices ALTER COLUMN status DROP DEFAULT;
ALTER TABLE devices ALTER COLUMN status DROP NOT NULL;
ALTER TABLE devices DROP CONSTRAINT IF EXISTS chk_device_status;

ALTER TABLE trips ALTER COLUMN status DROP DEFAULT;
ALTER TABLE trips ALTER COLUMN status DROP NOT NULL;
ALTER TABLE trips DROP CONSTRAINT IF EXISTS chk_trip_status;

ALTER TABLE drivers DROP COLUMN IF EXISTS updated_at;
ALTER TABLE drivers DROP COLUMN IF EXISTS created_at;

ALTER TABLE organizations DROP COLUMN IF EXISTS updated_at;
ALTER TABLE organizations DROP COLUMN IF EXISTS created_at;
ALTER TABLE organizations DROP COLUMN IF EXISTS name;

ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS uq_vehicle_number_plate;
ALTER TABLE vehicles DROP COLUMN IF EXISTS status;
ALTER TABLE vehicles DROP COLUMN IF EXISTS model;
ALTER TABLE vehicles ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
