## Backend MVP TODO (Go + Fiber + Supabase)

Это чек-лист, чтобы ты мог сам реализовать бэкенд по уже готовому скелету.
Ниже — шаги в порядке, в котором логично двигаться.
Мы используем **Supabase** как готовый Postgres в облаке (без их встроенного auth — авторизацию делаем сами).

---

## 0. Общая архитектура проекта (кратко)

- **Структура (из `backend-mvp-spec.md`, строки 20–34):**

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

- **Что за что отвечает (по папкам):**
  - `cmd/api/main.go`  
    - Точка входа. Загружает конфиг, собирает DSN для Supabase, коннектится к БД, создаёт сервер и запускает `Listen`.
  - `internal/config`  
    - Работа с `.env`.  
    - Файл `config.go`: структура `Config` + функция `Load()`, которая читает `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `PORT`, `JWT_SECRET`, `REFRESH_SECRET`, `BASE_URL` и т.п.
  - `internal/db`  
    - Подключение к Postgres (Supabase / pooler).  
    - Файл `db.go`: функция `Connect(dsn string) (*sql.DB, error)`.
  - `internal/middleware`  
    - Общие middleware для Fiber: логирование, recover, CORS, **auth-middleware для JWT**.  
    - Здесь будет файл `auth.go`, который:
      - достаёт токен из заголовка `Authorization: Bearer ...`,
      - валидирует его через пакет `internal/auth`,
      - кладёт `userID` в контекст Fiber.
  - `internal/auth`  
    - Логика, не завязанная на HTTP: генерация и валидация JWT, возможно, вспомогательные функции для паролей.  
    - Файл `jwt.go`: `GenerateToken(userID string)`, `ParseToken(token string)`.  
    - Можно добавить `password.go` с обёртками над `bcrypt`.
  - `internal/users`  
    - Работа с таблицей `users` в Supabase.  
    - `model.go`: структура `User`.  
    - `repository.go`: `CreateUser`, `GetUserByEmail`, `GetUserByID`.  
    - Эти функции используют `*sql.DB` из `internal/db`.
  - `internal/subscriptions`  
    - Вся логика по таблице `subscriptions`: CRUD, cancel/pause, связь с `transactions`.  
    - `model.go`, `repository.go`, `http.go` (хэндлеры и `RegisterRoutes`).
  - `internal/sources`  
    - Таблица `subscription_sources` + endpoints `/sources` и `/sources/:id/upload`.  
    - Разбор загруженного файла, создание `transactions` и возможных `subscriptions`.
  - `internal/analytics`  
    - Чистые функции, которые делают агрегирующие SQL-запросы по `transactions` и возвращают данные для графиков.  
    - HTTP-слой (`http.go`) только разбирает query-параметры и зовёт эти функции.
  - `internal/forecast`  
    - Сервис, который по списку подписок считает месячную/годовую сумму и возможную экономию.  
    - Тоже разделён на сервис (`service.go`) и http-обёртку (`http.go`).
  - `internal/notifications`  
    - Таблицы `notification_settings` и `notifications`.  
    - CRUD для настроек + выдача списка уведомлений.  
    - Фоновая логика по генерации уведомлений (может быть отдельным файлом `worker.go`).
  - `internal/recommendations`  
    - Таблица `recommendation_alternatives` и логика подбора альтернатив по категории и цене.  
    - HTTP-роуты `/recommendations` и `/recommendations/subscription/:id`.
  - `internal/common`  
    - Общие типы и хелперы, например:
      - `response.go` — функции `JSONError`, базовые форматы ответов.
      - Общие ошибки, константы и т.п.

- **Поток данных:**
  1. HTTP-запрос приходит в Fiber роут (`/api/v1/...`).
  2. Через middleware получаем `userID` (если роут защищён).
  3. Хэндлер вызывает функции из доменного пакета (service/repository).
  4. Доменный код ходит в Supabase через `*sql.DB`.
  5. Результат возвращаем в JSON.

---

## 0.1. Этапы реализации (high-level план)

1. **Схема БД в Supabase** — создать все таблицы по `backend-mvp-spec.md`.
2. **Инфраструктура** — убедиться, что Go-сервер поднимается и ходит в Supabase.
3. **Пользователи и авторизация** — таблица `users`, пакет `users`, пакет `auth`, JWT, middleware.
4. **Подписки (CRUD)** — список/деталь/создание/обновление подписок.
5. **Источники и загрузка выписки** — `subscription_sources`, upload файла, создание `transactions`.
6. **Аналитика** — summary, категории, сервисы.
7. **Прогноз на год** — расчёт по подпискам.
8. **Уведомления** — настройки и сами уведомления.
9. **Рекомендации и быстрые действия (cancel/pause)**.

Ниже эти этапы расписаны более детально по файлам и шагам.

---

### 1. Поднять окружение и проверить, что сервер стартует

- **Шаги**:
  - Установи Go 1.22+.
  - В корне `backend/` выполни:
    - `go mod tidy`
    - `go run ./cmd/api`
  - Убедись, что:
    - Сервер стартует на `:8080` (или порте из `HTTP_PORT`).
    - `GET /api/v1/health` отдаёт `{ "status": "ok" }`.

- **Файлы**:
  - `backend/go.mod`
    - Здесь уже подключены зависимости `fiber`, `godotenv`, `pq` (драйвер Postgres). Обычно править не нужно, только иногда добавлять новые либы через `go get`.
  - `backend/cmd/api/main.go`
    - Точка входа. Загружает конфиг, коннектится к Supabase (Postgres), поднимает Fiber-сервер.  
    - Если что-то падает при старте — лог смотреть отсюда.
  - `backend/internal/config/config.go`
    - Читает `.env` (в том числе `DATABASE_URL` Supabase, `HTTP_PORT`).  
    - Если нужно добавить новые настройки (секрет для JWT и т.п.) — добавляешь поля в `Config` + читаешь через `os.Getenv`.
  - `backend/internal/server/server.go`
    - Создаёт `fiber.App`, вешает middleware, регистрирует роуты `/api/v1`.  
    - Сюда ты будешь добавлять зависимостями БД и конфиг в роуты (чтобы в хэндлерах был доступ к DB).
  - `backend/internal/server/routes/routes.go`
    - Реестр всех v1-роутов.  
    - Здесь подключаются модули: `auth`, позже добавишь `subscriptions`, `sources`, `analytics` и т.д.

---

### 2. Подключить Supabase (Postgres в облаке)

- **Что сделать**:
  - Создай проект в **Supabase**.
  - В Supabase в разделе `Project Settings` → `Database` найди строку подключения (Connection string) для `psql`/`Go`.
  - Скопируй **URI в формате Postgres**, он будет примерно такой:
    - `postgres://postgres:<PASSWORD>@db.<hash>.supabase.co:5432/postgres?sslmode=require`
  - В `.env` в корне `backend/` пропиши:
    - `HTTP_PORT=8080`
    - `DATABASE_URL=postgres://postgres:<PASSWORD>@db.<hash>.supabase.co:5432/postgres?sslmode=require`
  - В `config.go` можешь:
    - Убрать дефолтный локальный DSN.
    - Оставить только чтение `DATABASE_URL` и падать с ошибкой, если он пустой (чтобы не забыть настроить).

- **Файлы (что в них должно быть)**:
  - `backend/internal/config/config.go`
    - Функция `Load()` читает `DATABASE_URL` из env и пробрасывает его в `Config.DatabaseURL`.  
    - Здесь же можно позже добавить `JWT_SECRET`, `SUPABASE_SCHEMA` и т.п.
  - `backend/internal/db/db.go`
    - Функция `Connect(dsn string)` просто вызывает `sql.Open("postgres", dsn)` и `Ping()`.  
    - Никакой магии под Supabase не нужно — с точки зрения кода это обычный Postgres.

---

### 3. Добавить миграции и схемы таблиц

- **Вариант простой (для хакатона)**:
  - Написать SQL-скрипт с `CREATE TABLE ...` и прогнать его руками/через psql.

- **Вариант нормальный**:
  - Подключить что-то типа `golang-migrate`, но можно и без этого.

- **Что нужно создать** (см. `backend-mvp-spec.md`):
  - `users`
  - `subscription_sources`
  - `subscriptions`
  - `transactions`
  - `usage_events`
  - `notification_settings`
  - `notifications`
  - `recommendation_alternatives`

---

### 4. Реализовать auth (register/login/me)

- **Endpoints**:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `GET /api/v1/auth/me`

- **Шаги**:
  1. Создай модель пользователя и репозиторий:
     - Новый пакет, например `internal/users`:
       - `model.go` — структура `User` с полями, которые соответствуют таблице `users` в Supabase (id, email, password_hash, created_at).
       - `repository.go` — методы `CreateUser`, `GetUserByEmail`, `GetUserByID`, работающие через `*sql.DB`.
         - Внутри — обычные `INSERT/SELECT` с `db.QueryRowContext`/`db.ExecContext`.
  2. В `auth`-пакете:
     - Файл `internal/server/routes/v1/auth/auth.go` уже создан:
       - В `registerHandler`:
         - Парсишь JSON (`email`, `password`).
         - Хешируешь пароль через `bcrypt` (надо будет добавить зависимость в `go.mod`).
         - Вызываешь `users.CreateUser(...)`.
         - Возвращаешь 201 + базовую инфу о пользователе (без пароля).
       - В `loginHandler`:
         - Парсишь JSON.
         - Ищешь пользователя по email через `GetUserByEmail`.
         - Сравниваешь пароль с хешем (`bcrypt.CompareHashAndPassword`).
         - Если всё ок — генерируешь JWT и возвращаешь `{ token, user }`.
       - В `meHandler`:
         - Берёшь `userID` из контекста (см. middleware ниже).
         - Достаёшь пользователя через `GetUserByID` и возвращаешь его.
     - Генерация JWT:
       - Создай новый пакет `internal/auth/jwt.go`:
         - Функции: `GenerateToken(userID string) (string, error)` и `ParseToken(token string) (userID string, err error)`.
         - Секрет для подписи бери из `Config` (например, `JWT_SECRET` из `.env`).
  3. Сделать middleware для проверки JWT:
     - Новый файл в `internal/middleware` (например, `auth.go`).
     - Middleware, который достаёт токен из `Authorization: Bearer ...`, валидирует и кладёт `userID` в контекст.
  4. В `routes.RegisterV1Routes` повесить этот middleware на группы `/subscriptions`, `/analytics` и т.д.

- **Подсказка**:
  - Флоу максимально стандартный: Fiber + bcrypt + JWT — кода в интернете куча, можно адаптировать.

---

### 5. Реализовать CRUD для подписок (MVP)

- **Endpoints**:
  - `GET /api/v1/subscriptions`
  - `GET /api/v1/subscriptions/:id`
  - `POST /api/v1/subscriptions`
  - `PUT /api/v1/subscriptions/:id`
  - (опционально) `DELETE` или отдельные `cancel/pause`, см. следующие шаги.

- **Шаги**:
  1. Создай пакет `internal/subscriptions`:
     - `model.go` — структура `Subscription`, которая маппится на таблицу `subscriptions` в Supabase.
     - `repository.go` — методы:
       - `ListByUser(db *sql.DB, userID string)`
       - `GetByID(db *sql.DB, userID, id string)`
       - `Create(db *sql.DB, sub *Subscription)`
       - `Update(db *sql.DB, sub *Subscription)`
     - Здесь же можно сделать хелпер для маппинга `rows.Scan(...)` → `Subscription`.
  2. Создай `http.go`/`handlers.go` в `internal/subscriptions`:
     - Функция `RegisterRoutes(r fiber.Router, db *sql.DB)`.
     - Хэндлеры:
       - `listSubscriptions`
       - `getSubscription`
       - `createSubscription`
       - `updateSubscription`
  3. В `internal/server/routes/routes.go`:
     - Импортируй пакет `subscriptions` и повесь `subscriptions.RegisterRoutes(v1.Group("/subscriptions"), db)` (db нужно будет пробросить в роуты — можно изменить сигнатуру `RegisterV1Routes`).

- **Важно**:
  - Всегда фильтруй по `user_id`, который достаёшь из контекста (middleware с JWT).

---

### 6. Источники данных (банки/почта) и upload выписки

- **Endpoints**:
  - `GET /api/v1/sources`
  - `POST /api/v1/sources`
  - `POST /api/v1/sources/:id/mark-connected`
  - `POST /api/v1/sources/:id/upload`

- **Шаги**:
  1. Пакет `internal/sources`:
     - `model.go` — структура `SubscriptionSource` (id, user_id, type, provider, status, meta, created_at).
     - `repository.go` — CRUD-методы:
       - `ListByUser(db *sql.DB, userID string)`
       - `Create(db *sql.DB, src *SubscriptionSource)`
       - `UpdateStatus(db *sql.DB, id, userID, status string)`
     - `http.go` — `RegisterRoutes(r fiber.Router, db *sql.DB)` + хэндлеры:
       - `listSources`
       - `createSource`
       - `markConnected`
       - `uploadStatement`
  2. Для `upload`:
     - Используй `c.FormFile("file")` в Fiber.
     - Прочитай файл в память, распарси (для MVP можно сделать фиксированный CSV/JSON формат).
     - Для каждой транзакции:
       - Найди сервис по паттернам (просто `strings.Contains` по `raw_description`).
       - Создай/обнови `subscriptions` и `transactions`.

- **Подсказка**:
  - Можно жёстко захардкодить несколько паттернов:
    - `"Netflix"` → category `streaming`.
    - `"Spotify"` → `streaming`.
    - `"Yandex Plus"` → `streaming`.

---

### 7. Аналитика (summary, categories, services)

- **Endpoints**:
  - `GET /api/v1/analytics/summary`
  - `GET /api/v1/analytics/categories`
  - `GET /api/v1/analytics/services`
  - (опционально) `GET /api/v1/analytics/subscription/:id`

- **Шаги**:
  1. Пакет `internal/analytics`:
     - `service.go` — функции, которые принимают `userID`, период (`from`, `to`) и делают SQL агрегации по `transactions`:
       - `GetSummary(db *sql.DB, userID string, from, to time.Time)`
       - `GetByCategories(...)`
       - `GetByServices(...)`
  2. HTTP-слой:
     - Можно сделать `http.go` в том же пакете `internal/analytics`:
       - `RegisterRoutes(r fiber.Router, db *sql.DB)`.
       - Хэндлеры:
         - `getSummary`
         - `getCategories`
         - `getServices`
     - В хэндлерах:
     - Разобрать query-параметры (period/from/to).
       - Вызвать соответствующую функцию из `service.go` и вернуть JSON.

- **Подсказка**:
  - Для MVP можно ограничиться текущим месяцем/годом:
    - `from` = начало месяца.
    - `to` = конец месяца.

---

### 8. Прогноз на год

- **Endpoint**:
  - `GET /api/v1/forecast/year`

- **Шаги**:
  1. Пакет `internal/forecast`:
     - `service.go`:
       - Функция `BuildYearForecast(db *sql.DB, userID string)`:
         - Получает все активные подписки (`status = 'active'`) пользователя из Supabase.
         - Для каждой считает `monthly_cost` в Go (нормализует годовые подписки).
         - Складывает и умножает на 12 → `total_year_cost`.
         - Опционально: помечает подписки с низким использованием и считает возможную экономию.
  2. HTTP-хэндлер:
     - В `http.go` пакета `forecast`:
     - Получает `userID` из контекста.
     - Вызвает `BuildYearForecast`.
     - Возвращает JSON с полями `total_year_cost`, `baseline_monthly_cost`, `economy_if_cancel_low_usage`.

---

### 9. Уведомления и настройки

- **Endpoints**:
  - `GET /api/v1/notifications/settings`
  - `PUT /api/v1/notifications/settings`
  - `GET /api/v1/notifications`
  - `PUT /api/v1/notifications/:id/read`

- **Background-логика**:
  - В `cmd/api/main.go` после старта сервера можно запустить горутину:
    - Таймер раз в N минут.
    - Проверяет:
      - Подписки с `next_charge_at` близко к текущей дате.
      - Подписки с мало `usage_events`.
    - Создаёт записи в `notifications`.

- **Подсказка**:
  - Для MVP не делай реальных пушей/писем — достаточно, что фронт забирает их через `GET /notifications` и показывает.

- **По файлам**:
  - `internal/notifications/model.go` — структуры `Notification` и `NotificationSettings`, соответствующие таблицам в Supabase.
  - `internal/notifications/repository.go` — функции для чтения/записи уведомлений и настроек.
  - `internal/notifications/http.go` — `RegisterRoutes(r fiber.Router, db *sql.DB)` + хэндлеры для всех четырёх эндпоинтов.
  - `cmd/api/main.go` — запуск фоновой горутины для генерации уведомлений (перед `srv.Listen`).

---

### 10. Рекомендации альтернатив

- **Endpoints**:
  - `GET /api/v1/recommendations`
  - `GET /api/v1/recommendations/subscription/:id`

- **Шаги**:
  1. Заполни `recommendation_alternatives` руками (несколько сервисов по категориям).
  2. Функции:
     - По категории + лимиту цены вернуть список альтернатив с расчётом потенциальной экономии.

- **По файлам**:
  - `internal/recommendations/model.go` — структура `RecommendationAlternative`.
  - `internal/recommendations/repository.go` — выборка альтернатив по категории и цене.
  - `internal/recommendations/http.go` — `RegisterRoutes(r fiber.Router, db *sql.DB)` + хэндлеры для `/recommendations` и `/recommendations/subscription/:id`.

---

### 11. Быстрая отмена / пауза

- **Endpoints**:
  - `POST /api/v1/subscriptions/:id/cancel-intent`
  - `POST /api/v1/subscriptions/:id/confirm-cancel`
  - `POST /api/v1/subscriptions/:id/pause`

- **Шаги**:
  1. Добавь поля `cancel_url`, `support_email` в таблицу `subscriptions` (или meta json).
  2. В `cancel-intent`:
     - Меняй статус на `pending_cancel`.
     - Возвращай `cancel_url` и сгенерированный `email_template`.
  3. В `confirm-cancel`:
     - Меняй статус на `cancelled`.
  4. В `pause`:
     - Меняй статус на `paused` + сохраняй период паузы (можно в отдельной таблице или meta).

- **По файлам**:
  - `internal/subscriptions/model.go` — добавить поля `CancelURL`, `SupportEmail`, `Meta` (если нужно).
  - `internal/subscriptions/repository.go` — методы:
    - `SetCancelIntent(...)` — обновляет статус и meta.
    - `ConfirmCancel(...)`.
    - `Pause(...)`.
  - `internal/subscriptions/http.go` — хэндлеры:
    - `cancelIntentHandler`
    - `confirmCancelHandler`
    - `pauseHandler`

---

### 12. Полировка и подготовка к демо

- **Что проверить**:
  - Все основные эндпоинты из `backend-mvp-spec.md` отвечают.
  - На некорректные запросы возвращаются нормальные JSON-ошибки (через `common.JSONError`).
  - Есть минимальные логи (`log.Printf`) для ключевых операций.
  - Health-check (`/api/v1/health`) зелёный.

