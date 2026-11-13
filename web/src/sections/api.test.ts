// Запросы разделов: метод, адрес и тело совпадают с internal/router и internal/handler/dto
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, recordFetch } from '../test/http'
import { alertsApi } from './alerts/api'
import { devicesApi } from './devices/api'
import { driversApi } from './drivers/api'
import { teamApi } from './team/api'
import { telemetryApi } from './telemetry/api'
import { vehiclesApi } from './vehicles/api'

afterEach(() => vi.unstubAllGlobals())

const sent = (calls: ReturnType<typeof recordFetch>) => calls.map((c) => `${c.method} ${c.url}`)

describe('машины', () => {
  it('создание и частичное изменение', async () => {
    const calls = recordFetch()
    await vehiclesApi.create({ vin: 'XTA21099012345678', number_plate: 'А123ВС77', model: 'КАМАЗ' })
    await vehiclesApi.update(3, { status: 'IN_SERVICE' })
    await vehiclesApi.remove(3)
    expect(sent(calls)).toEqual(['POST /api/vehicles', 'PATCH /api/vehicles/3', 'DELETE /api/vehicles/3'])
    // organization_id не шлём: бэк всё равно берёт организацию из токена
    expect(calls[0].body).toEqual({ vin: 'XTA21099012345678', number_plate: 'А123ВС77', model: 'КАМАЗ' })
    expect(calls[1].body).toEqual({ status: 'IN_SERVICE' })
  })
})

describe('водители', () => {
  it('создание и переименование', async () => {
    const calls = recordFetch()
    await driversApi.create('Пётр Сидоров')
    await driversApi.rename(4, 'Пётр Сидоров-Ким')
    expect(sent(calls)).toEqual(['POST /api/drivers', 'PATCH /api/drivers/4'])
    expect(calls[1].body).toEqual({ name: 'Пётр Сидоров-Ким' })
  })
})

describe('трекеры', () => {
  it('регистрация и привязка к машине', async () => {
    const calls = recordFetch()
    await devicesApi.create('TRK-0001')
    await devicesApi.assign(7, 3)
    expect(sent(calls)).toEqual(['POST /api/devices', 'POST /api/assignments'])
    expect(calls[0].body).toEqual({ serial_number: 'TRK-0001' })
    expect(calls[1].body).toEqual({ device_id: 7, vehicle_id: 3 })
  })
})

describe('телеметрия', () => {
  it('трек машины — за период, рейса — целиком', async () => {
    const calls = recordFetch(() => json([]))
    await telemetryApi.track({ vehicleId: 3, from: '2026-10-20T10:00:00.000Z' })
    await telemetryApi.track({ tripId: 12 })
    expect(calls[0].url).toBe('/api/telemetry?vehicle_id=3&from=2026-10-20T10%3A00%3A00.000Z&limit=500')
    expect(calls[1].url).toBe('/api/telemetry?trip_id=12&limit=500')
  })

  it('удаление точки и всей истории машины', async () => {
    const calls = recordFetch(() => json([]))
    await telemetryApi.remove(99)
    await telemetryApi.clearVehicle(3)
    expect(sent(calls)).toEqual(['DELETE /api/telemetry/99', 'DELETE /api/telemetry/vehicles/3'])
  })
})

describe('тревоги', () => {
  it('принять в работу можно только статусом ACKNOWLEDGED', async () => {
    const calls = recordFetch()
    await alertsApi.acknowledge(5)
    expect(sent(calls)).toEqual(['PATCH /api/alerts/5'])
    expect(calls[0].body).toEqual({ status: 'ACKNOWLEDGED' })
  })

  it('правило уходит без organization_id', async () => {
    const calls = recordFetch()
    await alertsApi.createRule({ type: 'LOW_FUEL', name: 'Мало топлива', threshold: 15, severity: 'MEDIUM', enabled: true })
    expect(calls[0].body).toEqual({ type: 'LOW_FUEL', name: 'Мало топлива', threshold: 15, severity: 'MEDIUM', enabled: true })
  })
})

describe('организация', () => {
  it('из списка берётся своя — единственная', async () => {
    recordFetch(() => json([{ id: 5, name: 'Северный тракт' }]))
    await expect(teamApi.organization()).resolves.toEqual({ id: 5, name: 'Северный тракт' })
  })
})
