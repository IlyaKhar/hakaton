package sources

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
	"github.com/hakaton/subscriptions-backend/internal/config"
)

// RegisterRoutes регистрирует HTTP-роуты для работы с источниками.
func RegisterRoutes(r fiber.Router, db *sql.DB, cfg *config.Config) {
	repo := NewRepository(db)
	parserSvc := NewParserService(db)
	oauthSvc := NewOAuthService(cfg, repo)

	h := &handler{
		repo:      repo,
		parserSvc: parserSvc,
		oauthSvc:  oauthSvc,
		cfg:       cfg,
	}

	r.Get("/", h.listSources)
	r.Get("/providers", h.listProviders)
	r.Post("/", h.createSource)
	r.Get("/oauth/:provider/authorize", h.oauthAuthorize)
	r.Post("/:id/mark-connected", h.markConnected)
	r.Post("/:id/upload", h.uploadStatement)
}

// RegisterOAuthCallbackRoutes регистрирует публичный callback OAuth.
// Важно: callback дергает провайдер без JWT, поэтому он должен быть БЕЗ AuthMiddleware.
func RegisterOAuthCallbackRoutes(r fiber.Router, db *sql.DB, cfg *config.Config) {
	repo := NewRepository(db)
	oauthSvc := NewOAuthService(cfg, repo)

	h := &handler{
		repo:     repo,
		oauthSvc: oauthSvc,
		cfg:      cfg,
	}

	// Внимание: этот роут должен совпадать с Redirect URI в настройках OAuth провайдера.
	// Полный путь в API будет: /api/v1/sources/oauth/:provider/callback
	r.Get("/:provider/callback", h.oauthCallback)
}

type handler struct {
	repo      *Repository
	parserSvc *ParserService
	oauthSvc  *OAuthService
	cfg       *config.Config
}

// listSources — хэндлер, который берёт userId из контекста.
// listSources godoc
// @Summary      Список источников
// @Description  Возвращает все подключённые источники пользователя (банки, почта и т.п.)
// @Tags         sources
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   SubscriptionSource
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /sources [get]
func (h *handler) listSources(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	sourcesList, err := h.repo.ListByUser(c.Context(), userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(sourcesList)
}

// createSourceRequest описывает тело запроса для создания источника.
type createSourceRequest struct {
	// Тип источника: "email", "bank" или "manual"
	Type string `json:"type" example:"email"`
	// Провайдер: "mailru", "yandex", "gmail" для email; "sberbank", "tinkoff" и т.д. для bank
	Provider string `json:"provider" example:"mailru"`
	// Метаданные источника. Для email обязательно указать meta.email
	Meta map[string]any `json:"meta"`
}

// createSource создаёт новый источник (банк/почта/ручной).
// createSource godoc
// @Summary      Создать источник
// @Description  Создаёт новый источник данных подписок. Для почты обязательно указать email в meta.email
// @Tags         sources
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      createSourceRequest  true  "Параметры источника"
// @Success      201      {object}  SubscriptionSource
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /sources [post]
func (h *handler) createSource(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	var payload createSourceRequest
	if err := c.BodyParser(&payload); err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось прочитать тело запроса")
	}

	if payload.Type == "" || payload.Provider == "" {
		return common.JSONError(c, http.StatusBadRequest, "type и provider обязательны")
	}

	// Валидация типа
	validTypes := []string{"bank", "email", "manual"}
	typeValid := false
	for _, vt := range validTypes {
		if payload.Type == vt {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return common.JSONError(c, http.StatusBadRequest, "type должен быть: bank, email или manual")
	}

	// Для email проверяем наличие email в meta
	if payload.Type == "email" {
		if payload.Meta == nil {
			payload.Meta = make(map[string]any)
		}
		if email, ok := payload.Meta["email"].(string); !ok || email == "" {
			return common.JSONError(c, http.StatusBadRequest, "для типа email необходимо указать meta.email")
		}
	}

	// Если meta не указан, создаём пустой
	if payload.Meta == nil {
		payload.Meta = make(map[string]any)
	}

	src := &SubscriptionSource{
		UserID:   userID,
		Type:     payload.Type,
		Provider: payload.Provider,
		Status:   "pending",
		Meta:     payload.Meta,
	}

	created, err := h.repo.Create(c.Context(), src)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	log.Printf("source created: id=%s, user_id=%s, type=%s, provider=%s", created.ID, created.UserID, created.Type, created.Provider)

	return c.Status(http.StatusCreated).JSON(created)
}

// markConnected имитирует успешное подключение источника (OAuth прошёл).
// markConnected godoc
// @Summary      Пометить источник как подключённый
// @Description  Обновляет статус источника на connected
// @Tags         sources
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "ID источника"
// @Success      204  "успешно"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /sources/{id}/mark-connected [post]
func (h *handler) markConnected(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	if err := h.repo.UpdateStatus(c.Context(), userID, id, "connected"); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}

// uploadStatement принимает файл выписки/чеков и парсит его.
// uploadStatement godoc
// @Summary      Загрузить выписку/чеки
// @Description  Принимает файл выписки (текст, CSV), парсит и создаёт транзакции
// @Tags         sources
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id    path   string true  "ID источника"
// @Param        file  formData file  true  "Файл выписки"
// @Success      200   {object}  map[string]any "transactionsCreated: количество созданных транзакций"
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /sources/{id}/upload [post]
func (h *handler) uploadStatement(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	// Проверяем, что источник существует и принадлежит пользователю
	source, err := h.repo.GetByID(c.Context(), userID, id)
	if err != nil {
		return common.JSONError(c, http.StatusNotFound, "источник не найден")
	}
	_ = source // используем для проверки

	file, err := c.FormFile("file")
	if err != nil {
		return common.JSONError(c, http.StatusBadRequest, "нужно передать файл в поле file")
	}

	// Открываем файл
	src, err := file.Open()
	if err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось открыть файл")
	}
	defer src.Close()

	// Читаем содержимое
	content, err := io.ReadAll(src)
	if err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось прочитать файл")
	}

	// Парсим файл
	parsed, err := h.parserSvc.ParseTextFile(c.Context(), string(content), userID, id)
	if err != nil {
		log.Printf("error parsing file: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при парсинге файла: "+err.Error())
	}

	// Обрабатываем транзакции (создаём подписки и транзакции)
	if err := h.parserSvc.ProcessParsedTransactions(c.Context(), parsed, userID, id); err != nil {
		log.Printf("error processing transactions: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при обработке транзакций")
	}

	// Помечаем источник как connected
	if err := h.repo.UpdateStatus(c.Context(), userID, id, "connected"); err != nil {
		log.Printf("warning: failed to update source status: %v", err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":             "файл успешно обработан",
		"transactionsCreated": len(parsed),
	})
}

// listProviders возвращает список доступных провайдеров для подключения.
// listProviders godoc
// @Summary      Список провайдеров
// @Description  Возвращает список доступных провайдеров (банки, почта) для подключения
// @Tags         sources
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]any
// @Router       /sources/providers [get]
func (h *handler) listProviders(c *fiber.Ctx) error {
	providers := fiber.Map{
		"banks": []fiber.Map{
			{"id": "sberbank", "name": "Сбербанк", "type": "bank", "icon": "sberbank"},
			{"id": "tinkoff", "name": "Тинькофф", "type": "bank", "icon": "tinkoff"},
			{"id": "yoomoney", "name": "ЮMoney", "type": "bank", "icon": "yoomoney"},
			{"id": "alfabank", "name": "Альфа-Банк", "type": "bank", "icon": "alfabank"},
		},
		"email": []fiber.Map{
			{"id": "mailru", "name": "Mail.ru", "type": "email", "icon": "mailru"},
			{"id": "yandex", "name": "Яндекс.Почта", "type": "email", "icon": "yandex"},
			{"id": "gmail", "name": "Gmail", "type": "email", "icon": "gmail"},
		},
	}

	return c.JSON(providers)
}

// oauthAuthorize возвращает URL для редиректа на страницу авторизации провайдера.
// oauthAuthorize godoc
// @Summary      Получить URL для OAuth авторизации
// @Description  Возвращает URL для редиректа пользователя на страницу авторизации провайдера
// @Tags         sources
// @Produce      json
// @Security     BearerAuth
// @Param        provider  path  string  true  "Провайдер: mailru, yandex, gmail"
// @Success      200  {object}  map[string]string "authorizationUrl"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /sources/oauth/{provider}/authorize [get]
func (h *handler) oauthAuthorize(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return common.JSONError(c, http.StatusBadRequest, "provider is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	// Формируем redirect URI (callback URL)
	redirectURI := h.cfg.BaseUrl + "/api/v1/sources/oauth/" + provider + "/callback"

	authURL, err := h.oauthSvc.GetAuthorizationURL(provider, userID, redirectURI)
	if err != nil {
		log.Printf("error generating auth URL: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при генерации URL авторизации: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"authorizationUrl": authURL,
	})
}

// oauthCallback обрабатывает callback от провайдера после авторизации.
// oauthCallback godoc
// @Summary      OAuth callback
// @Description  Обрабатывает callback от провайдера, обменивает code на token и создаёт источник
// @Tags         sources
// @Produce      json
// @Param        provider  path  string  true  "Провайдер: mailru, yandex, gmail"
// @Param        code      query string  true  "Authorization code от провайдера"
// @Param        state     query string  false "State для проверки"
// @Success      200  {object}  SubscriptionSource
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /sources/oauth/{provider}/callback [get]
func (h *handler) oauthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return common.JSONError(c, http.StatusBadRequest, "provider is required")
	}

	code := c.Query("code")
	if code == "" {
		return common.JSONError(c, http.StatusBadRequest, "code is required")
	}

	state := c.Query("state")
	if state == "" {
		return common.JSONError(c, http.StatusBadRequest, "state is required")
	}

	// Извлекаем userID из state
	userID, err := h.oauthSvc.ParseState(state)
	if err != nil {
		log.Printf("error parsing state: %v", err)
		return common.JSONError(c, http.StatusBadRequest, "неверный state")
	}

	// Формируем redirect URI (должен совпадать с тем, что был в authorize)
	redirectURI := h.cfg.BaseUrl + "/api/v1/sources/oauth/" + provider + "/callback"

	// Обмениваем code на access token
	tokenData, err := h.oauthSvc.ExchangeCodeForToken(provider, code, redirectURI)
	if err != nil {
		log.Printf("error exchanging code for token: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении токена: "+err.Error())
	}

	// Извлекаем access token
	accessToken, ok := tokenData["access_token"].(string)
	if !ok {
		return common.JSONError(c, http.StatusInternalServerError, "access_token not found in response")
	}

	// Получаем email пользователя
	email, err := h.oauthSvc.GetUserEmail(provider, accessToken)
	if err != nil {
		log.Printf("error getting user email: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении email: "+err.Error())
	}

	// Формируем meta с токенами
	meta := map[string]interface{}{
		"email":       email,
		"accessToken": accessToken,
	}

	// Добавляем refresh_token если есть
	if refreshToken, ok := tokenData["refresh_token"].(string); ok && refreshToken != "" {
		meta["refreshToken"] = refreshToken
	}

	// Добавляем expires_in если есть
	if expiresIn, ok := tokenData["expires_in"].(float64); ok {
		meta["expiresAt"] = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}

	// Создаём источник
	src := &SubscriptionSource{
		UserID:   userID,
		Type:     "email",
		Provider: provider,
		Status:   "connected",
		Meta:     meta,
	}

	created, err := h.repo.Create(c.Context(), src)
	if err != nil {
		log.Printf("error creating source: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при создании источника: "+err.Error())
	}

	// Редиректим на фронт с успешным сообщением
	// Для MVP можно вернуть JSON, фронт сам обработает
	return c.JSON(fiber.Map{
		"message":  "Почта успешно подключена",
		"source":   created,
		"redirect": h.cfg.BaseUrl + "/sources?connected=" + created.ID, // фронт сам обработает
	})

	// Когда userID будет получен:
	// meta := map[string]interface{}{
	// 	"email":        email,
	// 	"accessToken":  accessToken,
	// 	"refreshToken": tokenData["refresh_token"], // если есть
	// 	"expiresAt":    tokenData["expires_in"],    // если есть
	// }
	//
	// src := &SubscriptionSource{
	// 	UserID:   userID,
	// 	Type:     "email",
	// 	Provider: provider,
	// 	Status:   "connected",
	// 	Meta:     meta,
	// }
	//
	// created, err := h.repo.Create(c.Context(), src)
	// if err != nil {
	// 	return common.JSONError(c, http.StatusInternalServerError, err.Error())
	// }
	//
	// return c.JSON(created)
}
