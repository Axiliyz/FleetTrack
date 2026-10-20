/// <reference types="vite/client" />

interface ImportMetaEnv {
  // mock — встроенный симулятор, api — живой бэкенд
  readonly VITE_FLEET_SOURCE?: 'mock' | 'api'
}
