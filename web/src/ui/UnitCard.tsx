import { useMemo } from 'react'
import { formatDuration, formatInt, formatTime, formatTimeSec, initials } from '../lib/format'
import { acknowledgeAlert } from '../state/connect'
import { useFleet } from '../state/fleet'
import { ALERT_LABEL, CONDITION_LABEL, conditionOf, openAlertsOf, runningTrip, standings } from '../state/selectors'
import { Plate } from './Plate'
import { Segments } from './Segments'

const SPEED_SCALE_MAX = 130

export function UnitCard() {
  const id = useFleet((s) => s.selectedId)
  if (id === null) return null
  return <UnitCardBody key={id} vehicleId={id} />
}

function UnitCardBody({ vehicleId }: { vehicleId: number }) {
  const vehicle = useFleet((s) => s.vehicles[vehicleId])
  const point = useFleet((s) => s.telemetry[vehicleId])
  const alerts = useFleet((s) => s.alerts)
  const trips = useFleet((s) => s.trips)
  const drivers = useFleet((s) => s.drivers)
  const rules = useFleet((s) => s.rules)
  const clock = useFleet((s) => s.clock)
  const following = useFleet((s) => s.followId === vehicleId)
  const select = useFleet((s) => s.select)
  const setFollow = useFleet((s) => s.setFollow)

  const open = useMemo(() => openAlertsOf(alerts, vehicleId), [alerts, vehicleId])
  const trip = runningTrip(trips, vehicleId)
  const driver = trip ? drivers[trip.driver_id] : undefined
  const minute = Math.floor(clock / 60_000)
  const standing = useMemo(() => {
    if (!driver) return null
    const rows = standings(drivers, trips, alerts, minute * 60_000)
    const place = rows.findIndex((r) => r.driver.id === driver.id)
    return { ...rows[place], place: place + 1, of: rows.length }
    // trips намеренно не в зависимостях: пересчёт раз в сим-минуту
  }, [driver, drivers, alerts, minute]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!vehicle) return null

  const condition = conditionOf(vehicle, open)
  const speedLimit = rules.find((r) => r.type === 'SPEED_EXCEEDED')?.threshold
  const fuelLimit = rules.find((r) => r.type === 'LOW_FUEL')?.threshold
  const speed = point?.speed_kmh ?? 0
  const fuel = point?.fuel ?? null
  const silentFor = point ? clock - Date.parse(point.received_at) : 0

  return (
    <section className="card" data-cond={condition} aria-label={`Машина ${vehicle.number_plate}`}>
      <header className="card-head">
        <Plate value={vehicle.number_plate} size="lg" />
        <button type="button" className="card-close" aria-label="Закрыть карточку" onClick={() => select(null)}>
          ×
        </button>
        <p className="card-model">{vehicle.model}</p>
        <p className="card-state">
          <span className="lamp" aria-hidden />
          {CONDITION_LABEL[condition]}
          {condition === 'offline' && point && <>, последняя точка {formatDuration(silentFor)} назад</>}
        </p>
      </header>

      <div className="gauges">
        <div className="gauge">
          <span className="gauge-value" data-over={speedLimit !== undefined && speed > speedLimit || undefined}>
            {Math.round(speed)}
            <small>км/ч</small>
          </span>
          <Segments value={speed} max={SPEED_SCALE_MAX} count={26} threshold={speedLimit} danger="above" label="Скорость" />
          {speedLimit !== undefined && <span className="gauge-note">Порог {speedLimit} км/ч</span>}
        </div>
        <div className="gauge">
          <span className="gauge-value" data-over={fuel !== null && fuelLimit !== undefined && fuel < fuelLimit || undefined}>
            {fuel === null ? '—' : Math.round(fuel)}
            <small>% топлива</small>
          </span>
          <Segments value={fuel ?? 0} max={100} count={20} threshold={fuelLimit} danger="below" label="Топливо" />
          {fuelLimit !== undefined && <span className="gauge-note">Порог {fuelLimit}%</span>}
        </div>
      </div>

      {open.length > 0 && (
        <div className="card-section">
          <h3>Тревоги</h3>
          <ul className="card-alerts">
            {open.map((a) => (
              <li key={a.id} data-severity={a.severity} data-status={a.status}>
                <span className="card-alert-text">
                  <b>{ALERT_LABEL[a.type]}</b>
                  <span>
                    {a.message}, с {formatTime(a.created_at)}
                  </span>
                </span>
                {a.status === 'FIRED' ? (
                  <button type="button" className="btn btn-alarm" onClick={() => void acknowledgeAlert(a.id)}>
                    Принять
                  </button>
                ) : (
                  <span className="card-alert-ack">Принято в {formatTime(a.acknowledged_at ?? a.created_at)}</span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="card-section">
        <h3>Водитель</h3>
        {driver ? (
          <div className="driver">
            <span className="driver-avatar" aria-hidden>
              {initials(driver.name)}
            </span>
            <span className="driver-main">
              <b>{driver.name}</b>
              {standing && (
                <span>
                  {standing.place} место из {standing.of}, {formatInt(standing.km)} км за смену
                  {standing.violations > 0 && `, превышений ${standing.violations}`}
                </span>
              )}
            </span>
          </div>
        ) : (
          <p className="card-empty">Никто не назначен: машина без рейса.</p>
        )}
      </div>

      {trip && (
        <div className="card-section">
          <h3>Рейс №{trip.id}</h3>
          <dl className="stats">
            <div>
              <dt>В пути</dt>
              <dd>{formatDuration(clock - Date.parse(trip.started_at))}</dd>
            </div>
            <div>
              <dt>Пройдено</dt>
              <dd>{formatInt(trip.distance_km)} км</dd>
            </div>
            <div>
              <dt>Средняя</dt>
              <dd>{Math.round(trip.avg_speed_kmh)} км/ч</dd>
            </div>
            <div>
              <dt>Максимальная</dt>
              <dd data-over={speedLimit !== undefined && trip.max_speed_kmh > speedLimit || undefined}>
                {Math.round(trip.max_speed_kmh)} км/ч
              </dd>
            </div>
          </dl>
        </div>
      )}

      <div className="card-actions">
        <button
          type="button"
          className="btn"
          aria-pressed={following}
          onClick={() => setFollow(following ? null : vehicleId)}
        >
          {following ? 'Перестать следить' : 'Следить за машиной'}
        </button>
        <button type="button" className="btn btn-quiet" onClick={() => select(vehicleId, { fly: true })}>
          Показать на карте
        </button>
      </div>

      <p className="card-meta">
        VIN {vehicle.vin}
        <br />
        Трекер №{point?.device_id ?? '—'}, точка в {point ? formatTimeSec(point.received_at) : '—'}
      </p>
    </section>
  )
}
