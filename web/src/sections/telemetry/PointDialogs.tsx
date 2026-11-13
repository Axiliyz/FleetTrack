import { useState } from 'react'
import { formatInt } from '../../lib/format'
import type { TelemetryEvent, Vehicle } from '../../types'
import { ConfirmDialog, FormDialog } from '../../ui/Dialog'
import { Plate } from '../../ui/Plate'
import { useClearVehicleTrack, useDeletePoint, useSendPoint } from './api'

// parseNumber: в русской раскладке десятичный разделитель — запятая
const parseNumber = (raw: string) => Number(raw.trim().replace(',', '.'))

// SendPointDialog — точка «как от терминала». Удобно показать, как бэк считает скорость
// и поднимает тревоги: точка уходит в тот же POST /telemetry, что и у генератора парка
export function SendPointDialog({
  vehicle,
  last,
  onClose,
}: {
  vehicle: Vehicle
  last: TelemetryEvent | undefined
  onClose: () => void
}) {
  const send = useSendPoint()
  const [deviceId, setDeviceId] = useState(last ? String(last.device_id) : '')
  const [lat, setLat] = useState(last ? String(last.lat) : '')
  const [lon, setLon] = useState(last ? String(last.lon) : '')
  const [fuel, setFuel] = useState(last?.fuel != null ? String(Math.round(last.fuel)) : '')
  const [problem, setProblem] = useState<string | null>(null)

  const submit = () => {
    const point = {
      device_id: parseNumber(deviceId),
      vehicle_id: vehicle.id,
      lat: parseNumber(lat),
      lon: parseNumber(lon),
      fuel: fuel.trim() === '' ? null : parseNumber(fuel),
    }
    let found: string | null = null
    if (!Number.isInteger(point.device_id) || point.device_id <= 0) found = 'Укажите номер трекера'
    else if (!(Math.abs(point.lat) <= 90) || !(Math.abs(point.lon) <= 180)) found = 'Широта от −90 до 90, долгота от −180 до 180'
    else if (point.fuel !== null && !(point.fuel >= 0 && point.fuel <= 100)) found = 'Топливо — от 0 до 100%'
    setProblem(found)
    if (!found) send.mutate(point, { onSuccess: onClose })
  }

  return (
    <FormDialog
      title="Отправить точку"
      onClose={onClose}
      onSubmit={submit}
      submitLabel="Отправить"
      pending={send.isPending}
      error={problem ?? send.error}
    >
      <p className="dialog-text">
        Точка уйдёт от имени трекера, привязанного к машине <Plate value={vehicle.number_plate} size="sm" />. Если у
        машины есть рейс в пути, бэк добавит точку в рейс и пересчитает пробег и скорость.
      </p>
      <label className="field">
        <span>Трекер №</span>
        <input inputMode="numeric" required value={deviceId} onChange={(e) => setDeviceId(e.target.value)} />
      </label>
      <div className="field-row">
        <label className="field">
          <span>Широта</span>
          <input inputMode="decimal" required value={lat} onChange={(e) => setLat(e.target.value)} placeholder="55.7558" />
        </label>
        <label className="field">
          <span>Долгота</span>
          <input inputMode="decimal" required value={lon} onChange={(e) => setLon(e.target.value)} placeholder="37.6173" />
        </label>
      </div>
      <label className="field">
        <span>Топливо, %</span>
        <input inputMode="decimal" value={fuel} onChange={(e) => setFuel(e.target.value)} placeholder="необязательно" />
      </label>
    </FormDialog>
  )
}

export function DeletePointDialog({ point, onClose }: { point: TelemetryEvent; onClose: () => void }) {
  const remove = useDeletePoint()
  return (
    <ConfirmDialog
      title="Удалить точку"
      onClose={onClose}
      onConfirm={() => remove.mutate(point.telemetry_id, { onSuccess: onClose })}
      confirmLabel="Удалить"
      pending={remove.isPending}
      error={remove.error}
    >
      Точка №{point.telemetry_id} пропадёт из трека. Пробег рейса бэк не пересчитывает.
    </ConfirmDialog>
  )
}

export function ClearTrackDialog({ vehicle, onClose }: { vehicle: Vehicle; onClose: () => void }) {
  const clear = useClearVehicleTrack()
  return (
    <ConfirmDialog
      title="Очистить историю"
      onClose={onClose}
      onConfirm={() => clear.mutate(vehicle.id, { onSuccess: onClose })}
      confirmLabel="Удалить всю телеметрию"
      pending={clear.isPending}
      error={clear.error}
    >
      Все точки машины <Plate value={vehicle.number_plate} size="sm" /> будут удалены
      {clear.data ? ` (удалено ${formatInt(clear.data.length)})` : ''}. Отменить это нельзя.
    </ConfirmDialog>
  )
}
