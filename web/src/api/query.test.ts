import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, recordFetch } from '../test/http'
import { PAGE_SIZE, apiList, pageParams, toPage, withQuery } from './query'

afterEach(() => vi.unstubAllGlobals())

describe('apiList', () => {
  it('пустая выборка с data: null — пустой массив, а не падение таблицы', async () => {
    // так отвечает GET /trips: nil-слайс в Go сериализуется в null
    recordFetch(() => json(null))
    await expect(apiList('/trips', { vehicle_id: 3 })).resolves.toEqual([])
  })
})

describe('withQuery', () => {
  it('пропускает пустые значения и кодирует остальные', () => {
    expect(withQuery('/trips', { status: 'RUNNING', driver_id: '', vehicle_id: null, offset: 0, from: undefined })).toBe(
      '/trips?status=RUNNING&offset=0',
    )
    expect(withQuery('/vehicles', { number_plate: 'А123ВС77' })).toBe('/vehicles?number_plate=%D0%90123%D0%92%D0%A177')
  })

  it('без параметров — путь как есть', () => {
    expect(withQuery('/alerts', {})).toBe('/alerts')
  })
})

describe('страницы списков', () => {
  it('просит на одну запись больше, чтобы узнать о следующей странице', () => {
    expect(pageParams(100)).toEqual({ limit: PAGE_SIZE + 1, offset: 100 })
  })

  it('лишняя запись не показывается, а говорит, что дальше есть ещё', () => {
    const full = Array.from({ length: PAGE_SIZE + 1 }, (_, i) => i)
    expect(toPage(full)).toEqual({ rows: full.slice(0, PAGE_SIZE), hasNext: true })
    expect(toPage([1, 2])).toEqual({ rows: [1, 2], hasNext: false })
  })
})
