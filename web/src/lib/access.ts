import type { UserRole } from '../types'

// Права повторяют middleware.RequireRole в internal/router/router.go. Бэк проверяет их сам,
// фронт по ним только прячет разделы и кнопки, которые закончились бы 403
export interface Access {
  // карта и поток событий: водителю обзор всего парка не положен
  viewFleet: boolean
  // заводить и менять машины, водителей, трекеры, рейсы, правила; принимать тревоги
  manageFleet: boolean
  // заводить и удалять пользователей
  admin: boolean
}

export const NO_ACCESS: Access = { viewFleet: false, manageFleet: false, admin: false }

// FULL_ACCESS — симулятор: входа нет, показываем всё
export const FULL_ACCESS: Access = { viewFleet: true, manageFleet: true, admin: true }

export function accessOf(role: UserRole | undefined): Access {
  switch (role) {
    case 'ADMIN':
      return FULL_ACCESS
    case 'DISPATCHER':
      return { viewFleet: true, manageFleet: true, admin: false }
    case 'ANALYTIC':
      return { viewFleet: true, manageFleet: false, admin: false }
    default:
      return NO_ACCESS
  }
}
