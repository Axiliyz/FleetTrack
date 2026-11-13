import { describe, expect, it } from 'vitest'
import { accessOf } from './access'

// права сверены с middleware.RequireRole в internal/router/router.go
describe('accessOf', () => {
  it('администратор может всё', () => {
    expect(accessOf('ADMIN')).toEqual({ viewFleet: true, manageFleet: true, admin: true })
  })

  it('диспетчер управляет парком, но не людьми', () => {
    expect(accessOf('DISPATCHER')).toEqual({ viewFleet: true, manageFleet: true, admin: false })
  })

  it('аналитик только смотрит', () => {
    expect(accessOf('ANALYTIC')).toEqual({ viewFleet: true, manageFleet: false, admin: false })
  })

  it('водителю карта парка не положена, как и неизвестной роли', () => {
    expect(accessOf('DRIVER').viewFleet).toBe(false)
    expect(accessOf(undefined)).toEqual({ viewFleet: false, manageFleet: false, admin: false })
  })
})
