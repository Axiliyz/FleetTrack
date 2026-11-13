import { useState } from 'react'
import { Link } from 'react-router'
import { PAGE_SIZE } from '../../api/query'
import { formatDateTime } from '../../lib/format'
import { useListParams } from '../../lib/useListParams'
import { useAccess } from '../../state/session'
import type { Driver } from '../../types'
import { FilterBar, FilterText } from '../../ui/Filters'
import { Page, Pager, QueryView } from '../../ui/Page'
import { DRIVER_FILTER_KEYS, useDriverPage } from './api'
import { CreateDriverDialog, DeleteDriverDialog, RenameDriverDialog } from './DriverDialogs'

type Dialog = { kind: 'create' } | { kind: 'rename'; driver: Driver } | { kind: 'delete'; driver: Driver }

export function DriversPage() {
  const { manageFleet } = useAccess()
  const list = useListParams(DRIVER_FILTER_KEYS)
  const query = useDriverPage(list.filter, list.offset)
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const close = () => setDialog(null)

  return (
    <Page
      title="Водители"
      lead="Водителя назначают на рейс вместе с машиной. Рейтинг смены — на карте, во вкладке у списка машин."
      actions={
        manageFleet && (
          <button type="button" className="btn btn-primary" onClick={() => setDialog({ kind: 'create' })}>
            Добавить водителя
          </button>
        )
      }
    >
      <FilterBar active={list.active} onReset={list.reset}>
        <FilterText label="Имя, точно" value={list.filter.name} onCommit={(v) => list.set('name', v)} />
      </FilterBar>

      <QueryView
        query={query}
        isEmpty={(page) => page.rows.length === 0 && list.offset === 0}
        empty={list.active ? 'Никого не нашлось.' : 'Водителей пока нет.'}
      >
        {(page) => (
          <>
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th scope="col">Имя</th>
                    <th scope="col">Добавлен</th>
                    <th scope="col">Изменён</th>
                    {manageFleet && <th scope="col" aria-label="Действия" />}
                  </tr>
                </thead>
                <tbody>
                  {page.rows.map((d) => (
                    <tr key={d.id}>
                      <td>
                        <Link to={`/drivers/${d.id}`} className="row-link">
                          {d.name}
                        </Link>
                      </td>
                      <td className="nowrap">{formatDateTime(d.created_at)}</td>
                      <td className="nowrap">{d.updated_at ? formatDateTime(d.updated_at) : '—'}</td>
                      {manageFleet && (
                        <td className="row-actions">
                          <button type="button" className="btn btn-sm" onClick={() => setDialog({ kind: 'rename', driver: d })}>
                            Переименовать
                          </button>
                          <button
                            type="button"
                            className="btn btn-sm btn-quiet"
                            onClick={() => setDialog({ kind: 'delete', driver: d })}
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

      {dialog?.kind === 'create' && <CreateDriverDialog onClose={close} />}
      {dialog?.kind === 'rename' && <RenameDriverDialog driver={dialog.driver} onClose={close} />}
      {dialog?.kind === 'delete' && <DeleteDriverDialog driver={dialog.driver} onClose={close} />}
    </Page>
  )
}
