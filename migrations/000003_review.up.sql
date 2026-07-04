ALTER TABLE vehicles DROP COLUMN version;

ALTER TABLE vehicles ADD COLUMN model VARCHAR(30) NOT NULL;

ALTER TABLE vehicles 
ADD COLUMN status VARCHAR(15)
NOT NULL
DEFAULT 'IDLE'
CHECK (
    status IN (
        'IDLE', 
        'ON_TRIP', 
        'IN_SERVICE', 
        'DELETED'
    )
);

ALTER TABLE vehicles
ADD CONSTRAINT uq_vehicle_number_plate
UNIQUE (number_plate);

ALTER TABLE organizations ADD COLUMN name VARCHAR(255) UNIQUE NOT NULL;
ALTER TABLE organizations ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE organizations ADD COLUMN updated_at TIMESTAMPTZ;

ALTER TABLE drivers ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE drivers ADD COLUMN updated_at TIMESTAMPTZ;

ALTER TABLE trips
ADD CONSTRAINT chk_trip_status
CHECK (
    status IN (
        'CREATED',
        'ACTIVE',
        'FINISHED',
        'CANCELLED'
    )
);

ALTER TABLE trips
ALTER COLUMN status
SET NOT NULL DEFAULT 'CREATED';

ALTER TABLE devices
ADD CONSTRAINT chk_device_status
CHECK (
    status IN (
        'ACTIVE',
        'INACTIVE',
        'MAINTENANCE'
    )
);

ALTER TABLE devices
ALTER COLUMN status
SET NOT NULL DEFAULT 'ACTIVE';

CREATE INDEX idx_vehicle_organization
ON vehicles(organization_id);

CREATE INDEX idx_driver_organization
ON drivers(organization_id);

CREATE INDEX idx_trip_vehicle
ON trips(vehicle_id);

CREATE INDEX idx_trip_driver
ON trips(driver_id);

CREATE INDEX idx_telemetry_vehicle
ON telemetry(vehicle_id);

CREATE INDEX idx_telemetry_trip
ON telemetry(trip_id);

CREATE INDEX idx_device_vehicle
ON devices(vehicle_id);

CREATE INDEX idx_telemetry_vehicle_created
ON telemetry(vehicle_id, created_at DESC);