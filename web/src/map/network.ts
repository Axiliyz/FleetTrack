// Сеть федеральных трасс вокруг Москвы: подложка «мнемощита» и маршруты для симулятора.
// Трассы упрощены до прямых между городами — для масштаба области этого хватает.
import type { LngLat } from '../lib/geo'

export interface Place {
  name: string
  pos: LngLat
  major?: boolean
}

export interface Corridor {
  code: string
  name: string
  // города по порядку от МКАД наружу
  places: Place[]
}

export const MOSCOW: Place = { name: 'Москва', pos: [37.6173, 55.7558], major: true }

// МКАД как эллипс: в реальности кольцо ~34 км с севера на юг и ~30 км с запада на восток
const MKAD_CENTER: LngLat = [37.605, 55.743]
const MKAD_RX = 0.237
const MKAD_RY = 0.17

export const CORRIDORS: Corridor[] = [
  {
    code: 'М-10',
    name: 'Россия',
    places: [
      { name: 'Зеленоград', pos: [37.214, 55.991] },
      { name: 'Солнечногорск', pos: [36.977, 56.185] },
      { name: 'Клин', pos: [36.729, 56.333] },
      { name: 'Тверь', pos: [35.912, 56.859], major: true },
      { name: 'Торжок', pos: [34.961, 57.041] },
      { name: 'Вышний Волочёк', pos: [34.564, 57.588] },
      { name: 'Валдай', pos: [33.247, 57.978] },
      { name: 'Великий Новгород', pos: [31.27, 58.522], major: true },
      { name: 'Санкт-Петербург', pos: [30.316, 59.939], major: true },
    ],
  },
  {
    code: 'М-8',
    name: 'Холмогоры',
    places: [
      { name: 'Мытищи', pos: [37.73, 55.91] },
      { name: 'Сергиев Посад', pos: [38.136, 56.315] },
      { name: 'Переславль-Залесский', pos: [38.854, 56.736] },
      { name: 'Ростов', pos: [39.414, 57.185] },
      { name: 'Ярославль', pos: [39.894, 57.626], major: true },
    ],
  },
  {
    code: 'М-7',
    name: 'Волга',
    places: [
      { name: 'Балашиха', pos: [37.938, 55.796] },
      { name: 'Ногинск', pos: [38.441, 55.853] },
      { name: 'Покров', pos: [39.183, 55.917] },
      { name: 'Владимир', pos: [40.407, 56.129], major: true },
      { name: 'Вязники', pos: [42.159, 56.243] },
      { name: 'Гороховец', pos: [42.693, 56.202] },
      { name: 'Нижний Новгород', pos: [44.006, 56.327], major: true },
      { name: 'Чебоксары', pos: [47.25, 56.146] },
      { name: 'Казань', pos: [49.106, 55.796], major: true },
    ],
  },
  {
    code: 'М-5',
    name: 'Урал',
    places: [
      { name: 'Люберцы', pos: [37.893, 55.676] },
      { name: 'Бронницы', pos: [38.264, 55.425] },
      { name: 'Коломна', pos: [38.778, 55.079] },
      { name: 'Луховицы', pos: [39.025, 54.965] },
      { name: 'Рязань', pos: [39.742, 54.63], major: true },
    ],
  },
  {
    code: 'М-4',
    name: 'Дон',
    places: [
      { name: 'Домодедово', pos: [37.766, 55.437] },
      { name: 'Кашира', pos: [38.151, 54.834] },
      { name: 'Венёв', pos: [38.264, 54.354] },
      { name: 'Ефремов', pos: [38.12, 53.146] },
      { name: 'Елец', pos: [38.503, 52.624] },
      { name: 'Задонск', pos: [38.919, 52.389] },
      { name: 'Воронеж', pos: [39.2, 51.661], major: true },
    ],
  },
  {
    code: 'М-2',
    name: 'Крым',
    places: [
      { name: 'Подольск', pos: [37.545, 55.431] },
      { name: 'Серпухов', pos: [37.416, 54.913] },
      { name: 'Тула', pos: [37.618, 54.193], major: true },
      { name: 'Мценск', pos: [36.575, 53.279] },
      { name: 'Орёл', pos: [36.064, 52.97], major: true },
      { name: 'Курск', pos: [36.193, 51.73], major: true },
    ],
  },
  {
    code: 'М-1',
    name: 'Беларусь',
    places: [
      { name: 'Одинцово', pos: [37.278, 55.678] },
      { name: 'Кубинка', pos: [36.703, 55.575] },
      { name: 'Можайск', pos: [36.017, 55.506] },
      { name: 'Гагарин', pos: [34.997, 55.553] },
      { name: 'Вязьма', pos: [34.296, 55.211] },
      { name: 'Ярцево', pos: [32.69, 55.067] },
      { name: 'Смоленск', pos: [32.045, 54.782], major: true },
    ],
  },
]

// mkadExit — точка на кольце в направлении target
function mkadExit(target: LngLat): LngLat {
  const dx = (target[0] - MKAD_CENTER[0]) / MKAD_RX
  const dy = (target[1] - MKAD_CENTER[1]) / MKAD_RY
  const angle = Math.atan2(dy, dx)
  return [MKAD_CENTER[0] + MKAD_RX * Math.cos(angle), MKAD_CENTER[1] + MKAD_RY * Math.sin(angle)]
}

// corridorPath — путь трассы от съезда с МКАД до последнего города
export function corridorPath(corridor: Corridor): LngLat[] {
  return [mkadExit(corridor.places[0].pos), ...corridor.places.map((p) => p.pos)]
}

export const MKAD_RING: LngLat[] = Array.from({ length: 73 }, (_, i) => {
  const angle = (i / 72) * Math.PI * 2
  return [MKAD_CENTER[0] + MKAD_RX * Math.cos(angle), MKAD_CENTER[1] + MKAD_RY * Math.sin(angle)]
})

export const PLACES: Place[] = [MOSCOW, ...CORRIDORS.flatMap((c) => c.places)]

export const NETWORK_BOUNDS: [LngLat, LngLat] = PLACES.reduce<[LngLat, LngLat]>(
  ([min, max], { pos }) => [
    [Math.min(min[0], pos[0]), Math.min(min[1], pos[1])],
    [Math.max(max[0], pos[0]), Math.max(max[1], pos[1])],
  ],
  [
    [Infinity, Infinity],
    [-Infinity, -Infinity],
  ],
)
