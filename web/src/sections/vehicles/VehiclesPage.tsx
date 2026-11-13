import { useState } from 'react'
import { Link } from 'react-router'
import { PAGE_SIZE } from '../../api/query'
import { VEHICLE_STATUS_COND, VEHICLE_STATUS_LABEL, optionsOf } from '../../lib/labels'
import { useListParams } from '../../lib/useListParams'
import { normalizePlate } from '../../lib/validate'
import { useAccess } from '../../state/session'
import type { Vehicle } from '../../types'
import { FilterBar, FilterSelect, FilterText } from '../../ui/Filters'
import { Page, Pager, QueryView, Status } from '../../ui/Page'
import { Plate } from '../../ui/Plate'
import { VEHICLE_FILTER_KEYS, useVehiclePage } from './api'
import { CreateVehicleDialog, DeleteVehicleDialog, EditVehicleDialog } from './VehicleDialogs'

type Dialog = { kind: 'create' } | { kind: 'edit'; vehicle: Vehicle } | { kind: 'delete'; vehicle: Vehicle }

export function VehiclesPage() {
  const { manageFleet } = useAccess()
  const list = useListParams(VEHICLE_FILTER_KEYS)
  const query = useVehiclePage(list.filter, list.offset)
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const close = () => setDialog(null)

  return (
    <Page
      title="Машины"
      lead="Парк организации. Удалённые машины в список не попадают."
      actions={
        manageFleet && (
          <button type="button" className="btn btn-primary" onClick={() => setDialog({ kind: 'create' })}>
            Добавить машину
          </button>
        )
      }
    >
      <FilterBar active={list.active} onReset={list.reset}>
        <FilterSelect
          label="Статус"
          value={list.filter.status}
          onChange={(v) => list.set('status', v)}
          options={optionsOf(VEHICLE_STATUS_LABEL, ['IDLE', 'ON_TRIP', 'IN_SERVICE'])}
        />
        <FilterText
          label="Госномер, точно"
          value={list.filter.number_plate}
          onCommit={(v) => list.set('number_plate', v)}
          normalize={normalizePlate}
          placeholder="А123ВС77"
        />
        <FilterText label="Модель, точно" value={list.filter.model} onCommit={(v) => list.set('model', v)} />
      </FilterBar>

      <QueryView
        query={query}
        isEmpty={(page) => page.rows.length === 0 && list.offset === 0}
        empty={list.active ? 'По фильтру ничего не нашлось.' : 'Машин пока нет. Добавьте первую — или запустите генератор парка.'}
      >
        {(page) => (
          <>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col">Госномер</th>
                    <th scope="col">Модель</th>
                    <th scope="col">VIN</th>
                    <th scope="col">Статус</th>
                    {manageFleet && <th scope="col" aria-label="Действия" />}
                  </tr>
                </thead>
                <tbody>
                  {page.rows.map((v) => (
                    <tr key={v.id}>
                      <td>
                        <Link to={`/vehicles/${v.id}`} className="row-link" aria-label={`Машина ${v.number_plate}`}>
                          <Plate value={v.number_plate} size="sm" />
                        </Link>
                      </td>
                      <td>{v.model}</td>
                      <td className="mono">{v.vin}</td>
                      <td>
                        <Status cond={VEHICLE_STATUS_COND[v.status]}>{VEHICLE_STATUS_LABEL[v.status]}</Status>
                      </td>
                      {manageFleet && (
                        <td className="row-actions">
                          <button type="button" className="btn btn-sm" onClick={() => setDialog({ kind: 'edit', vehicle: v })}>
                            Изменить
                          </button>
                          <button
                            type="button"
                            className="btn btn-sm btn-quiet"
                            onClick={() => setDialog({ kind: 'delete', vehicle: v })}
                          >
                            Удалить
                          </button>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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

      {dialog?.kind === 'create' && <CreateVehicleDialog onClose={close} />}
      {dialog?.kind === 'edit' && <EditVehicleDialog vehicle={dialog.vehicle} onClose={close} />}
      {dialog?.kind === 'delete' && <DeleteVehicleDialog vehicle={dialog.vehicle} onClose={close} />}
    </Page>
  )
}
