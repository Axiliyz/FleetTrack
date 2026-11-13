import { useState } from 'react'
import type { Device, Vehicle } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { Plate } from '../../ui/Plate'
import { useVehicleIndex } from '../vehicles/api'
import { useAssignDevice, useCreateDevice, useDeleteDevice } from './api'

// серийный номер на бэке — VARCHAR(64)
const SERIAL_MAX = 64

export function CreateDeviceDialog({ onClose, onCreated }: { onClose: () => void; onCreated: (d: Device) => void }) {
  const create = useCreateDevice()
  const [serial, setSerial] = useState('')

  return (
    <FormDialog
      title="Новый трекер"
      onClose={onClose}
      onSubmit={() => create.mutate(serial.trim(), { onSuccess: onCreated })}
      submitLabel="Зарегистрировать"
      pending={create.isPending}
      error={create.error}
    >
      <label className="field">
        <span>Серийный номер</span>
        <input required maxLength={SERIAL_MAX} value={serial} onChange={(e) => setSerial(e.target.value)} />
      </label>
      <p className="field-note">Трекер появится в статусе «Работает». Чтобы он слал точки, привяжите его к машине.</p>
    </FormDialog>
  )
}

// AssignDeviceDialog — привязка с любой стороны: со страницы машины выбирают трекер, со страницы трекера — машину
export function AssignDeviceDialog({
  vehicle,
  device,
  onClose,
}: {
  vehicle?: Vehicle
  device?: Device
  onClose: () => void
}) {
  const assign = useAssignDevice()
  const vehicles = useVehicleIndex()
  const [deviceId, setDeviceId] = useState(device ? String(device.id) : '')
  const [vehicleId, setVehicleId] = useState(vehicle ? String(vehicle.id) : '')
  const [problem, setProblem] = useState<string | null>(null)

  // бэк привязывает трекер только к свободной машине
  const free = Object.values(vehicles.data ?? {})
    .filter((v) => v.status === 'IDLE')
    .sort((a, b) => a.number_plate.localeCompare(b.number_plate, 'ru'))

  const submit = () => {
    const d = Number(deviceId)
    const v = Number(vehicleId)
    const found = !Number.isInteger(d) || d <= 0 ? 'Укажите номер трекера' : !v ? 'Выберите машину' : null
    setProblem(found)
    if (!found) assign.mutate({ deviceId: d, vehicleId: v }, { onSuccess: onClose })
  }

  return (
    <FormDialog
      title="Привязать трекер"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Привязать"
      pending={assign.isPending}
      error={problem ?? assign.error}
    >
      {vehicle ? (
        <p className="dialog-text">
          Машина <Plate value={vehicle.number_plate} size="sm" /> {vehicle.model}
        </p>
      ) : (
        <label className="field">
          <span>Машина</span>
          <select required value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
            <option value="">{vehicles.isPending ? 'Загружаю…' : 'Выберите свободную машину'}</option>
            {free.map((v) => (
              <option key={v.id} value={v.id}>
                {v.number_plate} — {v.model}
              </option>
            ))}
          </select>
        </label>
      )}
      {device ? (
        <p className="dialog-text">
          Трекер №{device.id}, серийный {device.serial_number}
        </p>
      ) : (
        <label className="field">
          <span>Трекер №</span>
          <input inputMode="numeric" required value={deviceId} onChange={(e) => setDeviceId(e.target.value)} />
        </label>
      )}
      <p className="field-note">Если трекер стоял на другой машине, бэк закроет ту привязку и переставит его сюда.</p>
    </FormDialog>
  )
}

export function DeleteDeviceDialog({
  device,
  onClose,
  onDeleted,
}: {
  device: Device
  onClose: () => void
  onDeleted: () => void
}) {
  const remove = useDeleteDevice()
  return (
    <ConfirmDialog
      title="Удалить трекер"
      onClose={onClose}
      onConfirm={() => remove.mutate(device.id, { onSuccess: onDeleted })}
      confirmLabel="Удалить"
      pending={remove.isPending}
      error={remove.error}
    >
      Трекер №{device.id} ({device.serial_number}) отвяжется от машины и перестанет принимать точки.
    </ConfirmDialog>
  )
}
