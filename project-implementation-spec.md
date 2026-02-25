## 1. Общая архитектура и стек

- **Бэкенд**: Go + Fiber, REST API.
  - Go 1.22+, Fiber, sqlx/GORM, PostgreSQL, Redis (кэш/сессии), JWT.
  - Монолит с чётким разделением по модулям (auth, subscriptions, analytics и т.д.).
- **Фронтенд (общая кодовая база под мобилку + веб)**:
  - **React Native + Expo + React Native Web (один проект)**.
  - TypeScript, Expo Router (навигация), React Query (запросы к API), Zustand/Redux Toolkit (состояние).
  - Платформы:
    - iOS, Android — нативные билды через Expo.
    - Web — через React Native Web (expo web build).
- Причина выбора:
  - Одна кодовая база под все 3 клиента → быстрее для хакатона.
  - Быстрый старт, много готовых UI-компонентов, минимум нативной боли.

---

## 2. Бэкенд: модули и API

### 2.1. Структура проекта (Go + Fiber)

```text
backend/
  cmd/
    api/
      main.go          // запуск сервера
  internal/
    config/            // загрузка конфига (env)
    db/                // подключение к Postgres, миграции
    middleware/        // auth, логирование, cors
    auth/              // логика аутентификации/авторизации
    users/             // профиль пользователя
    subscriptions/     // подписки, статусы, действия
    sources/           // источники данных (банки, почта, ручной импорт)
    analytics/         // отчёты, графики, агрегации
    forecast/          // прогноз трат
    notifications/     // настройки уведомлений, генерация событий
    recommendations/   // поиск альтернатив
    common/            // утилиты, ошибки, респонсы
```

### 2.2. База данных (PostgreSQL)

Основные таблицы:

- `users`
  - `id`, `email`, `phone`, `password_hash` (или OAuth id), `created_at`.
- `auth_providers`
  - Привязки к Google/Apple/и т.д. (по необходимости).
- `subscription_sources`
  - `id`, `user_id`, `type` (`bank`, `email`, `manual`), `provider` (`sber`, `tinkoff`, `gmail` и т.д.), `status`, `meta`.
- `subscriptions`
  - `id`, `user_id`, `service_name`, `category`, `price`, `currency`,
    `billing_period` (`month`, `year`), `next_charge_at`, `status`.
- `transactions`
  - `id`, `user_id`, `source_id`, `subscription_id?`, `amount`, `currency`, `charged_at`, `raw_description`.
- `usage_events`
  - `id`, `user_id`, `subscription_id`, `date`, `metric` (например `used` / количество минут / заказов).
- `notification_settings`
  - `id`, `user_id`, `type` (`upcoming_charge`, `price_increase`, `low_usage`, `offers`),
    `channels` (json: push/email/telegram), `enabled`.
- `notifications`
  - `id`, `user_id`, `subscription_id?`, `type`, `payload` (json), `status` (`sent`, `read`).
- `recommendation_alternatives`
  - `id`, `category`, `service_name`, `price`, `billing_period`, `description`, `meta`.

### 2.3. Основные REST эндпоинты (черновая спецификация)

Базовый префикс: `/api/v1`.

- **Auth**
  - `POST /auth/register` — регистрация по почте/телефону.
  - `POST /auth/login` — логин.
  - `GET /auth/me` — текущий пользователь.

- **Источники данных (банки, почта, ручной импорт)**
  - `GET /sources` — список источников пользователя.
  - `POST /sources` — создать источник (тип/провайдер, начальный статус).
  - `POST /sources/:id/connect` — старт/фиксация подключения (для реального OAuth или симуляции).
  - `GET /sources/:id/status` — статус интеграции.
  - **MVP-фича**: `POST /sources/:id/upload` — загрузка файла выписки/примеров транзакций (имитация банка) → парсинг на бэке в `transactions`.

- **Подписки**
  - `GET /subscriptions` — список всех подписок пользователя (с фильтрами по статусу/категории).
  - `POST /subscriptions` — создать (ручной ввод).
  - `GET /subscriptions/:id` — деталь.
  - `PUT /subscriptions/:id` — обновить.
  - `DELETE /subscriptions/:id` — логическое удаление/отмена (статус `cancelled`).
  - `POST /subscriptions/:id/pause` — поставить на паузу (статус `paused`, период).
  - `POST /subscriptions/:id/resume` — снять с паузы.
  - `POST /subscriptions/:id/usage` — зафиксировать использование (создать `usage_event`).

- **Аналитика**
  - `GET /analytics/summary?period=month|year` — общая сумма, динамика.
  - `GET /analytics/categories?period=...` — разбивка по категориям.
  - `GET /analytics/services?period=...` — топ дорогих сервисов.
  - `GET /analytics/subscription/:id` — график трат + использования по конкретной подписке.

- **Прогноз**
  - `GET /forecast/year` — прогноз на год: общая сумма + сценарий экономии.

- **Уведомления**
  - `GET /notifications/settings` — получить настройки.
  - `PUT /notifications/settings` — обновить настройки.
  - `GET /notifications` — список уведомлений.
  - `PUT /notifications/:id/read` — пометить как прочитанное.

- **Рекомендации**
  - `GET /recommendations` — список по категориям/фильтрам.
  - `GET /recommendations/:subscriptionId` — альтернативы для конкретной подписки.

### 2.4. Внутренняя логика (минимально, но жизнеспособно)

- **Парсинг транзакций/писем**:
  - MVP: принимаем либо загруженный файл (csv/json примера выписки), либо мок-данные.
  - Маппим по маскам (`Netflix`, `Spotify`, `Yandex Plus` и т.п.) → создаём/обновляем `subscriptions`.
  - Для хакатона: делаем таблицу `patterns` с регулярками/строками для маппинга.

- **Прогноз трат**:
  - Для MVP: берём текущий набор подписок, нормализуем до `стоимость в месяц` и умножаем на 12.
  - Плюс можно корректировать по историям цен, если успеем (но можно не успеть — достаточно простого прогноза).

- **Анализ использования**:
  - На основе `usage_events` считаем:
    - Кол-во дней использования за месяц.
    - Делим цену на кол-во дней/часов → показываем «стоимость дня пользования».
  - Для MVP: только счётчик дней «пользовался/не пользовался».

- **Уведомления**:
  - Фоновый воркер (cron/горутинка по таймеру):
    - Раз в N минут/часов проверяет подписки:
      - Ближайшие списания → создаёт `notifications` и пушит через выбранные каналы (для демо можно ограничиться push/web-toast внутри приложения).
      - Изменения цены (если есть исторические данные) — опционально.

---

## 3. Фронтенд: структура и реализация (React Native + Expo + Web)

### 3.1. Стэк и структура

- **Стек**:
  - Expo (Managed Workflow).
  - React Native + React Native Web.
  - TypeScript.
  - Expo Router / React Navigation.
  - React Query (запросы).
  - Zustand или Redux Toolkit (глобальное состояние: пользователь, токены, флаги онбординга).

- **Структура проекта (условно)**:

```text
app/                    // Expo Router (или src/ и отдельная навигация)
  (auth)/
    login.tsx
    register.tsx
    onboarding.tsx
  (main)/
    dashboard.tsx
    subscriptions/
      index.tsx         // список подписок
      [id].tsx          // деталь подписки
    analytics/
      index.tsx
    recommendations/
      index.tsx
    notifications/
      index.tsx
    profile/
      index.tsx
shared/
  components/           // UI-компоненты, кроссплатформенные
  hooks/                // кастомные хуки
  api/                  // клиенты к REST (axios/fetch + React Query)
  store/                // Zustand/Redux
  theme/                // цвета, типографика
```

### 3.2. Основные экраны (mapping с юзер-флоу)

- `(auth)/onboarding.tsx` — UF-01.
- `(auth)/login.tsx`, `(auth)/register.tsx` — аутентификация.
- `(main)/dashboard.tsx` — UF-03: дашборд подписок.
- `(main)/subscriptions/index.tsx` — список с поиском/фильтрами.
- `(main)/subscriptions/[id].tsx` — UF-04: деталь подписки, действия (отмена, пауза, отметить использование).
- `(main)/analytics/index.tsx` — UF-05: аналитика + блок прогноза.
- `(main)/notifications/index.tsx` — UF-06: настройки уведомлений + история.
- `(main)/recommendations/index.tsx` — UF-07: рекомендации и альтернативы.
- FAB/кнопка `Добавить подписку` → модалка/страница с формой (UF-10).

### 3.3. Клиент для API

- Один модуль `shared/api/client.ts`:
  - Базовый `axios`/`fetch` клиент со:
    - Базовым URL (`/api/v1`).
    - Подставкой JWT из стора.
    - Обработкой ошибок (401 → логаут).
- Модули:
  - `shared/api/auth.ts`
  - `shared/api/subscriptions.ts`
  - `shared/api/analytics.ts`
  - `shared/api/notifications.ts`
  - `shared/api/recommendations.ts`

Каждый модуль экспортирует функции типа `getSubscriptions`, `createSubscription`, которые уже обёрнуты в React Query хуки (`useSubscriptionsQuery`, `useCreateSubscriptionMutation` и т.д.).

### 3.4. Состояние

- Глобальное через Zustand/Redux:
  - `authStore`: токен, пользователь, состояние логина.
  - `uiStore`: флаги онбординга, выбранный период (месяц/год).
- Остальные данные — через React Query (серверный стейт).

### 3.5. Специфика под web

- Используем React Native Web:
  - Вёрстка на `View`, `Text`, `Pressable` и т.д.
  - Для веба добавляем адаптивные стили (flex/Grid через styled-components/StyleSheet).
- Навигация:
  - Expo Router даёт урлы `/<route>` для веба — удобно для демки на хакатоне.

---

## 4. MVP-объём на 14 дней (реалистично)

- **Дни 1–2**:
  - Уточнение фич, финализация дизайн-спека (то, что уже есть).
  - Подъём скелета бэка (Go + Fiber, Postgres), базовый Auth.
  - Создание проекта Expo (React Native + Web), базовая навигация и тема.

- **Дни 3–6**:
  - Бэкенд: модели `users`, `subscriptions`, `subscription_sources`, `transactions`, `usage_events`.
  - API: `/auth`, `/subscriptions`, `/analytics/summary`, `/analytics/categories`.
  - Фронт: онбординг, логин/регистрация, дашборд, список и деталь подписки (без тяжёлой аналитики).

- **Дни 7–10**:
  - Бэкенд: простой прогноз (`/forecast/year`), базовая аналитика по подписке, загрузка файла выписки (`/sources/:id/upload`) + парсер по паттернам.
  - Фронт: экран аналитики, блок прогноза, форма ручного добавления подписки, отметка использования.

- **Дни 11–12**:
  - Уведомления (минимум: модель + API + имитация пушей/веб-тостов).
  - Экран уведомлений и настроек на фронте.
  - Базовые рекомендации: статический список альтернатив по категориям.

- **Дни 13–14**:
  - Полировка UX, анимации, демо-данные.
  - Сборка билдов (web + одно мобильное), подготовка презентации.

