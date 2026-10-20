// Генератор парка для живого режима.
//
// Регистрирует новую организацию, заводит водителей, машины, трекеры и правила алертов, а дальше
// гонит по трассам тот же симулятор, что работает в мок-режиме фронта, и отправляет каждую точку
// в POST /telemetry, как настоящий GPS-терминал. Рейсы и статусы машин он меняет через API,
// а алерты, скорость и пробег считает уже сам бэкенд.
//
// Время ускорено (FLEET_SCALE, по умолчанию ×60): точка уходит раз в секунду, а device_timestamp
// в ней — время симуляции. Бэк считает скорость по device_timestamp, поэтому она остаётся реальной.
//
// Запуск: npm run sim   (бэкенд должен работать на FLEET_API, по умолчанию http://localhost:8080)

import { randomBytes } from 'node:crypto'
import { writeFileSync } from 'node:fs'
import { MockFleetSource } from '../src/source/mock/MockFleetSource'
import type { AlertType, FleetEvent, Trip, VehicleStatus } from '../src/types'

const API = (process.env.FLEET_API ?? 'http://localhost:8080').replace(/\/$/, '')
const SCALE = Number(process.env.FLEET_SCALE ?? 60)
const CREDENTIALS_FILE = new URL('../.fleet-sim.local.json', import.meta.url)

// раз в сколько секунд один трекер «теряет связь» и на сколько: бэк должен поднять DEVICE_OFFLINE.
// Тишина с запасом над порогом правила (1 мин) и тиком OfflineWorker (30 с), который на большой базе
// успевает обойти не все машины за тик
const FIRST_SILENCE_S = 30
const SILENCE_EVERY_S = 240
const SILENCE_FOR_S = 150
// если бэк не успевает, точки машины копятся в очереди; сверх этого новые выбрасываются
const MAX_QUEUE_PER_VEHICLE = 30

// ---------- клиент API ----------

class ApiClient {
  private access = ''
  private refreshToken = ''

  setTokens(t: { access_token: string; refresh_token: string }) {
    this.access = t.access_token
    this.refreshToken = t.refresh_token
  }

  async call<T>(method: string, path: string, body?: unknown, retried = false): Promise<T> {
    const res = await fetch(`${API}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(this.access && { Authorization: `Bearer ${this.access}` }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    if (res.status === 401 && !retried && this.refreshToken) {
      await this.refresh()
      return this.call<T>(method, path, body, true)
    }
    const text = await res.text()
    const parsed = text ? (JSON.parse(text) as { data?: T; message?: string }) : {}
    if (!res.ok) throw new Error(`${method} ${path}: HTTP ${res.status} ${parsed.message ?? ''}`.trim())
    return parsed.data as T
  }

  private refreshing: Promise<void> | null = null

  private refresh(): Promise<void> {
    this.refreshing ??= this.call<{ access_token: string; refresh_token: string }>('POST', '/refresh', {
      refresh_token: this.refreshToken,
    }, true)
      .then((t) => this.setTokens(t))
      .finally(() => (this.refreshing = null))
    return this.refreshing
  }
}

// ---------- помощники ----------

// Российские номера в базе храним латинскими буквами-двойниками: валидатор бэка считает длину номера
// в байтах, а кириллическая буква в UTF-8 занимает два байта. Фронт показывает оба варианта одинаково
const CYR_TO_LAT: Record<string, string> = {
  А: 'A', В: 'B', Е: 'E', К: 'K', М: 'M', Н: 'H', О: 'O', Р: 'P', С: 'C', Т: 'T', У: 'Y', Х: 'X',
}
const latinPlate = (plate: string) => [...plate].map((ch) => CYR_TO_LAT[ch] ?? ch).join('')

const log = (msg: string) => console.log(`${new Date().toLocaleTimeString('ru-RU')}  ${msg}`)

// пароль под правила бэка: 8–72 символа, есть буква и цифра
const makePassword = () => `Sim${randomBytes(12).toString('base64url')}7`

// ---------- основной сценарий ----------

async function main() {
  const client = new ApiClient()
  const stamp = Date.now()
  const email = `sim-${stamp}@fleettrack.test`
  const password = makePassword()
  const orgName = `Симулятор ${new Date(stamp).toLocaleTimeString('ru-RU')}`

  log(`регистрирую организацию «${orgName}» на ${API}`)
  try {
    client.setTokens(
      await client.call('POST', '/register', {
        organization_name: orgName,
        // бэк ограничивает имя 35 байтами, а кириллица — 2 байта на букву: длинное имя не пройдёт
        admin_name: 'Диспетчер',
        admin_email: email,
        admin_password: password,
      }),
    )
  } catch (err) {
    console.error(`\nНе удалось зарегистрироваться: ${(err as Error).message}`)
    // fetch бросает TypeError, только когда до сервера не достучаться; ответ с ошибкой — это Error
    if (err instanceof TypeError) {
      console.error(`Бэкенд не отвечает на ${API}: запустите его (make run или make docker-up).`)
    }
    process.exit(1)
  }

  writeFileSync(
    CREDENTIALS_FILE,
    JSON.stringify({ api: API, organization: orgName, email, password, created_at: new Date(stamp).toISOString() }, null, 2),
  )

  const [org] = await client.call<Array<{ id: number; name: string }>>('GET', '/organizations')
  const rules: Array<{ type: AlertType; name: string; threshold: number; severity: string }> = [
    { type: 'SPEED_EXCEEDED', name: 'Превышение 90 км/ч', threshold: 90, severity: 'HIGH' },
    { type: 'LOW_FUEL', name: 'Топливо ниже 15%', threshold: 15, severity: 'MEDIUM' },
    { type: 'DEVICE_OFFLINE', name: 'Трекер молчит минуту', threshold: 1, severity: 'CRITICAL' },
  ]
  for (const rule of rules) {
    await client.call('POST', '/alert-rules', { ...rule, organization_id: org.id, enabled: true })
  }

  // seed от времени: номера и VIN уникальны по всей базе, повторный запуск не должен с ними столкнуться
  const source = new MockFleetSource({ seed: stamp % 2_147_483_647, scale: SCALE })
  const snap = await source.load()

  const driverIds = new Map<number, number>()
  for (const d of snap.drivers) {
    const created = await client.call<{ id: number }>('POST', '/drivers', { name: d.name })
    driverIds.set(d.id, created.id)
  }

  const vehicleIds = new Map<number, number>()
  const deviceIds = new Map<number, number>()
  const plates = new Map<number, string>()
  for (const [i, v] of snap.vehicles.entries()) {
    const plate = latinPlate(v.number_plate)
    const created = await client.call<{ id: number }>('POST', '/vehicles', { vin: v.vin, number_plate: plate, model: v.model })
    vehicleIds.set(v.id, created.id)
    plates.set(v.id, plate)

    const device = await client.call<{ id: number }>('POST', '/devices', { serial_number: `SIM-${stamp}-${i + 1}` })
    await client.call('POST', '/assignments', { device_id: device.id, vehicle_id: created.id })
    // в симуляторе трекер машины N — это устройство 100+N
    deviceIds.set(100 + v.id, device.id)

    if (v.status !== 'IDLE') await client.call('PATCH', `/vehicles/${created.id}`, { status: v.status })
  }

  const tripIds = new Map<number, number>()
  const openTrip = async (trip: Trip) => {
    const created = await client.call<{ id: number }>('POST', '/trips', {
      driver_id: driverIds.get(trip.driver_id),
      vehicle_id: vehicleIds.get(trip.vehicle_id),
    })
    tripIds.set(trip.id, created.id)
  }
  for (const trip of snap.trips.filter((t) => t.status === 'RUNNING')) await openTrip(trip)

  log(`готово: ${snap.drivers.length} водителей, ${snap.vehicles.length} машин с трекерами, ${tripIds.size} рейсов в пути`)
  console.log(`
  Войти во фронт (VITE_FLEET_SOURCE=api, npm run dev):
    почта:  ${email}
    пароль: ${password}
  Те же данные сохранены в web/.fleet-sim.local.json
  Время ×${SCALE}, остановка — Ctrl+C
`)

  // ---------- поток точек ----------

  const stats = { sent: 0, failed: 0, dropped: 0, lastError: '' }
  const queues = new Map<number, { tail: Promise<void>; depth: number }>()
  const silentUntil = new Map<number, number>()
  const lastSpeed = new Map<number, number>()

  // enqueue держит порядок запросов по каждой машине: рейс должен открыться раньше его первой точки
  const enqueue = (vehicleId: number, job: () => Promise<unknown>, droppable = false) => {
    const q = queues.get(vehicleId) ?? { tail: Promise.resolve(), depth: 0 }
    if (droppable && q.depth >= MAX_QUEUE_PER_VEHICLE) {
      stats.dropped++
      return
    }
    q.depth++
    q.tail = q.tail
      .then(job)
      .then(() => void stats.sent++)
      .catch((err: Error) => {
        stats.failed++
        stats.lastError = err.message
      })
      .finally(() => q.depth--)
    queues.set(vehicleId, q)
  }

  const onEvent = (event: FleetEvent) => {
    switch (event.type) {
      case 'telemetry': {
        const p = event.data
        lastSpeed.set(p.vehicle_id, p.speed_kmh)
        if ((silentUntil.get(p.vehicle_id) ?? 0) > Date.now()) return
        enqueue(
          p.vehicle_id,
          () =>
            client.call('POST', '/telemetry', {
              device_id: deviceIds.get(p.device_id),
              vehicle_id: vehicleIds.get(p.vehicle_id),
              lat: p.lat,
              lon: p.lon,
              fuel: p.fuel,
              device_timestamp: p.received_at,
            }),
          true,
        )
        return
      }
      case 'trip': {
        const trip = event.data
        if (!tripIds.has(trip.id) && trip.status === 'RUNNING') {
          enqueue(trip.vehicle_id, () => openTrip(trip))
        } else if (tripIds.has(trip.id) && trip.status === 'SUCCEEDED') {
          enqueue(trip.vehicle_id, () => client.call('PATCH', `/trips/${tripIds.get(trip.id)}`, { status: 'SUCCEEDED' }))
          log(`рейс ${plates.get(trip.vehicle_id)} завершён, ${Math.round(trip.distance_km)} км`)
        }
        return
      }
      case 'vehicle': {
        const status: VehicleStatus = event.data.status
        enqueue(event.data.id, () => client.call('PATCH', `/vehicles/${vehicleIds.get(event.data.id)}`, { status }))
        return
      }
      case 'alert':
        // алерты симулятора не отправляем: на живых данных их поднимает движок правил бэка
        return
    }
  }

  // первая точка каждой машины — сразу, чтобы карта не ждала
  for (const p of snap.positions) onEvent({ type: 'telemetry', data: p })
  const unsubscribe = source.subscribe({ onEvent })

  const silenceOnce = () => {
    // связь «теряет» машина, которая сейчас едет, — на стоянке это никто бы не заметил
    const moving = [...lastSpeed].filter(([id, speed]) => speed > 0 && (silentUntil.get(id) ?? 0) < Date.now())
    const victim = moving.length ? snap.vehicles.find((v) => v.id === moving[Math.floor(Math.random() * moving.length)][0]) : undefined
    if (!victim) return
    silentUntil.set(victim.id, Date.now() + SILENCE_FOR_S * 1000)
    log(`трекер ${plates.get(victim.id)} молчит ${SILENCE_FOR_S} с — бэк должен поднять DEVICE_OFFLINE`)
  }
  let silenceTimer: ReturnType<typeof setInterval> | undefined
  const firstSilence = setTimeout(() => {
    silenceOnce()
    silenceTimer = setInterval(silenceOnce, SILENCE_EVERY_S * 1000)
  }, FIRST_SILENCE_S * 1000)

  const statsTimer = setInterval(() => {
    const err = stats.lastError ? `, последняя ошибка: ${stats.lastError}` : ''
    log(`отправлено ${stats.sent}, ошибок ${stats.failed}, выброшено из-за отставания ${stats.dropped}${err}`)
    stats.lastError = ''
  }, 10_000)

  process.on('SIGINT', async () => {
    log('останавливаюсь, дожидаюсь отправки очереди…')
    unsubscribe()
    clearTimeout(firstSilence)
    clearInterval(silenceTimer)
    clearInterval(statsTimer)
    await Promise.all([...queues.values()].map((q) => q.tail))
    log(`итого отправлено ${stats.sent}, ошибок ${stats.failed}`)
    process.exit(0)
  })
}

main().catch((err: unknown) => {
  console.error(err)
  process.exit(1)
})
