-- Уникальность VIN, госномера и серийника теперь в пределах организации.
-- Глобальный ключ выдавал чужие данные («такой VIN уже есть» = машина есть у другого клиента)
-- и позволял занять серийник чужого трекера раньше владельца
ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_vin_key;
ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS vehicles_number_plate_key;
-- на базах, накативших старую 000003, лежит второй такой же ключ на номер
ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS uq_vehicle_number_plate;
ALTER TABLE devices DROP CONSTRAINT IF EXISTS devices_serial_number_key;

-- название организации не уникально: две «Ромашки» в разных городах - обычное дело,
-- а ответ «название занято» подтверждал, что такой клиент есть
ALTER TABLE organizations DROP CONSTRAINT IF EXISTS organizations_name_key;

-- удалённая машина (soft delete) VIN и номер не держит, иначе ту же машину не завести заново
CREATE UNIQUE INDEX uq_vehicles_org_vin
    ON vehicles (organization_id, vin)
    WHERE status <> 'DELETED';

CREATE UNIQUE INDEX uq_vehicles_org_plate
    ON vehicles (organization_id, number_plate)
    WHERE status <> 'DELETED';

ALTER TABLE devices
    ADD CONSTRAINT uq_devices_org_serial UNIQUE (organization_id, serial_number);
