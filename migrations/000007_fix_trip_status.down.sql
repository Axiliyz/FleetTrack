ALTER TABLE trips ALTER COLUMN status SET DEFAULT 'CREATED';

ALTER TABLE trips DROP CONSTRAINT chk_trip_status;

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
