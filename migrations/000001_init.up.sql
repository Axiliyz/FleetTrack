CREATE TABLE organizations(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY 
) ;

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    name VARCHAR(35)
);

CREATE TABLE vehicles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    version INTEGER NOT NULL DEFAULT 1,
    vin VARCHAR(17) UNIQUE, 
    number_plate VARCHAR(10) UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE TABLE devices (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version INTEGER NOT NULL DEFAULT 1,
    serial_number VARCHAR(64) UNIQUE NOT NULL,
    status VARCHAR(15),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE device_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    device_id BIGINT NOT NULL REFERENCES devices(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE TABLE telemetry (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    device_id BIGINT NOT NULL REFERENCES devices(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    latitude NUMERIC(9, 4),
    longitude NUMERIC(9, 4),
    fuel NUMERIC(5, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE drivers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    name VARCHAR(35)
);

CREATE TABLE driver_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    driver_id BIGINT NOT NULL REFERENCES drivers(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ
);

CREATE TABLE trips (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    driver_id BIGINT NOT NULL REFERENCES drivers(id),
    vehicle_id BIGINT NOT NULL REFERENCES vehicles(id),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    status VARCHAR(15)
);

CREATE UNIQUE INDEX device_active_assignment
ON device_assignments(device_id)
WHERE ended_at IS NULL;


CREATE UNIQUE INDEX vehicle_active_device
ON device_assignments(vehicle_id)
WHERE ended_at IS NULL;

CREATE INDEX idx_users_org
ON users(organization_id)

CREATE INDEX idx_vehicle_org
ON vehicles(organization_id)

CREATE INDEX idx_telemetry_device_time
ON telemetry(device_id, created_at DESC)

CREATE INDEX idx_telemetry_vehicle_time
ON telemetry(vehicle_id, created_at DESC)

--- Partitions ---

-- CREATE TABLE telemetry_26_06
-- PARTITION OF telemetry
-- FOR VALUES FROM ('2026-06-01')
-- TO ('2026-07-01');

-- CREATE TABLE telemetry_26_07
-- PARTITION OF telemetry
-- FOR VALUES FROM ('2026-07-01')
-- TO ('2026-08-01')