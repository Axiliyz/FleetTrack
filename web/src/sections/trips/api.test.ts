import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, recordFetch } from '../../test/http'
import type { Trip } from '../../types'
import { errorText } from '../../api/errors'
import { VehicleNotSynced, removeTrip, setTripStatus, startTrip, tripParams } from './api'

const TRIP: Trip = {
  id: 12,
  driver_id: 4,
  vehicle_id: 3,
  started_at: '2026-10-20T10:00:00Z',
  ended_at: null,
  status: 'RUNNING',
  distance_km: 120,
  avg_speed_kmh: 70,
  max_speed_kmh: 95,
  telemetry_count: 300,
}

afterEach(() => vi.unstubAllGlobals())

// бэк не связывает статус машины с рейсом — это делает фронт, как и генератор парка
describe('рейс и статус машины', () => {
  it('старт рейса переводит машину в «в рейсе»', async () => {
    const calls = recordFetch(({ url }) => json(url.endsWith('/trips') ? TRIP : {}))
    await startTrip(4, 3)
    expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual(['POST /api/trips', 'PATCH /api/vehicles/3'])
    expect(calls[0].body).toEqual({ driver_id: 4, vehicle_id: 3 })
    expect(calls[1].body).toEqual({ status: 'ON_TRIP' })
  })

  it('завершение и отмена рейса в пути освобождают машину', async () => {
    for (const status of ['SUCCEEDED', 'CANCELLED'] as const) {
      const calls = recordFetch(() => json(TRIP))
      await setTripStatus(12, status)
      expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual([
        'GET /api/trips/12',
        'PATCH /api/trips/12',
        'PATCH /api/vehicles/3',
      ])
      expect(calls[2].body).toEqual({ status: 'IDLE' })
    }
  })

  it('прежний статус берётся с сервера, а не из устаревшего списка', async () => {
    // на экране рейс ещё «в пути», а на сервере его уже завершили: машину не трогаем
    const calls = recordFetch(({ method }) => json(method === 'GET' ? { ...TRIP, status: 'SUCCEEDED' } : TRIP))
    await setTripStatus(12, 'CANCELLED')
    expect(calls.map((c) => c.method)).toEqual(['GET', 'PATCH'])
  })

  it('возобновлённый рейс снова зажигает «в рейсе»', async () => {
    const calls = recordFetch(({ method }) => json(method === 'GET' ? { ...TRIP, status: 'SLEEPING' } : TRIP))
    await setTripStatus(12, 'RUNNING')
    expect(calls[2]).toMatchObject({ url: '/api/vehicles/3', body: { status: 'ON_TRIP' } })
  })

  it('пауза рейса машину не освобождает', async () => {
    const calls = recordFetch(() => json(TRIP))
    await setTripStatus(12, 'SLEEPING')
    expect(calls).toHaveLength(2)
  })

  it('если рейс не создался, статус машины не трогаем', async () => {
    const calls = recordFetch(() => json('vehicle is busy', 409))
    await expect(startTrip(4, 3)).rejects.toMatchObject({ status: 409 })
    expect(calls).toHaveLength(1)
  })

  it('рейс создан, а машина не обновилась — ошибка несёт рейс, чтобы не создать его второй раз', async () => {
    recordFetch(({ url }) => (url.endsWith('/trips') ? json(TRIP) : json('unknown error', 500)))
    const err = await startTrip(4, 3).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(VehicleNotSynced)
    expect((err as VehicleNotSynced).trip.id).toBe(12)
    expect(errorText(err)).toMatch(/Рейс №12 сохранён/)
  })

  it('удаление рейса в пути освобождает машину, завершённого — нет', async () => {
    let calls = recordFetch(() => json(TRIP))
    await removeTrip(12)
    expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual([
      'GET /api/trips/12',
      'DELETE /api/trips/12',
      'PATCH /api/vehicles/3',
    ])

    calls = recordFetch(() => json({ ...TRIP, status: 'SUCCEEDED' }))
    await removeTrip(12)
    expect(calls).toHaveLength(2)
  })
})

describe('tripParams', () => {
  it('даты из адреса — в RFC3339, «по» включает весь день', () => {
    const params = tripParams({ status: '', vehicle_id: '3', driver_id: '', started_from: '2026-10-01', started_to: '2026-10-02' })
    expect(params.vehicle_id).toBe('3')
    expect(new Date(params.started_from ?? '').getDate()).toBe(1)
    expect(new Date(params.started_to ?? '').getHours()).toBe(23)
  })
})
