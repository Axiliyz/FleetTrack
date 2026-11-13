import { useState, type FormEvent } from 'react'
import { Link, Navigate, useSearchParams } from 'react-router'
import { formatDateTime, formatTimeSec } from '../../lib/format'
import { DEVICE_STATUS_LABEL } from '../../lib/labels'
import { useAccess } from '../../state/session'
import type { Device } from '../../types'
import { Page, QueryView, Status } from '../../ui/Page'
import { Plate } from '../../ui/Plate'
import { useRecentDevices } from '../telemetry/api'
import { useVehicleIndex } from '../vehicles/api'
import { useDevice } from './api'
import { AssignDeviceDialog, CreateDeviceDialog, DeleteDeviceDialog } from './DeviceDialogs'

type Dialog = { kind: 'create' } | { kind: 'assign'; device: Device } | { kind: 'delete'; device: Device }

// DevicesPage — трекеры. Списка трекеров в API нет: трекер открывают по номеру,
// а кто сейчас шлёт точки — видно по свежей телеметрии
export function DevicesPage() {
  const { manageFleet } = useAccess()
  const [params, setParams] = useSearchParams()
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const close = () => setDialog(null)

  const raw = Number(params.get('id'))
  const deviceId = Number.isInteger(raw) && raw > 0 ? raw : null
  const open = (id: number | null) => setParams(id ? { id: String(id) } : {}, { replace: true })

  if (!manageFleet) return <Navigate to="/" replace />

  return (
    <Page
      title="Трекеры"
      lead="GPS-терминалы на машинах. Точки принимаются только от трекера, привязанного к машине."
      actions={
        <button type="button" className="btn btn-primary" onClick={() => setDialog({ kind: 'create' })}>
          Зарегистрировать трекер
        </button>
      }
    >
      <DeviceLookup key={deviceId} initial={deviceId} onOpen={open} />

      {deviceId !== null && (
        <DeviceCard
          id={deviceId}
          onAssign={(device) => setDialog({ kind: 'assign', device })}
          onDelete={(device) => setDialog({ kind: 'delete', device })}
        />
      )}

      <OnAir onOpen={open} />

      {dialog?.kind === 'create' && (
        <CreateDeviceDialog
          onClose={close}
          onCreated={(device) => {
            close()
            open(device.id)
          }}
        />
      )}
      {dialog?.kind === 'assign' && <AssignDeviceDialog device={dialog.device} onClose={close} />}
      {dialog?.kind === 'delete' && (
        <DeleteDeviceDialog
          device={dialog.device}
          onClose={close}
          onDeleted={() => {
            close()
            open(null)
          }}
        />
      )}
    </Page>
  )
}

function DeviceLookup({ initial, onOpen }: { initial: number | null; onOpen: (id: number | null) => void }) {
  const [value, setValue] = useState(initial ? String(initial) : '')
  const submit = (e: FormEvent) => {
    e.preventDefault()
    const n = Number(value.trim())
    onOpen(Number.isInteger(n) && n > 0 ? n : null)
  }
  return (
    <form className="filters" role="search" onSubmit={submit}>
      <label className="filter">
        <span>Трекер №</span>
        <input inputMode="numeric" value={value} onChange={(e) => setValue(e.target.value)} placeholder="например, 12" />
      </label>
      <button type="submit" className="btn btn-sm">
        Найти
      </button>
    </form>
  )
}

function DeviceCard({
  id,
  onAssign,
  onDelete,
}: {
  id: number
  onAssign: (d: Device) => void
  onDelete: (d: Device) => void
}) {
  const query = useDevice(id)
  return (
    <section className="panel" aria-label={`Трекер №${id}`}>
      <QueryView query={query}>
        {(device) => (
          <>
            <header className="panel-head">
              <h2>
                Трекер №{device.id} <span className="mono dim">{device.serial_number}</span>
              </h2>
              <div className="panel-actions">
                <button type="button" className="btn btn-sm" onClick={() => onAssign(device)}>
                  Привязать к машине
                </button>
                <button type="button" className="btn btn-sm btn-quiet" onClick={() => onDelete(device)}>
                  Удалить
                </button>
              </div>
            </header>
            <dl className="stats stats-row">
              <div>
                <dt>Состояние</dt>
                <dd>
                  <Status tone={device.status === 'ACTIVE' ? 'ok' : 'warn'}>{DEVICE_STATUS_LABEL[device.status]}</Status>
                </dd>
              </div>
              <div>
                <dt>Зарегистрирован</dt>
                <dd>{formatDateTime(device.created_at)}</dd>
              </div>
            </dl>
          </>
        )}
      </QueryView>
    </section>
  )
}

function OnAir({ onOpen }: { onOpen: (id: number) => void }) {
  const query = useRecentDevices()
  const vehicles = useVehicleIndex().data ?? {}
  return (
    <section className="panel" aria-labelledby="onair-title">
      <header className="panel-head">
        <h2 id="onair-title">Сейчас на связи</h2>
      </header>
      <QueryView query={query} empty="Последние точки ни от одного трекера не приходили.">
        {(points) => (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th scope="col">Трекер</th>
                  <th scope="col">Машина</th>
                  <th scope="col">Последняя точка</th>
                </tr>
              </thead>
              <tbody>
                {points.map((p) => {
                  const vehicle = vehicles[p.vehicle_id]
                  return (
                    <tr key={p.device_id}>
                      <td>
                        <button type="button" className="link-button" onClick={() => onOpen(p.device_id)}>
                          №{p.device_id}
                        </button>
                      </td>
                      <td>
                        {vehicle ? (
                          <Link to={`/vehicles/${vehicle.id}`} aria-label={`Машина ${vehicle.number_plate}`}>
                            <Plate value={vehicle.number_plate} size="sm" />
                          </Link>
                        ) : (
                          `№${p.vehicle_id}`
                        )}
                      </td>
                      <td>{formatTimeSec(p.received_at)}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </QueryView>
    </section>
  )
}
