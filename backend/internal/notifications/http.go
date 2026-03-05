package notifications

import (
	"database/sql"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
)

// RegisterRoutes регистрирует HTTP-роуты для уведомлений.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	repo := NewRepository(db)
	h := &handler{repo: repo}

	r.Get("/settings", h.getSettings)
	r.Put("/settings", h.updateSettings)
	r.Get("/", h.listNotifications)
	r.Put("/:id/read", h.markRead)
}

type handler struct {
	repo *Repository
}

// Предопределённые типы уведомлений для нашего приложения.
const (
	// Напоминания о списаниях
	NotificationTypeBilling3Days  = "billing_3_days"
	NotificationTypeBilling1Day   = "billing_1_day"
	NotificationTypeTrialEnding   = "trial_ending"

	// Контроль бюджета
	NotificationTypeBudgetControl = "budget_control"
)

// getSettings godoc
// @Summary      Получить настройки уведомлений
// @Description  Возвращает текущие настройки уведомлений пользователя
// @Tags         notifications
// @Produce      json
// @Success      200  {array}   NotificationSetting
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /notifications/settings [get]
func (h *handler) getSettings(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	settings, err := h.repo.ListSettings(c.Context(), userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	// Если настроек ещё нет, возвращаем дефолтный набор под дизайн.
	if len(settings) == 0 {
		settings = []NotificationSetting{
			{
				UserID:  userID,
				Type:    NotificationTypeBilling3Days,
				Enabled: false,
				Channels: map[string]any{
					"push":  true,
					"email": false,
				},
			},
			{
				UserID:  userID,
				Type:    NotificationTypeBilling1Day,
				Enabled: true,
				Channels: map[string]any{
					"push":  true,
					"email": false,
				},
			},
			{
				UserID:  userID,
				Type:    NotificationTypeTrialEnding,
				Enabled: true,
				Channels: map[string]any{
					"push":  true,
					"email": true,
				},
			},
			{
				// Контроль бюджета: месячный лимит + пороги 80%/100%
				UserID:  userID,
				Type:    NotificationTypeBudgetControl,
				Enabled: false,
				Channels: map[string]any{
					"limit":            3000.0,
					"currency":         "RUB",
					"notifyOn80":       true,
					"notifyOnExceeded": true,
					"push":             true,
					"email":            false,
				},
			},
		}
	}

	return c.JSON(settings)
}

// updateSettings обновляет настройки уведомлений.
// updateSettings godoc
// @Summary      Обновить настройки уведомлений
// @Description  Сохраняет список настроек уведомлений для пользователя
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        payload  body      []NotificationSetting  true  "Настройки уведомлений"
// @Success      204      "успешно"
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /notifications/settings [put]
func (h *handler) updateSettings(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	var payload []NotificationSetting
	if err := c.BodyParser(&payload); err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось прочитать тело запроса")
	}

	if err := h.repo.UpsertSettings(c.Context(), userID, payload); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}

// listNotifications возвращает список уведомлений пользователя.
// listNotifications godoc
// @Summary      Список уведомлений
// @Description  Возвращает список уведомлений пользователя
// @Tags         notifications
// @Produce      json
// @Success      200  {array}   Notification
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /notifications [get]
func (h *handler) listNotifications(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	items, err := h.repo.ListNotifications(c.Context(), userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(items)
}

// markRead помечает конкретное уведомление как прочитанное.
// markRead godoc
// @Summary      Пометить уведомление прочитанным
// @Description  Отмечает выбранное уведомление как прочитанное
// @Tags         notifications
// @Produce      json
// @Param        id   path  string  true  "ID уведомления"
// @Success      204  "успешно"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /notifications/{id}/read [put]
func (h *handler) markRead(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	if err := h.repo.MarkRead(c.Context(), userID, id); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}
