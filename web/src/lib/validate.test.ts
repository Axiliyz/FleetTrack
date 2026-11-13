import { describe, expect, it } from 'vitest'
import { normalizePlate, normalizeVin, passwordProblem, plateProblem, vinProblem } from './validate'

describe('VIN', () => {
  it('приводится к виду бэка: без пробелов, заглавными', () => {
    expect(normalizeVin(' xta 21099012345678 ')).toBe('XTA21099012345678')
  })

  it('ровно 17 латинских букв и цифр', () => {
    expect(vinProblem('XTA21099012345678')).toBeNull()
    expect(vinProblem('XTA2109901234567')).not.toBeNull()
    // кириллица в VIN не проходит, как и в validator.IsVINValid
    expect(vinProblem('ХТА21099012345678')).not.toBeNull()
  })
})

describe('госномер', () => {
  it('считается в символах, а не в байтах', () => {
    expect(plateProblem(normalizePlate('а 123 вс 77'))).toBeNull()
    expect(plateProblem('А123ВС177')).toBeNull()
    expect(plateProblem('А123ВС')).not.toBeNull()
  })
})

describe('пароль', () => {
  it('от 8 символов, буквы и цифры', () => {
    expect(passwordProblem('secret12', 'a@b.c', 'Иван')).toBeNull()
    expect(passwordProblem('short1', 'a@b.c', 'Иван')).toMatch(/от 8/)
    expect(passwordProblem('onlyletters', 'a@b.c', 'Иван')).toMatch(/и буквы, и цифры/)
  })

  it('не совпадает с почтой или именем', () => {
    expect(passwordProblem('Disp4tcher', 'a@b.c', 'disp4tcher')).toMatch(/не должен совпадать/)
  })
})
