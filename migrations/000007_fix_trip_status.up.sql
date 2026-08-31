ALTER TABLE trips DROP CONSTRAINT chk_trip_status;

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

ALTER TABLE trips ALTER COLUMN status SET DEFAULT 'RUNNING';
