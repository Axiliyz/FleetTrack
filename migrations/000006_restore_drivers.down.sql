DROP INDEX idx_trip_driver;

ALTER TABLE trips DROP COLUMN driver_id;

DROP TABLE drivers;
