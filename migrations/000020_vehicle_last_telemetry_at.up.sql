ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS last_telemetry_at TIMESTAMPTZ;
UPDATE vehicles v
SET last_telemetry_at = sub.max_received_at
FROM (
    SELECT vehicle_id, MAX(received_at) AS max_received_at
    FROM telemetry
    GROUP BY vehicle_id
) sub
WHERE v.id = sub.vehicle_id;