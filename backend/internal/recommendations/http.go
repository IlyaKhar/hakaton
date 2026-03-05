package recommendations

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
	"github.com/hakaton/subscriptions-backend/internal/subscriptions"
)

// RegisterRoutes регистрирует HTTP-роуты для рекомендаций.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	repo := NewRepository(db)
	subsRepo := subscriptions.NewRepository(db)
	h := &handler{repo: repo, subsRepo: subsRepo}

	r.Get("/", h.listRecommendations)
}

type handler struct {
	repo     *Repository
	subsRepo *subscriptions.Repository
}

// listRecommendations — простой хэндлер, который принимает category и maxPrice.
// listRecommendations godoc
// @Summary      Рекомендации по категории
// @Description  Возвращает альтернативные сервисы в категории дешевле указанной цены
// @Tags         recommendations
// @Produce      json
// @Param        category  query    string  true   "Категория подписки"
// @Param        maxPrice  query    number  false  "Максимальная цена"  default(1000000)
// @Success      200       {array}  RecommendationAlternative
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /recommendations [get]
func (h *handler) listRecommendations(c *fiber.Ctx) error {
	category := c.Query("category")
	if category == "" {
		return common.JSONError(c, http.StatusBadRequest, "category is required")
	}

	maxPriceStr := c.Query("maxPrice", "1000000")
	maxPrice, err := strconv.ParseFloat(maxPriceStr, 64)
	if err != nil {
		return common.JSONError(c, http.StatusBadRequest, "invalid maxPrice")
	}

	result, err := h.repo.ListByCategory(c.Context(), category, maxPrice)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(result)
}

// listForSubscription — рекомендации для конкретной подписки.
// listForSubscription godoc
// @Summary      Рекомендации для подписки
// @Description  Возвращает альтернативные сервисы для конкретной подписки пользователя
// @Tags         recommendations
// @Produce      json
// @Param        id   path  string  true  "ID подписки"
// @Success      200  {array}  RecommendationAlternative
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /recommendations/{id} [get]
func (h *handler) listForSubscription(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	sub, err := h.subsRepo.GetByID(c.Context(), userID, id)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	// Ищем альтернативы в той же категории, но не дороже текущей подписки
	result, err := h.repo.ListByCategory(c.Context(), sub.Category, sub.Price)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(result)
}
