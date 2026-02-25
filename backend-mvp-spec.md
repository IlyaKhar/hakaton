## Backend MVP (Go + Fiber) под ТЗ

Цель: реализовать минимально жизнеспособный бэкенд, который закрывает все пункты ТЗ для демо, с явным разделением на **реализовано в MVP** и **имитировано/упрощено**.

ТЗ (кратко): единая панель подписок, автоматический сбор (банки/почта), аналитика, анализ использования, прогноз на год, умные уведомления, рекомендации альтернатив, быстрая отмена/пауза.

---

## 1. Стек и общая структура

- **Стек**:
  - Go 1.22+, Fiber.
  - PostgreSQL.
  - JWT (аутентификация).
  - Background-джоб/cron для уведомлений (простой таймер/горутинка).

- **Структура проекта** (по модулям):

```text
backend/
  cmd/api/main.go
  internal/
    config/
    db/
    middleware/
    auth/
    users/
    subscriptions/
    sources/
    analytics/
    forecast/
    notifications/
    recommendations/
    common/
```

---

## 2. Модели БД (MVP)

### 2.1. Пользователи

- `users`
  - `id` (uuid)
  - `email` (nullable)
  - `phone` (nullable)
  - `password_hash`
  - `created_at`

### 2.2. Источники данных (банки, почта, ручной импорт)

- `subscription_sources`
  - `id` (uuid)
  - `user_id` (fk → users)
  - `type` (`bank`, `email`, `manual`)
  - `provider` (`sber`, `tinkoff`, `yoomoney`, `gmail`, `yandex`, `mailru`, `manual`)
  - `status` (`pending`, `connected`, `error`)
  - `meta` (jsonb — токены/настройки, для MVP можно не использовать)
  - `created_at`, `updated_at`

### 2.3. Подписки и транзакции

- `subscriptions`
  - `id` (uuid)
  - `user_id`
  - `service_name` (`Netflix`, `Spotify`, `Яндекс Плюс` и т.д.)
  - `category` (`streaming`, `software`, `delivery`, `cloud`, `other`)
  - `price` (numeric)
  - `currency` (string, `RUB` по умолчанию)
  - `billing_period` (`month`, `year`, `other`)
  - `next_charge_at` (timestamp, nullable)
  - `status` (`active`, `paused`, `cancelled`, `trial`, `pending_cancel`)
  - `created_at`, `updated_at`

- `transactions`
  - `id` (uuid)
  - `user_id`
  - `source_id` (fk → subscription_sources)
  - `subscription_id` (nullable, fk → subscriptions)
  - `amount` (numeric)
  - `currency`
  - `charged_at` (timestamp)
  - `raw_description` (text)

### 2.4. Использование подписок

- `usage_events`
  - `id` (uuid)
  - `user_id`
  - `subscription_id`
  - `date` (date)
  - `metric` (string, для MVP — просто `used`)

### 2.5. Уведомления и настройки

- `notification_settings`
  - `id` (uuid)
  - `user_id`
  - `type` (`upcoming_charge`, `price_increase`, `low_usage`, `offers`)
  - `channels` (jsonb: `{ "push": true, "email": false }`)
  - `enabled` (bool)

- `notifications`
  - `id` (uuid)
  - `user_id`
  - `subscription_id` (nullable)
  - `type`
  - `payload` (jsonb)
  - `status` (`created`, `sent`, `read`)
  - `created_at`, `read_at`

### 2.6. Рекомендации

- `recommendation_alternatives`
  - `id` (uuid)
  - `category`
  - `service_name`
  - `price`
  - `billing_period`
  - `description`
  - `meta` (jsonb — ссылки и т.п.)

---

## 3. Соответствие пунктам ТЗ → реализации в бэкенде

### 3.1. Централизованное отображение всех активных подписок

- **Что нужно по ТЗ**: единая панель с подписками разных типов.
- **Как делаем на бэке (MVP)**:
  - Таблица `subscriptions`.
  - Эндпоинты:
    - `GET /api/v1/subscriptions`
      - Возвращает список подписок текущего пользователя (фильтры по `status`, `category`).
    - `GET /api/v1/subscriptions/:id`
      - Детальная информация, включая историю транзакций и использования.
    - `POST /api/v1/subscriptions`
      - Ручное создание подписки (для случаев, когда автоматический сбор не сработал).

### 3.2. Автоматический сбор данных о подписках (банки и почта)

- **Что нужно по ТЗ**:
  - Интеграция с платёжными системами и/или парсинг почты.
- **MVP-подход** (важно для жюри):
  - Реальные банковские/почтовые интеграции тяжёлые, поэтому:
    - Делаем **структуру** и API так, как будто интеграция есть.
    - Для демо используем:
      - Мок-источники (`subscription_sources`).
      - Загрузку файла/текста с примером выписки/писем, который мы парсим.

- **Эндпоинты**:
  - `GET /api/v1/sources`
    - Список источников пользователя.
  - `POST /api/v1/sources`
    - Создание источника: `{ type, provider }`.
    - Для банка/почты выставляем `status = pending`.
  - `POST /api/v1/sources/:id/mark-connected`
    - В MVP имитируем успешное подключение (как будто OAuth прошёл).
    - Ставим `status = connected`.
  - `POST /api/v1/sources/:id/upload`
    - **Ключевой MVP-эндпоинт**: принимаем файл (csv/json) или текстовый блок с транзакциями/чеками.
    - Логика:
      1. Парсим записи.
      2. Ищем по `raw_description` паттерны сервисов (таблица паттернов/константы в коде).
      3. Создаём/обновляем `subscriptions` и `transactions`.

> Для презентации можно сказать: «Сегодня это сделано через загрузку выписки/примеров чеков, но архитектура уже готова для реального OAuth к банкам и почте».

### 3.3. Визуализация аналитики по затратам (категории, сервисы, периоды)

- **Что нужно по ТЗ**:
  - Суммы по категориям/сервисам, разные периоды.
- **Эндпоинты**:
  - `GET /api/v1/analytics/summary?period=month|year&from=&to=`
    - Возвращает:
      - Общую сумму трат за период.
      - Динамику к предыдущему периоду.
  - `GET /api/v1/analytics/categories?period=...`
    - Группировка трат по `category`.
  - `GET /api/v1/analytics/services?period=...`
    - Группировка по `service_name`, топ-Х самых дорогих.

- **Реализация**:
  - SQL-агрегации по `transactions.amount` с фильтрацией по пользователю и периоду.

### 3.4. Анализ использования подписок (на основе данных пользователя)

- **Что нужно по ТЗ**:
  - Учитывать отметки пользователя: смотрел ли контент, пользовался ли софтом и т.п.
- **MVP**:
  - Пользователь кликает «пользовался сегодня» → создаём запись в `usage_events`.
  - На бэке:
    - `POST /api/v1/subscriptions/:id/usage`
      - Тело: `{ date?, metric? }`, по умолчанию `metric = "used"`, `date = today`.
    - `GET /api/v1/subscriptions/:id/usage?from=&to=`
      - Для аналитики: отдаём количество дней/событий использования за период.

- **Связь с аналитикой**:
  - В `GET /api/v1/analytics/subscription/:id` можно возвращать:
    - трату по месяцам,
    - количество usage-событий по месяцам.

### 3.5. Прогноз трат на год вперёд

- **Что нужно по ТЗ**:
  - Прогноз на год вперёд.
- **MVP-логика**:
  - Нормализуем каждую подписку до `monthly_cost`:
    - если `billing_period = month` → `monthly_cost = price`.
    - если `billing_period = year` → `monthly_cost = price / 12`.
  - Прогноз на год: `sum(monthly_cost) * 12`.
  - Плюс опционально сценарий экономии: если исключить подписки с низкой активностью.

- **Эндпоинт**:
  - `GET /api/v1/forecast/year`
    - Ответ:
      - `total_year_cost`
      - `baseline_monthly_cost`
      - `economy_if_cancel_low_usage`

### 3.6. Умные уведомления (списание, рост цены, неактивность)

- **Что нужно по ТЗ**:
  - Предупреждение о предстоящем списании.
  - Повышение цены.
  - Неактивность.

- **MVP**:
  - Реальные интеграции с пушами/почтой можно упростить до:
    - Запись в `notifications`.
    - Возврат списка на фронт + in-app уведомление.

- **Настройки**:
  - `GET /api/v1/notifications/settings`
  - `PUT /api/v1/notifications/settings`

- **Уведомления**:
  - `GET /api/v1/notifications`
  - `PUT /api/v1/notifications/:id/read`

- **Фоновая джоба** (псевдо-код логики):
  - Каждые N минут:
    - Ищем подписки, у которых `next_charge_at` в течение X дней → создаём уведомления `upcoming_charge`.
    - Ищем подписки с очень редкими `usage_events` → `low_usage`.
    - Повышение цены можно смоделировать (для MVP — заранее заложенные изменения/демо-данные).

### 3.7. Рекомендации альтернативных сервисов

- **Что нужно по ТЗ**:
  - Поиск и рекомендации более выгодных сервисов в той же категории.

- **MVP**:
  - Таблица `recommendation_alternatives` с заранее заполненными данными.
  - Логика на бэке:
    - По категории и цене текущей подписки подбираем альтернативы с меньшей ценой.

- **Эндпоинты**:
  - `GET /api/v1/recommendations?category=&max_price=`
  - `GET /api/v1/recommendations/subscription/:id`
    - Находит категорию и цену подписки → возвращает список альтернатив + экономию.

### 3.8. Быстрая отмена / приостановка подписки

- **Что нужно по ТЗ**:
  - Быстрая отмена/пауза с перенаправлением в сервис или генерацией письма.

- **MVP на бэке**:
  - Статусы в `subscriptions`:
    - `status = "pending_cancel"`, `status = "cancelled"`, `status = "paused"`.
  - Поля:
    - `cancel_url` (nullable) — ссылка на страницу управления подпиской.
    - `support_email` (nullable) — почта поддержки сервиса.

- **Эндпоинты**:
  - `POST /api/v1/subscriptions/:id/cancel-intent`
    - Меняет статус на `pending_cancel`.
    - Возвращает:
      - `cancel_url`
      - сгенерированный текст письма (на лету) для фронта.
  - `POST /api/v1/subscriptions/:id/confirm-cancel`
    - Пользователь подтвердил, что отменил у провайдера → статус `cancelled`.
  - `POST /api/v1/subscriptions/:id/pause`
    - Тело: `{ from, to, analytics_only }`.
    - Меняем статус на `paused` (для `analytics_only = true` — только у нас).

---

## 4. Минимальный набор эндпоинтов для MVP

Сводно:

- Auth: `POST /auth/register`, `POST /auth/login`, `GET /auth/me`.
- Sources: `GET /sources`, `POST /sources`, `POST /sources/:id/mark-connected`, `POST /sources/:id/upload`.
- Subscriptions: `GET /subscriptions`, `GET /subscriptions/:id`, `POST /subscriptions`, `PUT /subscriptions/:id`, `POST /subscriptions/:id/usage`, `POST /subscriptions/:id/cancel-intent`, `POST /subscriptions/:id/confirm-cancel`, `POST /subscriptions/:id/pause`.
- Analytics: `GET /analytics/summary`, `GET /analytics/categories`, `GET /analytics/services`, `GET /analytics/subscription/:id`.
- Forecast: `GET /forecast/year`.
- Notifications: `GET /notifications/settings`, `PUT /notifications/settings`, `GET /notifications`, `PUT /notifications/:id/read`.
- Recommendations: `GET /recommendations`, `GET /recommendations/subscription/:id`.

Это закрывает все буллеты ТЗ в формате **MVP, готового к показу жюри**.

