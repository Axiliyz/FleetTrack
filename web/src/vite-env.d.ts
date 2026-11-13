/// <reference types="vite/client" />

interface ImportMetaEnv {
  // mock — встроенный симулятор, api — живой бэкенд
  readonly VITE_FLEET_SOURCE?: 'mock' | 'api'
  // учётка демо-стенда для кнопки «Войти в демо-парк»; без них кнопки нет
  readonly VITE_DEMO_EMAIL?: string
  readonly VITE_DEMO_PASSWORD?: string
}
