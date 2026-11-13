// LIVE — живой бэкенд (VITE_FLEET_SOURCE=api), иначе встроенный симулятор; см. web/.env.example
export const LIVE = import.meta.env.VITE_FLEET_SOURCE === 'api'

// DEMO — учётка демо-стенда: кнопка «Войти в демо-парк» на экране входа.
// Это публичный аккаунт витрины, а не секрет: в сборку попадает только то, что задано при сборке
const demoEmail = import.meta.env.VITE_DEMO_EMAIL
const demoPassword = import.meta.env.VITE_DEMO_PASSWORD
export const DEMO = demoEmail && demoPassword ? { email: demoEmail, password: demoPassword } : null
