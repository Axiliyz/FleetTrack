ALTER TABLE trips
DROP COLUMN IF EXISTS avg_speed_kmh,
DROP COLUMN IF EXISTS max_speed_kmh,
DROP COLUMN IF EXISTS telemetry_count;