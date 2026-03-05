package analytics

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
)

// RegisterRoutes регистрирует HTTP-роуты аналитики.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	svc := NewService(db)
	h := &handler{svc: svc}

	r.Get("/summary", h.getSummary)
	r.Get("/categories", h.getCategories)
	r.Get("/services", h.getServices)
}

type handler struct {
	svc *Service
}

// getSummary godoc
// @Summary      Сводная аналитика
// @Description  Возвращает общие суммы и метрики по подпискам за период
// @Tags         analytics
// @Produce      json
// @Param        period  query    string  false  "Период (month|year)"  default(month)
// @Success      200     {object}  SummaryResult
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /analytics/summary [get]
func (h *handler) getSummary(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	period := c.Query("period", "month")

	now := time.Now()
	var from time.Time

	switch period {
	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	to := now

	summary, err := h.svc.GetSummary(c.Context(), userID, from, to)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(summary)
}

// getCategories — суммы по категориям за период.
// getCategories godoc
// @Summary      Аналитика по категориям
// @Description  Возвращает суммы по категориям за выбранный период
// @Tags         analytics
// @Produce      json
// @Param        period  query    string  false  "Период (month|year)"  default(month)
// @Success      200     {array}   CategoryStat
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /analytics/categories [get]
func (h *handler) getCategories(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	period := c.Query("period", "month")

	now := time.Now()
	var from time.Time

	switch period {
	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	to := now

	stats, err := h.svc.GetByCategories(c.Context(), userID, from, to)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(stats)
}

// getServices — суммы по сервисам за период.
// getServices godoc
// @Summary      Аналитика по сервисам
// @Description  Возвращает суммы по конкретным сервисам за выбранный период
// @Tags         analytics
// @Produce      json
// @Param        period  query    string  false  "Период (month|year)"  default(month)
// @Success      200     {array}   ServiceStat
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /analytics/services [get]
func (h *handler) getServices(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	period := c.Query("period", "month")

	now := time.Now()
	var from time.Time

	switch period {
	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	to := now

	stats, err := h.svc.GetByServices(c.Context(), userID, from, to)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(stats)
}
