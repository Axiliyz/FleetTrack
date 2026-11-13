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

ALTER TABLE organizations ADD COLUMN name VARCHAR(255) UNIQUE NOT NULL;
ALTER TABLE organizations ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE organizations ADD COLUMN updated_at TIMESTAMPTZ;

ALTER TABLE drivers ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE drivers ADD COLUMN updated_at TIMESTAMPTZ;

ALTER TABLE trips
ADD CONSTRAINT chk_trip_status
CHECK (
    status IN (
        'RUNNING',
        'CANCELLED',
        'SUCCEEDED',
        'SLEEPING',
        'SERVING'
    )
);

ALTER TABLE trips
ALTER COLUMN status SET NOT NULL;

ALTER TABLE trips
ALTER COLUMN status SET DEFAULT 'RUNNING';

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
ALTER COLUMN status SET NOT NULL;

ALTER TABLE devices
ALTER COLUMN status SET DEFAULT 'ACTIVE';

CREATE INDEX idx_driver_organization
ON drivers(organization_id);

CREATE INDEX idx_trip_vehicle
ON trips(vehicle_id);

CREATE INDEX idx_trip_driver
ON trips(driver_id);
