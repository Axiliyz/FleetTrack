DROP INDEX IF EXISTS idx_telemetry_device_time;
DROP INDEX IF EXISTS idx_telemetry_vehicle_time;

CREATE INDEX idx_telemetry_device_time
  ON telemetry(device_id, created_at DESC);

CREATE INDEX idx_telemetry_vehicle_time
  ON telemetry(vehicle_id, created_at DESC);

ALTER TABLE telemetry
  DROP COLUMN device_timestamp;

ALTER TABLE telemetry
  RENAME COLUMN received_at TO created_at;
