import { useMemo } from 'react'
import { useFleet } from '../state/fleet'
import { CONDITION_LABEL, conditionOf, openAlertsOf } from '../state/selectors'
import { Plate } from '../ui/Plate'

const FUEL_SEGMENTS = 5

// TruckMarker — фура на карте: лампа статуса, стрелка курса, запас топлива и номер.
// Положение и курс задаёт цикл анимации через CSS-переменную --heading на родителе.
export function TruckMarker({ vehicleId }: { vehicleId: number }) {
  const vehicle = useFleet((s) => s.vehicles[vehicleId])
  const alerts = useFleet((s) => s.alerts)
  const point = useFleet((s) => s.telemetry[vehicleId])
  const selected = useFleet((s) => s.selectedId === vehicleId)
  const fuelThreshold = useFleet((s) => s.rules.find((r) => r.type === 'LOW_FUEL')?.threshold ?? 15)
  const select = useFleet((s) => s.select)

  const open = useMemo(() => openAlertsOf(alerts, vehicleId), [alerts, vehicleId])
  if (!vehicle) return null

  const condition = conditionOf(vehicle, open)
  const fired = open.some((a) => a.status === 'FIRED')
  const fuel = point?.fuel ?? null
  const lit = fuel === null ? 0 : Math.ceil((fuel / 100) * FUEL_SEGMENTS)
  const speed = point ? Math.round(point.speed_kmh) : 0

  return (
    <button
      type="button"
      className="unit"
      data-cond={condition}
      data-fired={fired || undefined}
      data-selected={selected || undefined}
      data-moving={speed > 0 || undefined}
      aria-label={`${vehicle.number_plate}, ${CONDITION_LABEL[condition].toLowerCase()}, ${speed} км/ч`}
      aria-pressed={selected}
      onClick={() => select(selected ? null : vehicleId)}
    >
      <span className="unit-heading" aria-hidden />
      <span className="unit-lamp" aria-hidden />
      {fuel !== null && (
        <span className="unit-fuel" data-low={fuel < fuelThreshold || undefined} aria-hidden>
          {Array.from({ length: FUEL_SEGMENTS }, (_, i) => (
            <i key={i} data-on={i < lit || undefined} />
          ))}
        </span>
      )}
      <span className="unit-plate" aria-hidden>
        <Plate value={vehicle.number_plate} size="sm" />
      </span>
    </button>
  )
}
