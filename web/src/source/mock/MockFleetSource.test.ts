import { describe, expect, it } from 'vitest'
import type { Alert, AlertType, FleetEvent } from '../../types'
import { MockFleetSource } from './MockFleetSource'

const START = Date.parse('2026-10-10T09:00:00Z')

function record(source: MockFleetSource) {
  const events: FleetEvent[] = []
  source.subscribe({ onEvent: (e) => events.push(e) })
  return events
}

const alertsOf = (events: FleetEvent[], type?: AlertType): Alert[] =>
  events.flatMap((e) => (e.type === 'alert' && (!type || e.data.type === type) ? [e.data] : []))

describe('MockFleetSource', () => {
  it('отдаёт согласованный снимок парка', async () => {
    const snap = await new MockFleetSource({ startAt: START, manual: true }).load()

    expect(snap.vehicles).toHaveLength(14)
    expect(snap.positions.map((p) => p.vehicle_id).sort((a, b) => a - b)).toEqual(snap.vehicles.map((v) => v.id))

    const running = snap.trips.filter((t) => t.status === 'RUNNING')
    const onTrip = snap.vehicles.filter((v) => v.status === 'ON_TRIP')
    expect(running.map((t) => t.vehicle_id).sort()).toEqual(onTrip.map((v) => v.id).sort())
    // один водитель не может вести две машины
    expect(new Set(running.map((t) => t.driver_id)).size).toBe(running.length)
  })

  it('открывает и сам закрывает превышение скорости', () => {
    const source = new MockFleetSource({ startAt: START, manual: true })
    const events = record(source)
    source.advance(30_000)

    const speeding = alertsOf(events, 'SPEED_EXCEEDED')
    const fired = speeding.find((a) => a.status === 'FIRED')
    expect(fired).toBeDefined()
    expect(fired!.value).toBeGreaterThan(90)
    expect(speeding.some((a) => a.id === fired!.id && a.status === 'RESOLVED')).toBe(true)
  })

  it('поднимает DEVICE_OFFLINE, когда трекер молчит, и закрывает при возврате связи', () => {
    const source = new MockFleetSource({ startAt: START, manual: true })
    const events = record(source)
    source.advance(40_000)

    const offline = alertsOf(events, 'DEVICE_OFFLINE')
    const fired = offline.find((a) => a.status === 'FIRED')
    expect(fired).toBeDefined()
    expect(offline.some((a) => a.id === fired!.id && a.status === 'RESOLVED')).toBe(true)
  })

  it('держит не больше одного открытого алерта на машину и тип', () => {
    const source = new MockFleetSource({ startAt: START, manual: true, scale: 300 })
    const events = record(source)
    const open = new Map<string, number>()

    for (let i = 0; i < 240; i++) {
      source.advance(1000)
      for (const a of alertsOf(events.splice(0))) {
        const key = `${a.vehicle_id}:${a.type}`
        if (a.status === 'FIRED') {
          expect(open.has(key), `второй открытый алерт ${key}`).toBe(false)
          open.set(key, a.id)
        } else if (a.status === 'RESOLVED') {
          expect(open.get(key)).toBe(a.id)
          open.delete(key)
        }
      }
    }
  })

  it('принимает алерт в работу от имени диспетчера', async () => {
    const source = new MockFleetSource({ startAt: START, manual: true })
    const events = record(source)
    source.advance(30_000)
    // топливо держится ниже порога до заправки, так что алерт ещё открыт
    const fired = alertsOf(events, 'LOW_FUEL').find((a) => a.status === 'FIRED')!
    expect(alertsOf(events, 'LOW_FUEL').some((a) => a.id === fired.id && a.status === 'RESOLVED')).toBe(false)

    events.length = 0
    await source.acknowledgeAlert(fired.id)

    const [ack] = alertsOf(events)
    expect(ack).toMatchObject({ id: fired.id, status: 'ACKNOWLEDGED', acknowledged_by: 1 })
    expect(ack.acknowledged_at).not.toBeNull()
  })

  it('на паузе не шлёт событий и не двигает время', () => {
    const source = new MockFleetSource({ startAt: START, manual: true })
    const events = record(source)
    source.sim.setScale(0)
    source.advance(10_000)

    expect(events).toHaveLength(0)
    expect(source.sim.now()).toBe(START)
  })

  it('доводит рейс до конца и разворачивает машину обратно', () => {
    const source = new MockFleetSource({ startAt: START, manual: true, scale: 300 })
    const events = record(source)
    source.advance(240_000)

    const trips = events.flatMap((e) => (e.type === 'trip' ? [e.data] : []))
    const done = trips.find((t) => t.status === 'SUCCEEDED')
    expect(done).toBeDefined()
    expect(done!.ended_at).not.toBeNull()
    // после разгрузки та же машина уходит в новый рейс
    expect(trips.some((t) => t.vehicle_id === done!.vehicle_id && t.id > done!.id && t.status === 'RUNNING')).toBe(true)
  })
})
