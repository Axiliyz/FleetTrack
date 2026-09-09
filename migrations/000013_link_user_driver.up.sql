ALTER TABLE users
ADD COLUMN driver_id BIGINT REFERENCES drivers(id);

CREATE UNIQUE INDEX idx_users_driver_id ON users(driver_id) WHERE driver_id IS NOT NULL;
