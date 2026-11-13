import { Link } from 'react-router'
import { formatDateTime, formatDuration, formatInt } from '../../lib/format'
import { TRIP_STATUS_LABEL, TRIP_STATUS_TONE } from '../../lib/labels'
import type { Trip } from '../../types'
import { QueryView, Status } from '../../ui/Page'
import { Plate } from '../../ui/Plate'
import { useDriverIndex } from '../drivers/api'
import { useVehicleIndex } from '../vehicles/api'
import { useTripPage, type TripFilter } from './api'

const EMPTY_FILTER: TripFilter = { status: '', vehicle_id: '', driver_id: '', started_from: '', started_to: '' }
const RECENT_ROWS = 10

// RecentTrips — рейсы машины или водителя на их странице: последние десять, без листания
export function RecentTrips({ filter }: { filter: Partial<TripFilter> }) {
  const query = useTripPage({ ...EMPTY_FILTER, ...filter }, 0)
  return (
    <QueryView query={query} isEmpty={(page) => page.rows.length === 0} empty="Рейсов пока не было.">
      {(page) => (
        <TripRows
          trips={page.rows.slice(0, RECENT_ROWS)}
          showVehicle={!filter.vehicle_id}
          showDriver={!filter.driver_id}
        />
      )}
    </QueryView>
  )
}

export interface TripActions {
  onStatus: (trip: Trip) => void
  onDelete: (trip: Trip) => void
}

// showVehicle и showDriver: на странице машины колонка машины повторяла бы одно и то же, как и водителя
export function TripRows({
  trips,
  actions,
  showVehicle = true,
  showDriver = true,
}: {
  trips: Trip[]
  actions?: TripActions
  showVehicle?: boolean
  showDriver?: boolean
}) {
  const vehicles = useVehicleIndex().data ?? {}
  const drivers = useDriverIndex().data ?? {}
  const now = Date.now()

  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th scope="col">Рейс</th>
            {showVehicle && <th scope="col">Машина</th>}
            {showDriver && <th scope="col">Водитель</th>}
            <th scope="col">Начат</th>
            <th scope="col" className="num">В пути</th>
            <th scope="col" className="num">Пробег</th>
            <th scope="col" className="num">Средняя / макс.</th>
            <th scope="col">Статус</th>
            {actions && <th scope="col" aria-label="Действия" />}
          </tr>
        </thead>
        <tbody>
          {trips.map((t) => {
            const vehicle = vehicles[t.vehicle_id]
            const driver = drivers[t.driver_id]
            const end = t.ended_at ? Date.parse(t.ended_at) : now
            return (
              <tr key={t.id}>
                <td>
                  <Link to={`/trips/${t.id}`} className="row-link">
                    №{t.id}
                  </Link>
                </td>
                {showVehicle && (
                  <td>
                    {vehicle ? (
                      <Link to={`/vehicles/${vehicle.id}`} aria-label={`Машина ${vehicle.number_plate}`}>
                        <Plate value={vehicle.number_plate} size="sm" />
                      </Link>
                    ) : (
                      <span className="dim">машина №{t.vehicle_id}</span>
                    )}
                  </td>
                )}
                {showDriver && (
                  <td className="nowrap">
                    {driver ? (
                      <Link to={`/drivers/${driver.id}`}>{driver.name}</Link>
                    ) : (
                      <span className="dim">№{t.driver_id}</span>
                    )}
                  </td>
                )}
                <td className="nowrap">{formatDateTime(t.started_at)}</td>
                <td className="num">{formatDuration(end - Date.parse(t.started_at))}</td>
                <td className="num">{formatInt(t.distance_km)} км</td>
                <td className="num">
                  {Math.round(t.avg_speed_kmh)} / {Math.round(t.max_speed_kmh)} км/ч
                </td>
                <td>
                  <Status tone={TRIP_STATUS_TONE[t.status]}>{TRIP_STATUS_LABEL[t.status]}</Status>
                </td>
                {actions && (
                  <td className="row-actions">
                    <button type="button" className="btn btn-sm" onClick={() => actions.onStatus(t)}>
                      Статус
                    </button>
                    <button type="button" className="btn btn-sm btn-quiet" onClick={() => actions.onDelete(t)}>
                      Удалить
                    </button>
                  </td>
                )}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
