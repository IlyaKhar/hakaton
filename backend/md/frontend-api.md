# API для фронтенда (MVP подписочного сервиса)

Базовый URL для всех запросов:

```text
http://localhost:3000/api/v1
```

Все эндпоинты, кроме `auth/register`, `auth/login`, `auth/forgot-password`, `auth/verify-code`, `auth/reset-password` и публичного OAuth‑callback, требуют заголовок:

```text
Authorization: Bearer <JWT>
```

---

## 1. Auth / Профиль

### Регистрация и вход

| Метод | URL              | Описание                |
|-------|------------------|-------------------------|
| POST  | `/auth/register` | Регистрация             |
| POST  | `/auth/login`    | Логин, получить токен   |

**Тела запросов**

```json
// POST /auth/register
{ "email": "user@mail.com", "password": "123456" }

// POST /auth/login
{ "email": "user@mail.com", "password": "123456" }
```

Ответ `POST /auth/login` содержит поле `token` — его сохраняем и потом шлём в заголовке `Authorization`.

### Профиль

| Метод | URL                     | Описание                    |
|-------|-------------------------|-----------------------------|
| GET   | `/auth/me`             | Текущий пользователь        |
| PUT   | `/auth/profile`        | Обновить имя/почту          |
| POST  | `/auth/change-password`| Сменить пароль              |

```json
// PUT /auth/profile
{ "name": "Иван", "email": "new@mail.com" }

// POST /auth/change-password
{ "oldPassword": "123456", "newPassword": "qwerty" }
```

### Восстановление пароля

| Метод | URL                      | Body                                   |
|-------|--------------------------|----------------------------------------|
| POST  | `/auth/forgot-password` | `{ "email" }`                          |
| POST  | `/auth/verify-code`     | `{ "email", "code" }`                  |
| POST  | `/auth/reset-password`  | `{ "email", "code", "newPassword" }`   |

---

## 2. Дашборд и аналитика (главный экран)

### Дашборд (главная)

| Метод | URL          | Описание                         |
|-------|--------------|----------------------------------|
| GET   | `/dashboard` | Сводка по подпискам/картам/тратам|

**Пример ответа**

```json
{
  "totalSubscriptions": 4,
  "activeSubscriptions": 3,
  "pausedSubscriptions": 1,
  "canceledSubscriptions": 0,
  "totalCards": 2,
  "defaultCardLast4": "1234",
  "monthSpend": 3999.99,
  "currency": "RUB",
  "topCategories": [ /* CategoryStat */ ],
  "topServices": [ /* ServiceStat */ ]
}
```

Использование:
- карточка **“Все операции”** → `monthSpend`, `currency`
- карточка **“Все подписки”** → `totalSubscriptions`, `activeSubscriptions`
- карточка **“Все карты”** → `totalCards`, `defaultCardLast4`

### Аналитика

| Метод | URL                        | Описание                     |
|-------|----------------------------|------------------------------|
| GET   | `/analytics/summary`      | Суммарная аналитика          |
| GET   | `/analytics/categories`   | Суммы по категориям           |
| GET   | `/analytics/services`     | Суммы по сервисам             |

`period` (query) можно не указывать — по умолчанию `month`.

---

## 3. Подписки

### Список и деталка

| Метод | URL                     | Описание              |
|-------|-------------------------|-----------------------|
| GET   | `/subscriptions`       | Список подписок       |
| GET   | `/subscriptions/{id}`  | Одна подписка         |

Ответ элемента:

```json
{
  "id": "...",
  "userId": "...",
  "serviceName": "Spotify",
  "category": "music",
  "price": 999.0,
  "currency": "RUB",
  "billingPeriod": "month",
  "nextChargeAt": "2026-02-28T00:00:00Z",
  "status": "active"
}
```

### Создание/редактирование

| Метод | URL                    | Описание        |
|-------|------------------------|-----------------|
| POST  | `/subscriptions`      | Создать         |
| PUT   | `/subscriptions/{id}` | Обновить        |

**Body для POST/PUT**

```json
{
  "serviceName": "Spotify",
  "category": "music",
  "price": 999.0,
  "currency": "RUB",
  "billingPeriod": "month",
  "nextChargeAt": "2026-02-28T00:00:00Z",
  "status": "active"
}
```

### Действия (отключить / заморозить)

| Метод | URL                                          | Описание                    |
|-------|----------------------------------------------|-----------------------------|
| POST  | `/subscriptions/{id}/cancel-intent`         | (опционально) intent        |
| POST  | `/subscriptions/{id}/confirm-cancel`        | Отключить                   |
| POST  | `/subscriptions/{id}/pause`                 | Заморозить                  |

На фронте можно сразу вызывать `confirm-cancel`, если не нужен промежуточный шаг.

---

## 4. Транзакции (“Операции”)

### Список операций

| Метод | URL              | Описание                       |
|-------|------------------|--------------------------------|
| GET   | `/transactions` | Все транзакции пользователя    |

Query:
- `from` (опц.) — `YYYY-MM-DD`
- `to` (опц.) — `YYYY-MM-DD`

**Ответ**

```json
[
  {
    "id": "...",
    "userId": "...",
    "sourceId": "...",
    "subscriptionId": "...",
    "amount": 999.0,
    "currency": "RUB",
    "chargedAt": "2026-02-26T15:57:00Z",
    "rawDescription": "Spotify *1234 ..."
  }
]
```

### Детали операции

| Метод | URL                      | Описание          |
|-------|--------------------------|-------------------|
| GET   | `/transactions/{id}`    | Одна транзакция   |

Используется на экране “детали операции”.

---

## 5. Источники (банки/почта)

### Список источников и провайдеров

| Метод | URL                     | Описание                       |
|-------|-------------------------|--------------------------------|
| GET   | `/sources`             | Список источников пользователя |
| GET   | `/sources/providers`   | Провайдеры (банки, почта)      |

Ответ `/sources/providers`:

```json
{
  "banks": [
    { "id": "sberbank", "name": "Сбербанк", "type": "bank", "icon": "sberbank" },
    ...
  ],
  "email": [
    { "id": "mailru", "name": "Mail.ru", "type": "email", "icon": "mailru" },
    { "id": "yandex", "name": "Яндекс.Почта", "type": "email", "icon": "yandex" },
    { "id": "gmail", "name": "Gmail", "type": "email", "icon": "gmail" }
  ]
}
```

### Создание источника (fallback, без OAuth)

| Метод | URL          | Описание         |
|-------|--------------|------------------|
| POST  | `/sources`  | Создать источник |

```json
{
  "type": "email",          // "email" | "bank" | "manual"
  "provider": "mailru",     // "mailru" | "gmail" | ...
  "meta": {
    "email": "user@mail.ru"
  }
}
```

### Подключение почты через OAuth (Gmail/Mail.ru)

1. **Получить URL авторизации**

| Метод | URL                                      |
|-------|------------------------------------------|
| GET   | `/sources/oauth/{provider}/authorize`   |

`provider`: `gmail` или `mailru`.

Ответ:

```json
{ "authorizationUrl": "https://accounts.google.com/..." }
```

Фронт открывает `authorizationUrl` в браузере или WebView.

2. **Callback**

Провайдер сам вызывает:

```text
GET /api/v1/sources/oauth/{provider}/callback?code=...&state=...
```

Фронту **ничего делать не надо** — backend создаёт запись в `sources`.

3. **После авторизации**

Фронт дергает:

```http
GET /api/v1/sources
```

и видит новый источник с `type: "email"`, `provider: "gmail"/"mailru"`, `status: "connected"`.

---

## 6. Карты

### Список и добавление

| Метод | URL                | Описание          |
|-------|--------------------|-------------------|
| GET   | `/payment-cards`  | Список карт       |
| POST  | `/payment-cards`  | Добавить карту    |
| DELETE| `/payment-cards/{id}` | Удалить       |

**Пример Body для добавления**

```json
{
  "cardNumber": "4111111111111111",
  "expiryMonth": 12,
  "expiryYear": 2027,
  "holderName": "IVAN IVANOV",
  "isDefault": true
}
```

Номер карты не сохраняется полностью — бэк сам вытащит `lastFour`, маску и тип.

---

## 7. Уведомления (экран “Уведомления”)

### Получить настройки

| Метод | URL                          |
|-------|------------------------------|
| GET   | `/notifications/settings`   |

Если настроек ещё нет, бэк вернёт **дефолтный набор** под макет:

```json
[
  {
    "type": "billing_3_days",
    "enabled": false,
    "channels": { "push": true, "email": false }
  },
  {
    "type": "billing_1_day",
    "enabled": true,
    "channels": { "push": true, "email": false }
  },
  {
    "type": "trial_ending",
    "enabled": true,
    "channels": { "push": true, "email": true }
  },
  {
    "type": "budget_control",
    "enabled": false,
    "channels": {
      "limit": 3000.0,
      "currency": "RUB",
      "notifyOn80": true,
      "notifyOnExceeded": true,
      "push": true,
      "email": false
    }
  }
]
```

Маппинг:
- `billing_3_days` → “За 3 дня до списания”
- `billing_1_day`  → “За 1 день до списания”
- `trial_ending`   → “Окончание пробного периода”
- `budget_control` → “Контроль бюджета”

Формат уведомлений:
- `channels.push` / `channels.email`:
  - `true/false` → Push
  - `false/true` → Email
  - `true/true`  → Push + Email

### Сохранить настройки

| Метод | URL                          |
|-------|------------------------------|
| PUT   | `/notifications/settings`   |

Body — **полный массив** настроек (как вернул `GET`, но с изменениями).

```json
[
  {
    "type": "billing_3_days",
    "enabled": true,
    "channels": { "push": true, "email": false }
  },
  {
    "type": "budget_control",
    "enabled": true,
    "channels": {
      "limit": 5000,
      "currency": "RUB",
      "notifyOn80": true,
      "notifyOnExceeded": false,
      "push": true,
      "email": true
    }
  }
]
```

### Лента уведомлений (если нужна)

| Метод | URL                      | Описание               |
|-------|--------------------------|------------------------|
| GET   | `/notifications`        | Список уведомлений     |
| PUT   | `/notifications/{id}/read` | Пометить прочитанным |

---

## 8. Рекомендации (аналоги подписок)

| Метод | URL                                      | Описание                             |
|-------|------------------------------------------|--------------------------------------|
| GET   | `/recommendations`                      | По категории/цене                    |
| GET   | `/recommendations/subscription/{id}`    | Аналоги для конкретной подписки      |

Можно использовать на экране деталки подписки в блоке “Аналоги дешевле”.

