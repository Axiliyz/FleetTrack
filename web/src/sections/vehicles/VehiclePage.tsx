import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { formatDecimal, formatTimeSec } from '../../lib/format'
import { VEHICLE_STATUS_COND, VEHICLE_STATUS_LABEL } from '../../lib/labels'
import { useIdParam } from '../../lib/useIdParam'
import { useAccess } from '../../state/session'
import type { TelemetryEvent, Vehicle } from '../../types'
import { Page, QueryView, Status } from '../../ui/Page'
import { Plate } from '../../ui/Plate'
import { AssignDeviceDialog } from '../devices/DeviceDialogs'
import { useTrack } from '../telemetry/api'
import { ClearTrackDialog, DeletePointDialog, SendPointDialog } from '../telemetry/PointDialogs'
import { TrackView } from '../telemetry/TrackView'
import { RecentTrips } from '../trips/TripTable'
import { useVehicle } from './api'
import { DeleteVehicleDialog, EditVehicleDialog } from './VehicleDialogs'

const PERIODS = [
  { hours: 1, label: 'Час' },
  { hours: 6, label: '6 часов' },
  { hours: 24, label: 'Сутки' },
  { hours: 24 * 7, label: 'Неделя' },
]

const periodFrom = (hours: number) => ({ hours, from: new Date(Date.now() - hours * 3_600_000).toISOString() })

// в таблице — последние точки; весь период виден на карте и графике
const TABLE_POINTS = 20

type Dialog =
  | { kind: 'edit' }
  | { kind: 'delete' }
  | { kind: 'assign' }
  | { kind: 'send' }
  | { kind: 'clear' }
  | { kind: 'deletePoint'; point: TelemetryEvent }

export function VehiclePage() {
  const id = useIdParam()
  // key: другая машина — чистое состояние страницы
  return <VehicleDetail key={id} id={id ?? 0} />
}

function VehicleDetail({ id }: { id: number }) {
  const query = useVehicle(id)
  if (query.data) return <VehicleView vehicle={query.data} />
  return (
    <Page title="Машина">
      <QueryView query={query}>{() => null}</QueryView>
    </Page>
  )
}

function VehicleView({ vehicle }: { vehicle: Vehicle }) {
  const { manageFleet } = useAccess()
  const navigate = useNavigate()
  // начало периода фиксируется при выборе: автообновление дописывает новые точки в тот же трек,
  // а не перезапрашивает его с новым началом каждую минуту
  const [period, setPeriod] = useState(() => periodFrom(24))
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const close = () => setDialog(null)

  const track = useTrack({ vehicleId: vehicle.id, from: period.from })
  const last = track.data?.at(-1)

  return (
    <Page
      title={
        <span className="title-plate">
          <Plate value={vehicle.number_plate} size="lg" /> {vehicle.model}
        </span>
      }
      lead={
        <>
          <Status cond={VEHICLE_STATUS_COND[vehicle.status]}>{VEHICLE_STATUS_LABEL[vehicle.status]}</Status>
          <span className="mono">VIN {vehicle.vin}</span>
        </>
      }
      actions={
        manageFleet && (
          <>
            <button type="button" className="btn" onClick={() => setDialog({ kind: 'edit' })}>
              Изменить
            </button>
            <button type="button" className="btn btn-quiet" onClick={() => setDialog({ kind: 'delete' })}>
              Удалить
            </button>
          </>
        )
      }
    >
      <nav className="crumbs" aria-label="Путь">
        <Link to="/vehicles">Машины</Link>
      </nav>

      <section className="panel" aria-labelledby="tracker-title">
        <header className="panel-head">
          <h2 id="tracker-title">Трекер</h2>
          {manageFleet && (
            <button type="button" className="btn btn-sm" onClick={() => setDialog({ kind: 'assign' })}>
              Привязать трекер
            </button>
          )}
        </header>
        <p className="panel-text">
          {last ? (
            <>
              Последняя точка от трекера <Link to={`/devices?id=${last.device_id}`}>№{last.device_id}</Link> в{' '}
              {formatTimeSec(last.received_at)}.
            </>
          ) : (
            'За выбранный период точек нет. Новый трекер можно привязать только к свободной машине.'
          )}
        </p>
      </section>

      <section className="panel" aria-labelledby="track-title">
        <header className="panel-head">
          <h2 id="track-title">Телеметрия</h2>
          <div className="segmented" role="group" aria-label="Период">
            {PERIODS.map((p) => (
              <button
                key={p.hours}
                type="button"
                aria-pressed={period.hours === p.hours}
                onClick={() => setPeriod(periodFrom(p.hours))}
              >
                {p.label}
              </button>
            ))}
          </div>
          {manageFleet && (
            <div className="panel-actions">
              <button type="button" className="btn btn-sm" onClick={() => setDialog({ kind: 'send' })}>
                Отправить точку
              </button>
              <button type="button" className="btn btn-sm btn-quiet" onClick={() => setDialog({ kind: 'clear' })}>
                Очистить историю
              </button>
            </div>
          )}
        </header>
        <QueryView query={track} empty="За выбранный период трекер ничего не присылал.">
          {(points) => (
            <>
              <TrackView points={points} fitKey={`${vehicle.id}:${period.from}`} />
              <PointTable
                points={points.slice(-TABLE_POINTS).reverse()}
                onDelete={manageFleet ? (point) => setDialog({ kind: 'deletePoint', point }) : undefined}
              />
            </>
          )}
        </QueryView>
      </section>

      <section className="panel" aria-labelledby="trips-title">
        <header className="panel-head">
          <h2 id="trips-title">Рейсы</h2>
          <Link to={`/trips?vehicle_id=${vehicle.id}`} className="btn btn-sm btn-quiet">
            Все рейсы машины
          </Link>
        </header>
        <RecentTrips filter={{ vehicle_id: String(vehicle.id) }} />
      </section>

      {dialog?.kind === 'edit' && <EditVehicleDialog vehicle={vehicle} onClose={close} />}
      {dialog?.kind === 'delete' && (
        <DeleteVehicleDialog vehicle={vehicle} onClose={close} onDeleted={() => navigate('/vehicles', { replace: true })} />
      )}
      {dialog?.kind === 'assign' && <AssignDeviceDialog vehicle={vehicle} onClose={close} />}
      {dialog?.kind === 'send' && <SendPointDialog vehicle={vehicle} last={last} onClose={close} />}
      {dialog?.kind === 'clear' && <ClearTrackDialog vehicle={vehicle} onClose={close} />}
      {dialog?.kind === 'deletePoint' && <DeletePointDialog point={dialog.point} onClose={close} />}
    </Page>
  )
}

function PointTable({ points, onDelete }: { points: TelemetryEvent[]; onDelete?: (p: TelemetryEvent) => void }) {
  return (
    <div className="table-wrap">
      <table className="table">
        <caption>Последние точки</caption>
        <thead>
          <tr>
            <th scope="col">Время</th>
            <th scope="col" className="num">Скорость</th>
            <th scope="col" className="num">Топливо</th>
            <th scope="col">Координаты</th>
            <th scope="col">Рейс</th>
            {onDelete && <th scope="col" aria-label="Действия" />}
          </tr>
        </thead>
        <tbody>
          {points.map((p) => (
            <tr key={p.telemetry_id}>
              <td>{formatTimeSec(p.received_at)}</td>
              <td className="num">{Math.round(p.speed_kmh)} км/ч</td>
              <td className="num">{p.fuel === null ? '—' : `${formatDecimal(p.fuel)}%`}</td>
              <td className="mono">
                {p.lat.toFixed(5)}, {p.lon.toFixed(5)}
              </td>
              <td>{p.trip_id ? <Link to={`/trips/${p.trip_id}`}>№{p.trip_id}</Link> : '—'}</td>
              {onDelete && (
                <td className="row-actions">
                  <button type="button" className="btn btn-sm btn-quiet" onClick={() => onDelete(p)}>
                    Удалить
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
