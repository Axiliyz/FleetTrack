import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { formatDateTime, formatDuration, formatInt } from '../../lib/format'
import { TRIP_STATUS_LABEL, TRIP_STATUS_TONE } from '../../lib/labels'
import { useIdParam } from '../../lib/useIdParam'
import { useAccess } from '../../state/session'
import type { Trip } from '../../types'
import { Page, QueryView, Status } from '../../ui/Page'
import { Plate } from '../../ui/Plate'
import { useDriverIndex } from '../drivers/api'
import { useTrack } from '../telemetry/api'
import { TrackView, useSpeedLimit } from '../telemetry/TrackView'
import { useVehicleIndex } from '../vehicles/api'
import { useTrip } from './api'
import { DeleteTripDialog, TripStatusDialog } from './TripDialogs'

export function TripPage() {
  const id = useIdParam()
  return <TripDetail key={id} id={id ?? 0} />
}

function TripDetail({ id }: { id: number }) {
  const query = useTrip(id)
  if (query.data) return <TripView trip={query.data} />
  return (
    <Page title={`Рейс №${id}`}>
      <QueryView query={query}>{() => null}</QueryView>
    </Page>
  )
}

function TripView({ trip }: { trip: Trip }) {
  const { manageFleet } = useAccess()
  const navigate = useNavigate()
  const vehicle = useVehicleIndex().data?.[trip.vehicle_id]
  const driver = useDriverIndex().data?.[trip.driver_id]
  const speedLimit = useSpeedLimit()
  const track = useTrack({ tripId: trip.id })
  const [dialog, setDialog] = useState<'status' | 'delete' | null>(null)
  const close = () => setDialog(null)

  const end = trip.ended_at ? Date.parse(trip.ended_at) : Date.now()

  return (
    <Page
      title={`Рейс №${trip.id}`}
      lead={
        <>
          <Status tone={TRIP_STATUS_TONE[trip.status]}>{TRIP_STATUS_LABEL[trip.status]}</Status>
          <span>
            {formatDateTime(trip.started_at)}
            {trip.ended_at && ` — ${formatDateTime(trip.ended_at)}`}
          </span>
        </>
      }
      actions={
        manageFleet && (
          <>
            <button type="button" className="btn" onClick={() => setDialog('status')}>
              Сменить статус
            </button>
            <button type="button" className="btn btn-quiet" onClick={() => setDialog('delete')}>
              Удалить
            </button>
          </>
        )
      }
    >
      <nav className="crumbs" aria-label="Путь">
        <Link to="/trips">Рейсы</Link>
      </nav>

      <section className="panel">
        <dl className="stats stats-row">
          <div>
            <dt>Машина</dt>
            <dd>
              {vehicle ? (
                <Link to={`/vehicles/${vehicle.id}`} aria-label={`Машина ${vehicle.number_plate}`}>
                  <Plate value={vehicle.number_plate} size="sm" />
                </Link>
              ) : (
                `№${trip.vehicle_id}`
              )}
            </dd>
          </div>
          <div>
            <dt>Водитель</dt>
            <dd>{driver ? <Link to={`/drivers/${driver.id}`}>{driver.name}</Link> : `№${trip.driver_id}`}</dd>
          </div>
          <div>
            <dt>В пути</dt>
            <dd>{formatDuration(end - Date.parse(trip.started_at))}</dd>
          </div>
          <div>
            <dt>Пробег</dt>
            <dd>{formatInt(trip.distance_km)} км</dd>
          </div>
          <div>
            <dt>Средняя</dt>
            <dd>{Math.round(trip.avg_speed_kmh)} км/ч</dd>
          </div>
          <div>
            <dt>Максимальная</dt>
            <dd data-over={(speedLimit !== undefined && trip.max_speed_kmh > speedLimit) || undefined}>
              {Math.round(trip.max_speed_kmh)} км/ч
            </dd>
          </div>
          <div>
            <dt>Точек</dt>
            <dd>{formatInt(trip.telemetry_count)}</dd>
          </div>
        </dl>
      </section>

      <section className="panel" aria-labelledby="trip-track-title">
        <header className="panel-head">
          <h2 id="trip-track-title">Трек</h2>
        </header>
        <QueryView query={track} empty="Трекер ещё не присылал точек по этому рейсу.">
          {(points) => <TrackView points={points} fitKey={`trip:${trip.id}`} />}
        </QueryView>
      </section>

      {dialog === 'status' && <TripStatusDialog trip={trip} onClose={close} />}
      {dialog === 'delete' && (
        <DeleteTripDialog trip={trip} onClose={close} onDeleted={() => navigate('/trips', { replace: true })} />
      )}
    </Page>
  )
}
