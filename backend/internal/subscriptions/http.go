package subscriptions

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
)

// RegisterRoutes регистрирует HTTP-роуты для работы с подписками.
// TODO: вызывать эту функцию из общего реестра роутов, когда будет готов auth-middleware.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	repo := NewRepository(db)

	h := &handler{
		repo: repo,
	}

	r.Get("/", h.listSubscriptions)
	r.Get("/:id", h.getSubscription)
	r.Post("/", h.createSubscription)
	r.Put("/:id", h.updateSubscription)
	r.Post("/:id/cancel-intent", h.cancelIntent)
	r.Post("/:id/confirm-cancel", h.confirmCancel)
	r.Post("/:id/pause", h.pause)
}

// handler хранит зависимости для хэндлеров подписок.
type handler struct {
	repo *Repository
}

// listSubscriptions godoc
// @Summary      Список подписок пользователя
// @Description  Возвращает все подписки текущего пользователя
// @Tags         subscriptions
// @Produce      json
// @Success      200  {array}   Subscription
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /subscriptions [get]
func (h *handler) listSubscriptions(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	subs, err := h.repo.ListByUser(c.Context(), userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(subs)
}

// getSubscription — хэндлер деталей подписки.
// getSubscription godoc
// @Summary      Детали подписки
// @Description  Возвращает одну подписку по ID
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "ID подписки"
// @Success      200  {object}  Subscription
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /subscriptions/{id} [get]
func (h *handler) getSubscription(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	sub, err := h.repo.GetByID(c.Context(), userID, id)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(sub)
}

// createSubscriptionRequest описывает тело запроса для создания подписки.
type createSubscriptionRequest struct {
	ServiceName   string     `json:"serviceName"`
	Category      string     `json:"category"`
	Price         float64    `json:"price"`
	Currency      string     `json:"currency"`
	BillingPeriod string     `json:"billingPeriod"`
	NextChargeAt  *time.Time `json:"nextChargeAt"`
}

// createSubscription создаёт новую подписку.
// createSubscription godoc
// @Summary      Создать подписку
// @Description  Создаёт новую подписку для текущего пользователя
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        payload  body      createSubscriptionRequest  true  "Параметры подписки"
// @Success      201      {object}  Subscription
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /subscriptions [post]
func (h *handler) createSubscription(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	var payload struct {
		ServiceName   string     `json:"serviceName"`
		Category      string     `json:"category"`
		Price         float64    `json:"price"`
		Currency      string     `json:"currency"`
		BillingPeriod string     `json:"billingPeriod"`
		NextChargeAt  *time.Time `json:"nextChargeAt"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось прочитать тело запроса")
	}

	if payload.ServiceName == "" || payload.Category == "" || payload.Price <= 0 || payload.BillingPeriod == "" {
		return common.JSONError(c, http.StatusBadRequest, "serviceName, category, price и billingPeriod обязательны")
	}

	sub := &Subscription{
		UserID:        userID,
		ServiceName:   payload.ServiceName,
		Category:      payload.Category,
		Price:         payload.Price,
		Currency:      payload.Currency,
		BillingPeriod: payload.BillingPeriod,
		NextChargeAt:  payload.NextChargeAt,
		Status:        "active",
	}

	created, err := h.repo.Create(c.Context(), sub)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusCreated).JSON(created)
}

// updateSubscriptionRequest описывает тело запроса для обновления подписки.
type updateSubscriptionRequest struct {
	ServiceName   string     `json:"serviceName"`
	Category      string     `json:"category"`
	Price         float64    `json:"price"`
	Currency      string     `json:"currency"`
	BillingPeriod string     `json:"billingPeriod"`
	NextChargeAt  *time.Time `json:"nextChargeAt"`
	Status        string     `json:"status"`
	CancelURL     *string    `json:"cancelUrl"`
	SupportEmail  *string    `json:"supportEmail"`
}

// updateSubscription обновляет существующую подписку.
// updateSubscription godoc
// @Summary      Обновить подписку
// @Description  Обновляет параметры существующей подписки
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "ID подписки"
// @Param        payload  body      updateSubscriptionRequest true  "Новые параметры подписки"
// @Success      200      {object}  Subscription
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /subscriptions/{id} [put]
func (h *handler) updateSubscription(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	var payload struct {
		ServiceName   string     `json:"serviceName"`
		Category      string     `json:"category"`
		Price         float64    `json:"price"`
		Currency      string     `json:"currency"`
		BillingPeriod string     `json:"billingPeriod"`
		NextChargeAt  *time.Time `json:"nextChargeAt"`
		Status        string     `json:"status"`
		CancelURL     *string    `json:"cancelUrl"`
		SupportEmail  *string    `json:"supportEmail"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось прочитать тело запроса")
	}

	sub := &Subscription{
		ID:            id,
		UserID:        userID,
		ServiceName:   payload.ServiceName,
		Category:      payload.Category,
		Price:         payload.Price,
		Currency:      payload.Currency,
		BillingPeriod: payload.BillingPeriod,
		NextChargeAt:  payload.NextChargeAt,
		Status:        payload.Status,
		CancelURL:     payload.CancelURL,
		SupportEmail:  payload.SupportEmail,
	}

	updated, err := h.repo.Update(c.Context(), sub)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(updated)
}

// cancelIntent помечает подписку как "в процессе отмены".
// cancelIntent godoc
// @Summary      Пометить подписку как в процессе отмены
// @Description  Обновляет статус подписки на pending_cancel
// @Tags         subscriptions
// @Produce      json
// @Param        id   path  string  true  "ID подписки"
// @Success      204  "успешно"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /subscriptions/{id}/cancel-intent [post]
func (h *handler) cancelIntent(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	if err := h.repo.SetStatus(c.Context(), userID, id, "pending_cancel"); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}

// confirmCancel окончательно помечает подписку как отменённую.
// confirmCancel godoc
// @Summary      Отменить подписку
// @Description  Обновляет статус подписки на cancelled
// @Tags         subscriptions
// @Produce      json
// @Param        id   path  string  true  "ID подписки"
// @Success      204  "успешно"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /subscriptions/{id}/confirm-cancel [post]
func (h *handler) confirmCancel(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	if err := h.repo.SetStatus(c.Context(), userID, id, "cancelled"); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}

// pause ставит подписку на паузу (только для аналитики в MVP).
// pause godoc
// @Summary      Поставить подписку на паузу
// @Description  Обновляет статус подписки на paused
// @Tags         subscriptions
// @Produce      json
// @Param        id   path  string  true  "ID подписки"
// @Success      204  "успешно"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /subscriptions/{id}/pause [post]
func (h *handler) pause(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	if err := h.repo.SetStatus(c.Context(), userID, id, "paused"); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}
