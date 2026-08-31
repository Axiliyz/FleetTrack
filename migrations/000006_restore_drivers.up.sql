CREATE TABLE drivers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    name VARCHAR(35),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- Старые записи trips не содержат данных о водителе (driver_id был удалён миграцией 000005) -
-- восстановить связь нечем, поэтому таблица очищается перед добавлением NOT NULL колонки
TRUNCATE TABLE trips;

ALTER TABLE trips ADD COLUMN driver_id BIGINT NOT NULL REFERENCES drivers(id);

CREATE INDEX idx_trip_driver ON trips(driver_id);
