package forecast

import (
	"database/sql"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
)

// RegisterRoutes регистрирует HTTP-роуты прогноза.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	svc := NewService(db)
	h := &handler{svc: svc}

	r.Get("/year", h.getYearForecast)
}

type handler struct {
	svc *Service
}

// getYearForecast godoc
// @Summary      Годовой прогноз трат
// @Description  Строит прогноз трат на подписки на год вперёд
// @Tags         forecast
// @Produce      json
// @Success      200  {object}  YearForecast
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /forecast/year [get]
func (h *handler) getYearForecast(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	result, err := h.svc.BuildYearForecast(c.Context(), userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(result)
}

