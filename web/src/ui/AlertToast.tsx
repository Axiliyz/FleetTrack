import { useEffect } from 'react'
import { acknowledgeAlert } from '../state/connect'
import { useFleet } from '../state/fleet'
import { ALERT_LABEL } from '../state/selectors'
import { Plate } from './Plate'

const TOAST_MS = 7000

// AlertToast — всплывающая тревога высокой и критической важности
export function AlertToast() {
  const alertId = useFleet((s) => s.toastAlertId)
  const alert = useFleet((s) => (s.toastAlertId === null ? undefined : s.alerts[s.toastAlertId]))
  const vehicle = useFleet((s) => (alert ? s.vehicles[alert.vehicle_id] : undefined))
  const select = useFleet((s) => s.select)
  const dismiss = useFleet((s) => s.dismissToast)

  useEffect(() => {
    if (alertId === null) return
    const timer = setTimeout(dismiss, TOAST_MS)
    return () => clearTimeout(timer)
  }, [alertId, dismiss])

  if (!alert || !vehicle) return null

  return (
    <div className="toast" role="alert" data-severity={alert.severity}>
      <span className="toast-lamp" aria-hidden />
      <div className="toast-body">
        <b>{ALERT_LABEL[alert.type]}</b>
        <span>{alert.message}</span>
      </div>
      <Plate value={vehicle.number_plate} size="sm" />
      <div className="toast-actions">
        <button
          type="button"
          className="btn"
          onClick={() => {
            select(vehicle.id, { fly: true })
            dismiss()
          }}
        >
          Показать
        </button>
        <button type="button" className="btn btn-alarm" onClick={() => void acknowledgeAlert(alert.id)}>
          Принять
        </button>
      </div>
    </div>
  )
}
