import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { formatDateTime, formatInt } from '../../lib/format'
import { useIdParam } from '../../lib/useIdParam'
import { useAccess } from '../../state/session'
import type { Driver } from '../../types'
import { Page, QueryView } from '../../ui/Page'
import { useTripPage } from '../trips/api'
import { RecentTrips } from '../trips/TripTable'
import { useDriver } from './api'
import { DeleteDriverDialog, RenameDriverDialog } from './DriverDialogs'

export function DriverPage() {
  const id = useIdParam()
  return <DriverDetail key={id} id={id ?? 0} />
}

function DriverDetail({ id }: { id: number }) {
  const query = useDriver(id)
  if (query.data) return <DriverView driver={query.data} />
  return (
    <Page title="Водитель">
      <QueryView query={query}>{() => null}</QueryView>
    </Page>
  )
}

function DriverView({ driver }: { driver: Driver }) {
  const { manageFleet } = useAccess()
  const navigate = useNavigate()
  const [dialog, setDialog] = useState<'rename' | 'delete' | null>(null)
  const close = () => setDialog(null)
  const totals = useDriverTotals(driver.id)

  return (
    <Page
      title={driver.name}
      lead={`В парке с ${formatDateTime(driver.created_at)}`}
      actions={
        manageFleet && (
          <>
            <button type="button" className="btn" onClick={() => setDialog('rename')}>
              Переименовать
            </button>
            <button type="button" className="btn btn-quiet" onClick={() => setDialog('delete')}>
              Удалить
            </button>
          </>
        )
      }
    >
      <nav className="crumbs" aria-label="Путь">
        <Link to="/drivers">Водители</Link>
      </nav>

      {totals && (
        <section className="panel">
          <dl className="stats stats-row">
            <div>
              <dt>Рейсов</dt>
              <dd>
                {formatInt(totals.trips)}
                {totals.partial && '+'}
              </dd>
            </div>
            <div>
              <dt>Пробег</dt>
              <dd>
                {formatInt(totals.km)} км{totals.partial && '+'}
              </dd>
            </div>
            <div>
              <dt>Сейчас</dt>
              <dd>{totals.running ? <Link to={`/trips/${totals.running}`}>в рейсе №{totals.running}</Link> : 'отдыхает'}</dd>
            </div>
          </dl>
        </section>
      )}

      <section className="panel" aria-labelledby="driver-trips-title">
        <header className="panel-head">
          <h2 id="driver-trips-title">Последние рейсы</h2>
          <Link to={`/trips?driver_id=${driver.id}`} className="btn btn-sm btn-quiet">
            Все рейсы водителя
          </Link>
        </header>
        <RecentTrips filter={{ driver_id: String(driver.id) }} />
      </section>

      {dialog === 'rename' && <RenameDriverDialog driver={driver} onClose={close} />}
      {dialog === 'delete' && (
        <DeleteDriverDialog driver={driver} onClose={close} onDeleted={() => navigate('/drivers', { replace: true })} />
      )}
    </Page>
  )
}

// useDriverTotals — итоги по первой странице рейсов. Агрегатов бэк не отдаёт;
// если рейсов больше страницы, итог помечается «+», а не выдаётся за точный
function useDriverTotals(driverId: number) {
  const query = useTripPage(
    { status: '', vehicle_id: '', driver_id: String(driverId), started_from: '', started_to: '' },
    0,
  )
  if (!query.data) return null
  const { rows, hasNext } = query.data
  // отменённые рейсы не считаем — как в рейтинге смены на карте
  const counted = rows.filter((t) => t.status !== 'CANCELLED')
  return {
    trips: counted.length,
    km: counted.reduce((sum, t) => sum + t.distance_km, 0),
    running: rows.find((t) => t.status === 'RUNNING')?.id ?? null,
    partial: hasNext,
  }
}
