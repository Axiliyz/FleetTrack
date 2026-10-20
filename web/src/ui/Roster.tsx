import { useMemo, useState } from 'react'
import { formatInt } from '../lib/format'
import { useFleet } from '../state/fleet'
import { CONDITION_LABEL, conditionsById, runningTrip, standings } from '../state/selectors'
import { Plate } from './Plate'
import { Segments } from './Segments'

type Tab = 'vehicles' | 'drivers'

export function Roster() {
  const [tab, setTab] = useState<Tab>('vehicles')
  const open = useFleet((s) => s.rosterOpen)
  const vehicleCount = useFleet((s) => Object.keys(s.vehicles).length)
  const driverCount = useFleet((s) => Object.keys(s.drivers).length)

  return (
    <aside id="roster" className="roster" data-open={open || undefined} aria-label="Парк">
      <div className="tabs" role="tablist">
        <button type="button" role="tab" aria-selected={tab === 'vehicles'} onClick={() => setTab('vehicles')}>
          Машины <span className="tab-count">{vehicleCount}</span>
        </button>
        <button type="button" role="tab" aria-selected={tab === 'drivers'} onClick={() => setTab('drivers')}>
          Рейтинг смены <span className="tab-count">{driverCount}</span>
        </button>
      </div>

      <div className="roster-body" role="tabpanel">
        {tab === 'vehicles' ? <VehicleList /> : <DriverStandings />}
      </div>

      <p className="hint">
        <kbd>P</kbd> или <kbd>Пробел</kbd> пауза, <kbd>[</kbd> <kbd>]</kbd> соседняя машина, <kbd>F</kbd> следить, <kbd>Esc</kbd> снять выбор
      </p>
    </aside>
  )
}

function VehicleList() {
  const vehicles = useFleet((s) => s.vehicles)
  const alerts = useFleet((s) => s.alerts)
  const trips = useFleet((s) => s.trips)
  const drivers = useFleet((s) => s.drivers)
  const telemetry = useFleet((s) => s.telemetry)
  const selectedId = useFleet((s) => s.selectedId)
  const fuelThreshold = useFleet((s) => s.rules.find((r) => r.type === 'LOW_FUEL')?.threshold)
  const select = useFleet((s) => s.select)
  const setRosterOpen = useFleet((s) => s.setRosterOpen)

  const conditions = useMemo(() => conditionsById(vehicles, alerts), [vehicles, alerts])
  const list = useMemo(() => Object.values(vehicles).sort((a, b) => a.id - b.id), [vehicles])

  return (
    <ul className="units">
      {list.map((v) => {
        const condition = conditions.get(v.id) ?? 'idle'
        const trip = runningTrip(trips, v.id)
        const driver = trip ? drivers[trip.driver_id] : undefined
        const point = telemetry[v.id]
        return (
          <li key={v.id}>
            <button
              type="button"
              className="unit-row"
              data-cond={condition}
              aria-current={selectedId === v.id || undefined}
              onClick={() => {
                select(v.id, { fly: true })
                setRosterOpen(false)
              }}
            >
              <span className="lamp" aria-hidden />
              <Plate value={v.number_plate} size="sm" />
              <span className="unit-row-main">
                <span className="unit-row-driver">{driver?.name ?? CONDITION_LABEL[condition]}</span>
                <span className="unit-row-model">{v.model}</span>
              </span>
              <span className="unit-row-speed">
                {point ? Math.round(point.speed_kmh) : '—'}
                <small>км/ч</small>
              </span>
              <Segments
                value={point?.fuel ?? 0}
                max={100}
                count={8}
                threshold={fuelThreshold}
                danger="below"
                label="Топливо"
              />
            </button>
          </li>
        )
      })}
    </ul>
  )
}

function DriverStandings() {
  const drivers = useFleet((s) => s.drivers)
  const trips = useFleet((s) => s.trips)
  const alerts = useFleet((s) => s.alerts)
  const clock = useFleet((s) => s.clock)
  const vehicles = useFleet((s) => s.vehicles)
  const select = useFleet((s) => s.select)

  // рейтинг пересчитываем раз в сим-минуту: trips меняются с каждой точкой телеметрии
  const minute = Math.floor(clock / 60_000)
  const rows = useMemo(() => standings(drivers, trips, alerts, minute * 60_000), [drivers, alerts, minute]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <>
      <p className="roster-note">Очки — километры за смену минус 50 за каждое превышение скорости.</p>
      <ol className="standings">
        {rows.map((row, i) => {
          const vehicle = row.vehicleId !== null ? vehicles[row.vehicleId] : undefined
          return (
            <li key={row.driver.id} data-podium={i < 3 || undefined}>
              <span className="standing-rank">{i + 1}</span>
              <span className="standing-main">
                <span className="standing-name">{row.driver.name}</span>
                <span className="standing-meta">
                  {formatInt(row.km)} км, рейсов {row.trips}
                  {row.violations > 0 && <span className="standing-violations">, превышений {row.violations}</span>}
                </span>
              </span>
              {vehicle ? (
                <button type="button" className="standing-vehicle" onClick={() => select(vehicle.id, { fly: true })}>
                  <Plate value={vehicle.number_plate} size="sm" />
                </button>
              ) : (
                <span className="standing-rest">отдыхает</span>
              )}
              <b className="standing-score">{formatInt(row.score)}</b>
            </li>
          )
        })}
      </ol>
    </>
  )
}
