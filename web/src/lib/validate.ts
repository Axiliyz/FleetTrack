// Проверки форм повторяют internal/validator: человек узнаёт об ошибке сразу, а не после запроса.
// Окончательное слово всё равно за бэком — его ответ тоже показывается в форме

const VIN_RE = /^[A-Z0-9]{17}$/

// normalizeVin: VIN пишут как попало — с пробелами и строчными буквами
export const normalizeVin = (vin: string) => vin.replace(/\s/g, '').toUpperCase()

export function vinProblem(vin: string): string | null {
  return VIN_RE.test(vin) ? null : 'VIN — ровно 17 латинских букв и цифр'
}

export const normalizePlate = (plate: string) => plate.replace(/\s/g, '').toUpperCase()

// номер считается в символах, а не в байтах: «А123ВС77» кириллицей — тоже 8
export function plateProblem(plate: string): string | null {
  const n = [...plate].length
  return n >= 8 && n <= 9 ? null : 'Номер — 8 или 9 символов без пробелов, например А123ВС77'
}

export function passwordProblem(password: string, email: string, name: string): string | null {
  if ([...password].length < 8 || new TextEncoder().encode(password).length > 72) return 'Пароль — от 8 до 72 символов'
  if (!/\p{L}/u.test(password) || !/\p{N}/u.test(password)) return 'В пароле нужны и буквы, и цифры'
  const lower = password.toLowerCase()
  if (lower === email.toLowerCase() || lower === name.toLowerCase()) {
    return 'Пароль не должен совпадать с почтой или именем'
  }
  return null
}
