# FleetTrack — архитектура backend (конспект)

Дата: 22.06.2026

---

# Главная идея

Backend — это не набор файлов, а система ответственности.

Главный вопрос при добавлении кода:

> Кто должен этим заниматься?

Не:
> Куда мне впихнуть этот код?

А:

- HTTP → Handler
- Бизнес-правила → Service
- Хранение → Repository
- Сквозные вещи → Middleware
- Логи → Logger

---

# Общий pipeline

ESP / датчик
      |
      v
 HTTP Request
      |
      v
 Handler
      |
      v
 Service
      |
      v
 Repository
      |
      v
 База / файл / память

Обратный путь:

Repository
      |
      v
 Service
      |
      v
 Handler
      |
      v
 JSON Response

---

# 1. Handler

Handler работает только с HTTP.

Он знает:

- POST / GET
- JSON
- headers
- status codes
- request context

Он НЕ знает:

- как валидировать координаты
- где лежат данные
- какая БД используется

---

# 2. Service

Service — бизнес-логика.

Он решает:

- валидировать данные
- ставить ReceivedAt
- вызывать repository

Он НЕ знает:

- HTTP
- JSON
- Postman

---

# 3. Repository

Repository отвечает за хранение.

Сегодня:

- MemoryRepository

Завтра:

- PostgresRepository
- MongoRepository

Service при этом не меняется.

---

# Зачем интерфейсы

Service зависит от интерфейса, а не от конкретной реализации.

Это позволяет менять:

- БД
- файл
- память
- внешние сервисы

без переписывания бизнес-логики.

---

# Dependency Injection

main.go собирает приложение:

- создаёт Repository
- создаёт Service
- создаёт Handler

Зависимости не создаются внутри слоёв.

---

# 4. Model и DTO

## Model

Реальная сущность системы.

Например:

- Telemetry

## DTO

Формат передачи данных наружу.

Например:

- APIResponse
- TelemetryResponse

DTO — это не бизнес-сущность.

---

# 5. APIResponse

Единый формат ответа API.

Успех:

```json
{
  "status": "success",
  "message": "Telemetry saved",
  "request_id": "...",
  "data": {}
}
```

Ошибка:

```json
{
  "status": "error",
  "message": "invalid coordinates",
  "request_id": "..."
}
```

---

# 6. DeviceTimestamp и ReceivedAt

## DeviceTimestamp

Когда датчик отправил данные.

## ReceivedAt

Когда сервер получил данные.

Нужны оба времени.

Так можно увидеть задержку доставки.

---

# 7. Middleware и RequestID

Каждому запросу выдаётся UUID.

Он сохраняется в Context.

Благодаря этому можно:

- искать запрос в логах
- связывать ошибки с запросом
- трассировать выполнение

---

# 8. Ошибки

Используем заранее объявленные ошибки:

- ErrInvalidID
- ErrInvalidCoords

Проверяем через errors.Is().

---

# 9. Logger

Уровни логирования:

- DEBUG
- INFO
- WARN
- ERROR

Пример:

- INFO → телеметрия сохранена
- ERROR → неверные координаты

---

# 10. Почему нельзя всё делать в Handler

Иначе получится God Object.

Проблемы:

- сложно тестировать
- сложно поддерживать
- сложно расширять

---

# Финальная архитектура FleetTrack

Middleware
    |
    v

ESP -> Handler -> Service -> Repository
                   |
                   v
                 Logger

---

# Самопроверка

1. Почему Service не должен знать про HTTP?
2. Почему Repository это интерфейс?
3. Почему DTO не лежит рядом с Telemetry?
4. Где ставить ReceivedAt?
5. Где создавать RequestID?
6. Почему нельзя создавать Repository внутри Service?
7. Почему ответ JSON делает Handler?
