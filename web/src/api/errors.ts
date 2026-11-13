import { ApiError } from './client'

// Бэк отвечает на ошибки короткими фразами из internal/handler/error_mapper.go.
// Здесь они переведены для людей; незнакомая фраза показывается как есть
const MESSAGES: Record<string, string> = {
  'invalid json': 'Сервер не понял запрос',
  'record not found': 'Запись не найдена: возможно, её уже удалили',
  forbidden: 'Недостаточно прав',
  'invalid vin': 'VIN — ровно 17 заглавных латинских букв и цифр',
  'vehicle with this vin already exists': 'Машина с таким VIN уже есть',
  'vehicle with this number plate already exists': 'Машина с таким номером уже есть',
  'invalid number plate': 'Номер — 8 или 9 символов, например А123ВС77',
  'invalid car model': 'Укажите модель',
  'invalid status': 'Недопустимый статус',
  'device is already assigned': 'Трекер уже стоит на другой машине',
  'vehicle is busy': 'Машина занята: привязать трекер можно только к свободной',
  'device is active or on maintenance': 'Трекер сейчас недоступен',
  'device is not assigned to this vehicle': 'Этот трекер не привязан к машине',
  'invalid serial number': 'Укажите серийный номер',
  'device with this serial number already exists': 'Трекер с таким серийным номером уже есть',
  'trip is already finished': 'Рейс уже завершён',
  'invalid driver name': 'Имя — от 1 до 35 символов',
  "driver has trips and can't be deleted": 'У водителя есть рейсы, удалить его нельзя',
  'no active trip': 'У машины нет рейса в пути',
  'organization with this name already exists': 'Организация с таким названием уже есть',
  'invalid user name': 'Имя — от 1 до 35 символов',
  'invalid email': 'Проверьте почту',
  'user with this email already exists': 'Пользователь с такой почтой уже есть',
  'forbidden password': 'Пароль — от 8 символов, буквы и цифры, не совпадает с почтой и именем',
  'invalid user role': 'Недопустимая роль',
  'invalid threshold': 'Порог не может быть отрицательным',
  'invalid severity': 'Недопустимая критичность',
  'alert is already acknowledged or resolved': 'Тревога уже принята или закрыта',
  'invalid timestamp': 'Проверьте даты',
  'invalid coords': 'Широта от −90 до 90, долгота от −180 до 180',
  'invalid fuel': 'Топливо — от 0 до 100%',
  'invalid device id': 'Проверьте номер трекера',
  'invalid vehicle id': 'Проверьте машину',
  'invalid driver id': 'Проверьте водителя',
  'service is unavailable, try later': 'Сервис временно недоступен, попробуйте позже',
}

// UserFacingError — ошибка, текст которой уже написан для человека и показывается как есть
export class UserFacingError extends Error {}

// errorText переводит ошибку запроса в фразу для интерфейса
export function errorText(err: unknown): string {
  if (err instanceof UserFacingError) return err.message
  if (err instanceof ApiError) {
    if (MESSAGES[err.message]) return MESSAGES[err.message]
    if (err.status === 403) return MESSAGES.forbidden
    if (err.status >= 500) return 'Ошибка на сервере. Попробуйте ещё раз'
    return err.message
  }
  return 'Нет связи с сервером'
}
