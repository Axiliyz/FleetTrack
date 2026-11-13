import { useState } from 'react'
import { PAGE_SIZE } from '../../api/query'
import { TRIP_STATUS_LABEL, optionsOf } from '../../lib/labels'
import { useListParams } from '../../lib/useListParams'
import { useAccess, useClaims } from '../../state/session'
import type { Trip } from '../../types'
import { FilterBar, FilterDate, FilterSelect } from '../../ui/Filters'
import { Page, Pager, QueryView } from '../../ui/Page'
import { useDriverIndex } from '../drivers/api'
import { useVehicleIndex } from '../vehicles/api'
import { TRIP_FILTER_KEYS, useTripPage } from './api'
import { DeleteTripDialog, StartTripDialog, TripStatusDialog } from './TripDialogs'
import { TripRows } from './TripTable'

type Dialog = { kind: 'start' } | { kind: 'status'; trip: Trip } | { kind: 'delete'; trip: Trip }

export function TripsPage() {
  const { manageFleet } = useAccess()
  const claims = useClaims()
  const list = useListParams(TRIP_FILTER_KEYS)
  const query = useTripPage(list.filter, list.offset)
  const vehicles = useVehicleIndex()
  const drivers = useDriverIndex()
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const close = () => setDialog(null)
  // водителю бэк сам отдаёт только его рейсы, фильтр по водителю ему ни к чему
  const isDriver = claims?.role === 'DRIVER'

  return (
    <Page
      title={isDriver ? 'Мои рейсы' : 'Рейсы'}
      lead="Пробег и скорость бэк считает по точкам трекера, пока рейс в пути."
      actions={
        manageFleet && (
          <button type="button" className="btn btn-primary" onClick={() => setDialog({ kind: 'start' })}>
            Новый рейс
          </button>
        )
      }
    >
      <FilterBar active={list.active} onReset={list.reset}>
        <FilterSelect
          label="Статус"
          value={list.filter.status}
          onChange={(v) => list.set('status', v)}
          options={optionsOf(TRIP_STATUS_LABEL)}
        />
        <FilterSelect
          label="Машина"
          value={list.filter.vehicle_id}
          onChange={(v) => list.set('vehicle_id', v)}
          options={Object.values(vehicles.data ?? {}).map((v) => ({ value: String(v.id), label: v.number_plate }))}
        />
        {!isDriver && (
          <FilterSelect
            label="Водитель"
            value={list.filter.driver_id}
            onChange={(v) => list.set('driver_id', v)}
            options={Object.values(drivers.data ?? {}).map((d) => ({ value: String(d.id), label: d.name }))}
          />
        )}
        <FilterDate label="Начат с" value={list.filter.started_from} onChange={(v) => list.set('started_from', v)} />
        <FilterDate label="по" value={list.filter.started_to} onChange={(v) => list.set('started_to', v)} />
      </FilterBar>

      <QueryView
        query={query}
        isEmpty={(page) => page.rows.length === 0 && list.offset === 0}
        empty={list.active ? 'По фильтру рейсов нет.' : 'Рейсов пока не было.'}
      >
        {(page) => (
          <>
            <TripRows
              trips={page.rows}
              actions={
                manageFleet
                  ? {
                      onStatus: (trip) => setDialog({ kind: 'status', trip }),
                      onDelete: (trip) => setDialog({ kind: 'delete', trip }),
                    }
                  : undefined
              }
            />
            <Pager
              offset={list.offset}
              pageSize={PAGE_SIZE}
              shown={page.rows.length}
              hasNext={page.hasNext}
              onChange={list.setOffset}
            />
          </>
        )}
      </QueryView>

      {dialog?.kind === 'start' && <StartTripDialog onClose={close} />}
      {dialog?.kind === 'status' && <TripStatusDialog trip={dialog.trip} onClose={close} />}
      {dialog?.kind === 'delete' && <DeleteTripDialog trip={dialog.trip} onClose={close} />}
    </Page>
  )
}
