import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import { errorText } from './errors'

describe('errorText', () => {
  it('переводит фразы из error_mapper.go', () => {
    expect(errorText(new ApiError(409, 'vehicle with this vin already exists'))).toBe('Машина с таким VIN уже есть')
    expect(errorText(new ApiError(409, "driver has trips and can't be deleted"))).toBe(
      'У водителя есть рейсы, удалить его нельзя',
    )
  })

  it('незнакомый 403 — нет прав, незнакомый 5xx — ошибка сервера, остальное как есть', () => {
    expect(errorText(new ApiError(403, 'whatever'))).toBe('Недостаточно прав')
    expect(errorText(new ApiError(500, 'unknown error'))).toBe('Ошибка на сервере. Попробуйте ещё раз')
    expect(errorText(new ApiError(400, 'something new'))).toBe('something new')
  })

  it('сетевой сбой — нет связи', () => {
    expect(errorText(new TypeError('Failed to fetch'))).toBe('Нет связи с сервером')
  })
})
