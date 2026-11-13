ALTER TABLE devices DROP CONSTRAINT IF EXISTS uq_devices_org_serial;
DROP INDEX IF EXISTS uq_vehicles_org_plate;
DROP INDEX IF EXISTS uq_vehicles_org_vin;

ALTER TABLE organizations ADD CONSTRAINT organizations_name_key UNIQUE (name);
ALTER TABLE devices ADD CONSTRAINT devices_serial_number_key UNIQUE (serial_number);
ALTER TABLE vehicles ADD CONSTRAINT vehicles_number_plate_key UNIQUE (number_plate);
ALTER TABLE vehicles ADD CONSTRAINT vehicles_vin_key UNIQUE (vin);
