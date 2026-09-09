TRUNCATE TABLE devices CASCADE;
ALTER TABLE devices ADD COLUMN organization_id BIGINT NOT NULL REFERENCES organizations(id);