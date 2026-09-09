DROP INDEX IF EXISTS idx_users_driver_id;
ALTER TABLE users
DROP COLUMN IF EXISTS driver_id;
