import { useState } from 'react'
import { useNavigate } from 'react-router'
import { TRIP_STATUS_LABEL, optionsOf } from '../../lib/labels'
import type { Trip, TripStatus } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { useDriverIndex } from '../drivers/api'
import { useVehicleIndex } from '../vehicles/api'
import { VehicleNotSynced, useDeleteTrip, useSetTripStatus, useStartTrip } from './api'

// StartTripDialog — назначить рейс: водитель и свободная машина
export function StartTripDialog({ onClose }: { onClose: () => void }) {
  const start = useStartTrip()
  const navigate = useNavigate()
  const vehicles = useVehicleIndex()
  const drivers = useDriverIndex()
  const [vehicle, setVehicle] = useState('')
  const [driver, setDriver] = useState('')

  const free = Object.values(vehicles.data ?? {})
    .filter((v) => v.status === 'IDLE')
    .sort((a, b) => a.number_plate.localeCompare(b.number_plate, 'ru'))
  const people = Object.values(drivers.data ?? {}).sort((a, b) => a.name.localeCompare(b.name, 'ru'))
  const loading = vehicles.isPending || drivers.isPending

  // рейс создан, а статус машины — нет: повтор создал бы второй рейс, поэтому кнопка ведёт к созданному
  const created = start.error instanceof VehicleNotSynced ? start.error.trip : null

  const submit = () => {
    if (created) {
      navigate(`/trips/${created.id}`)
      return
    }
    if (!vehicle || !driver) return
    start.mutate(
      { driverId: Number(driver), vehicleId: Number(vehicle) },
      { onSuccess: (trip) => navigate(`/trips/${trip.id}`) },
    )
  }

  return (
    <FormDialog
      title="Новый рейс"
      onClose={onClose}
      onSubmit={submit}
      submitLabel={created ? 'Открыть рейс' : 'Отправить в рейс'}
      pending={start.isPending}
      error={start.error}
    >
      <label className="field">
        <span>Машина</span>
        <select required value={vehicle} onChange={(e) => setVehicle(e.target.value)}>
          <option value="">{loading ? 'Загружаю…' : 'Выберите свободную машину'}</option>
          {free.map((v) => (
            <option key={v.id} value={v.id}>
              {v.number_plate} — {v.model}
            </option>
          ))}
        </select>
      </label>
      {!loading && free.length === 0 && <p className="field-note">Свободных машин нет: все в рейсе или на обслуживании.</p>}
      <label className="field">
        <span>Водитель</span>
        <select required value={driver} onChange={(e) => setDriver(e.target.value)}>
          <option value="">{loading ? 'Загружаю…' : 'Выберите водителя'}</option>
          {people.map((d) => (
            <option key={d.id} value={d.id}>
              {d.name}
            </option>
          ))}
        </select>
      </label>
    </FormDialog>
  )
}

export function TripStatusDialog({ trip, onClose }: { trip: Trip; onClose: () => void }) {
  const setStatus = useSetTripStatus()
  const [status, setValue] = useState<TripStatus>(trip.status === 'RUNNING' ? 'SUCCEEDED' : trip.status)

  return (
    <FormDialog
      title={`Рейс №${trip.id}`}
      onClose={onClose}
      onSubmit={() =>
        // после частичной ошибки рейс уже сохранён: повтор только упёрся бы в «рейс уже завершён»
        status === trip.status || setStatus.error instanceof VehicleNotSynced
          ? onClose()
          : setStatus.mutate({ id: trip.id, status }, { onSuccess: onClose })
      }
      submitLabel={setStatus.error instanceof VehicleNotSynced ? 'Закрыть' : 'Сохранить'}
      pending={setStatus.isPending}
      error={setStatus.error}
    >
      <label className="field">
        <span>Статус</span>
        <select value={status} onChange={(e) => setValue(e.target.value as TripStatus)}>
          {optionsOf(TRIP_STATUS_LABEL).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </label>
      {trip.status === 'RUNNING' && (
        <p className="field-note">Завершённый или отменённый рейс освобождает машину — её можно снова отправить в рейс.</p>
      )}
    </FormDialog>
  )
}

export function DeleteTripDialog({
  trip,
  onClose,
  onDeleted,
}: {
  trip: Trip
  onClose: () => void
  onDeleted?: () => void
}) {
  const remove = useDeleteTrip()
  return (
    <ConfirmDialog
      title="Удалить рейс"
      onClose={onClose}
      onConfirm={() =>
        remove.error instanceof VehicleNotSynced
          ? (onDeleted ?? onClose)()
          : remove.mutate(trip.id, { onSuccess: onDeleted ?? onClose })
      }
      confirmLabel={remove.error instanceof VehicleNotSynced ? 'Закрыть' : 'Удалить'}
      pending={remove.isPending}
      error={remove.error}
    >
      Рейс №{trip.id} пропадёт из истории и из рейтинга водителей. Обычно рейс не удаляют, а отменяют.
    </ConfirmDialog>
  )
}
