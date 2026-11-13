import { useState } from 'react'
import { useNavigate } from 'react-router'
import { VEHICLE_STATUS_LABEL, optionsOf } from '../../lib/labels'
import { normalizePlate, normalizeVin, plateProblem, vinProblem } from '../../lib/validate'
import type { Vehicle, VehicleStatus } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { Plate } from '../../ui/Plate'
import { useCreateVehicle, useDeleteVehicle, useUpdateVehicle, type VehiclePatch } from './api'

// вручную ставим «свободна» и «на обслуживании»; «в рейсе» машина становится вместе с рейсом
const MANUAL_STATUSES: VehicleStatus[] = ['IDLE', 'IN_SERVICE']

export function CreateVehicleDialog({ onClose }: { onClose: () => void }) {
  const create = useCreateVehicle()
  const navigate = useNavigate()
  const [vin, setVin] = useState('')
  const [plate, setPlate] = useState('')
  const [model, setModel] = useState('')
  const [problem, setProblem] = useState<string | null>(null)

  const submit = () => {
    const clean = { vin: normalizeVin(vin), number_plate: normalizePlate(plate), model: model.trim() }
    const found = vinProblem(clean.vin) ?? plateProblem(clean.number_plate) ?? (clean.model ? null : 'Укажите модель')
    setProblem(found)
    if (found) return
    create.mutate(clean, { onSuccess: (vehicle) => navigate(`/vehicles/${vehicle.id}`) })
  }

  return (
    <FormDialog
      title="Новая машина"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Добавить"
      pending={create.isPending}
      error={problem ?? create.error}
    >
      <label className="field">
        <span>Госномер</span>
        <input required value={plate} onChange={(e) => setPlate(e.target.value)} placeholder="А123ВС77" />
      </label>
      <label className="field">
        <span>VIN</span>
        <input required maxLength={20} value={vin} onChange={(e) => setVin(e.target.value)} placeholder="XTA21099012345678" />
      </label>
      <label className="field">
        <span>Модель</span>
        <input required maxLength={30} value={model} onChange={(e) => setModel(e.target.value)} placeholder="КАМАЗ 54901" />
      </label>
    </FormDialog>
  )
}

export function EditVehicleDialog({ vehicle, onClose }: { vehicle: Vehicle; onClose: () => void }) {
  const update = useUpdateVehicle()
  const [plate, setPlate] = useState(vehicle.number_plate)
  const [status, setStatus] = useState(vehicle.status)
  const [problem, setProblem] = useState<string | null>(null)
  const onTrip = vehicle.status === 'ON_TRIP'

  const submit = () => {
    const nextPlate = normalizePlate(plate)
    const found = plateProblem(nextPlate)
    setProblem(found)
    if (found) return
    // отправляем только изменённые поля: PATCH не должен затирать то, что поменял кто-то другой
    const patch: VehiclePatch = {
      ...(nextPlate !== vehicle.number_plate && { number_plate: nextPlate }),
      ...(!onTrip && status !== vehicle.status && { status }),
    }
    if (Object.keys(patch).length === 0) {
      onClose()
      return
    }
    update.mutate({ id: vehicle.id, patch }, { onSuccess: onClose })
  }

  return (
    <FormDialog
      title="Изменить машину"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Сохранить"
      pending={update.isPending}
      error={problem ?? update.error}
    >
      <label className="field">
        <span>Госномер</span>
        <input required value={plate} onChange={(e) => setPlate(e.target.value)} />
      </label>
      <label className="field">
        <span>Статус</span>
        <select disabled={onTrip} value={status} onChange={(e) => setStatus(e.target.value as VehicleStatus)}>
          {optionsOf(VEHICLE_STATUS_LABEL, onTrip ? ['ON_TRIP'] : MANUAL_STATUSES).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </label>
      {onTrip && (
        <p className="field-note">
          Машина в рейсе: статусом управляет рейс, он сменится сам, когда рейс завершат или отменят.
        </p>
      )}
    </FormDialog>
  )
}

export function DeleteVehicleDialog({
  vehicle,
  onClose,
  onDeleted,
}: {
  vehicle: Vehicle
  onClose: () => void
  onDeleted?: () => void
}) {
  const remove = useDeleteVehicle()
  return (
    <ConfirmDialog
      title="Удалить машину"
      onClose={onClose}
      onConfirm={() => remove.mutate(vehicle.id, { onSuccess: onDeleted ?? onClose })}
      confirmLabel="Удалить"
      pending={remove.isPending}
      error={remove.error}
    >
      <Plate value={vehicle.number_plate} size="sm" /> {vehicle.model} пропадёт с карты и из списков. История рейсов и
      телеметрии останется в базе.
    </ConfirmDialog>
  )
}
