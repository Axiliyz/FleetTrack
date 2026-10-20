// Справочники мока: организация, водители, машины, правила алертов
import type { AlertRule, Driver, Org, Vehicle } from '../../types'
import type { Rng } from '../../lib/random'

export const ORG: Org = { id: 1, name: 'Северный тракт' }

// CURRENT_USER_ID — диспетчер, от имени которого принимаются алерты
export const CURRENT_USER_ID = 1

const DRIVER_NAMES = [
  'Сергей Ковалёв',
  'Андрей Мельников',
  'Ильдар Сафин',
  'Олег Петренко',
  'Дмитрий Орлов',
  'Руслан Гайнуллин',
  'Виктор Лапин',
  'Алексей Жуков',
  'Марина Соколова',
  'Николай Белов',
  'Тимур Хасанов',
  'Павел Зуев',
  'Евгений Крылов',
  'Артём Воронин',
  'Георгий Титов',
  'Константин Ершов',
]

const MODELS = [
  'КАМАЗ 54901',
  'КАМАЗ 5490 Neo',
  'Volvo FH 500',
  'Scania R450',
  'MAN TGX 18.510',
  'Mercedes-Benz Actros 1845',
  'DAF XF 480',
  'Sitrak C7H',
]

// в российских номерах только буквы, похожие на латинские
const PLATE_LETTERS = 'АВЕКМНОРСТУХ'
const REGIONS = ['77', '97', '177', '199', '750', '50', '190']
const VIN_CHARS = 'ABCDEFGHJKLMNPRSTUVWXYZ0123456789'

export function makeDrivers(createdAt: string): Driver[] {
  return DRIVER_NAMES.map((name, i) => ({
    id: i + 1,
    organization_id: ORG.id,
    name,
    created_at: createdAt,
    updated_at: null,
  }))
}

export function makeVehicles(rng: Rng, count: number): Vehicle[] {
  const plates = new Set<string>()
  const vehicles: Vehicle[] = []
  for (let i = 0; i < count; i++) {
    let plate: string
    do {
      const letter = () => rng.pick([...PLATE_LETTERS])
      plate = `${letter()}${rng.int(1, 999).toString().padStart(3, '0')}${letter()}${letter()}${rng.pick(REGIONS)}`
    } while (plates.has(plate))
    plates.add(plate)

    vehicles.push({
      id: i + 1,
      organization_id: ORG.id,
      vin: Array.from({ length: 17 }, () => rng.pick([...VIN_CHARS])).join(''),
      number_plate: plate,
      model: rng.pick(MODELS),
      status: 'IDLE',
    })
  }
  return vehicles
}

export function makeRules(createdAt: string): AlertRule[] {
  const base = { organization_id: ORG.id, enabled: true, created_at: createdAt }
  return [
    { ...base, id: 1, type: 'SPEED_EXCEEDED', name: 'Превышение 90 км/ч', threshold: 90, severity: 'HIGH' },
    { ...base, id: 2, type: 'LOW_FUEL', name: 'Топливо ниже 15%', threshold: 15, severity: 'MEDIUM' },
    // порог в минутах тишины трекера
    { ...base, id: 3, type: 'DEVICE_OFFLINE', name: 'Трекер молчит 3 минуты', threshold: 3, severity: 'CRITICAL' },
  ]
}
