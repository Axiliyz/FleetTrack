ALTER TABLE telemetry
  RENAME COLUMN created_at TO received_at;

ALTER TABLE telemetry
  ADD COLUMN device_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW();

DROP INDEX IF EXISTS idx_telemetry_device_time;
DROP INDEX IF EXISTS idx_telemetry_vehicle_time;

CREATE INDEX idx_telemetry_device_time
  ON telemetry(device_id, received_at DESC);

CREATE INDEX idx_telemetry_vehicle_time
  ON telemetry(vehicle_id, received_at DESC);
