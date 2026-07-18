CREATE UNIQUE INDEX vehicle_active_device
ON device_assignments(vehicle_id)
WHERE ended_at IS NULL;
