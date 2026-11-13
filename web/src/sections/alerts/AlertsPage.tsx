import { Link } from 'react-router'
import { PAGE_SIZE } from '../../api/query'
import { errorText } from '../../api/errors'
import { formatDateTime, formatDecimal, formatTime } from '../../lib/format'
import {
  ALERT_STATUS_LABEL,
  ALERT_STATUS_TONE,
  SEVERITY_LABEL,
  SEVERITY_TONE,
  optionsOf,
} from '../../lib/labels'
import { useListParams } from '../../lib/useListParams'
import { ALERT_LABEL } from '../../state/selectors'
import { useAccess } from '../../state/session'
import type { Alert } from '../../types'
import { FilterBar, FilterDate, FilterSelect } from '../../ui/Filters'
import { Page, Pager, QueryView, Status } from '../../ui/Page'
import { Plate } from '../../ui/Plate'
import { useVehicleIndex } from '../vehicles/api'
import { ALERT_FILTER_KEYS, useAcknowledge, useAlertPage } from './api'
import { AlertTabs } from './AlertTabs'

// единицы значения, на котором сработало правило
const VALUE_UNIT: Record<Alert['type'], string> = {
  SPEED_EXCEEDED: 'км/ч',
  LOW_FUEL: '%',
  DEVICE_OFFLINE: 'мин',
}

export function AlertsPage() {
  const { manageFleet } = useAccess()
  const list = useListParams(ALERT_FILTER_KEYS)
  const query = useAlertPage(list.filter, list.offset)
  const vehicles = useVehicleIndex()
  const acknowledge = useAcknowledge()

  return (
    <Page title="Тревоги" lead="Тревоги поднимает движок правил на бэке. Журнал обновляется сам раз в полминуты.">
      <AlertTabs />

      <FilterBar active={list.active} onReset={list.reset}>
        <FilterSelect
          label="Состояние"
          value={list.filter.status}
          onChange={(v) => list.set('status', v)}
          options={optionsOf(ALERT_STATUS_LABEL)}
        />
        <FilterSelect
          label="Критичность"
          value={list.filter.severity}
          onChange={(v) => list.set('severity', v)}
          options={optionsOf(SEVERITY_LABEL)}
        />
        <FilterSelect
          label="Машина"
          value={list.filter.vehicle_id}
          onChange={(v) => list.set('vehicle_id', v)}
          options={Object.values(vehicles.data ?? {}).map((v) => ({ value: String(v.id), label: v.number_plate }))}
        />
        <FilterDate label="С даты" value={list.filter.created_from} onChange={(v) => list.set('created_from', v)} />
      </FilterBar>

      {acknowledge.isError && (
        <p className="form-error" role="alert">
          Не удалось принять: {errorText(acknowledge.error)}
        </p>
      )}

      <QueryView
        query={query}
        isEmpty={(page) => page.rows.length === 0 && list.offset === 0}
        empty={list.active ? 'По фильтру тревог нет.' : 'Тревог не было. Так держать.'}
      >
        {(page) => (
          <>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col">Когда</th>
                    <th scope="col">Машина</th>
                    <th scope="col">Что случилось</th>
                    <th scope="col" className="num">Значение</th>
                    <th scope="col">Критичность</th>
                    <th scope="col">Состояние</th>
                    {manageFleet && <th scope="col" aria-label="Действия" />}
                  </tr>
                </thead>
                <tbody>
                  {page.rows.map((a) => {
                    const vehicle = vehicles.data?.[a.vehicle_id]
                    return (
                      <tr key={a.id} data-status={a.status}>
                        <td className="nowrap">{formatDateTime(a.created_at)}</td>
                        <td>
                          {vehicle ? (
                            <Link to={`/vehicles/${vehicle.id}`} aria-label={`Машина ${vehicle.number_plate}`}>
                              <Plate value={vehicle.number_plate} size="sm" />
                            </Link>
                          ) : (
                            <span className="dim">№{a.vehicle_id}</span>
                          )}
                        </td>
                        <td>
                          <b>{ALERT_LABEL[a.type]}</b>
                          <span className="cell-note">{a.message}</span>
                        </td>
                        <td className="num">{a.value === null ? '—' : `${formatDecimal(a.value)} ${VALUE_UNIT[a.type]}`}</td>
                        <td>
                          <Status tone={SEVERITY_TONE[a.severity]}>{SEVERITY_LABEL[a.severity]}</Status>
                        </td>
                        <td>
                          <Status tone={ALERT_STATUS_TONE[a.status]}>{ALERT_STATUS_LABEL[a.status]}</Status>
                          {a.acknowledged_at && <span className="cell-note">принята в {formatTime(a.acknowledged_at)}</span>}
                          {a.resolved_at && <span className="cell-note">закрыта в {formatTime(a.resolved_at)}</span>}
                        </td>
                        {manageFleet && (
                          <td className="row-actions">
                            {a.status === 'FIRED' && (
                              <button
                                type="button"
                                className="btn btn-sm btn-alarm"
                                disabled={acknowledge.isPending && acknowledge.variables === a.id}
                                onClick={() => acknowledge.mutate(a.id)}
                              >
                                Принять
                              </button>
                            )}
                          </td>
                        )}
                      </tr>
                    )
                  })}
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
    </Page>
  )
}
