// LIVE — живой бэкенд (VITE_FLEET_SOURCE=api), иначе встроенный симулятор; см. web/.env.example
export const LIVE = import.meta.env.VITE_FLEET_SOURCE === 'api'
