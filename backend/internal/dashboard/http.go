package dashboard

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/analytics"
	"github.com/hakaton/subscriptions-backend/internal/common"
	"github.com/hakaton/subscriptions-backend/internal/payment_cards"
	"github.com/hakaton/subscriptions-backend/internal/subscriptions"
)

// RegisterRoutes регистрирует HTTP-роуты дашборда.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	subsRepo := subscriptions.NewRepository(db)
	cardsRepo := payment_cards.NewRepository(db)
	analyticsSvc := analytics.NewService(db)

	h := &handler{
		subsRepo:  subsRepo,
		cardsRepo: cardsRepo,
		analytics: analyticsSvc,
	}

	r.Get("/", h.getSummary)
}

type handler struct {
	subsRepo  *subscriptions.Repository
	cardsRepo *payment_cards.Repository
	analytics *analytics.Service
}

// SummaryResponse описывает данные для главного экрана.
type SummaryResponse struct {
	TotalSubscriptions    int `json:"totalSubscriptions"`
	ActiveSubscriptions   int `json:"activeSubscriptions"`
	PausedSubscriptions   int `json:"pausedSubscriptions"`
	CanceledSubscriptions int `json:"canceledSubscriptions"`

	TotalCards       int     `json:"totalCards"`
	DefaultCardLast4 *string `json:"defaultCardLast4,omitempty"`

	MonthSpend    float64                  `json:"monthSpend"`
	Currency      string                   `json:"currency"`
	TopCategories []analytics.CategoryStat `json:"topCategories"`
	TopServices   []analytics.ServiceStat  `json:"topServices"`
}

// getSummary godoc
// @Summary      Дашборд главного экрана
// @Description  Возвращает агрегированные данные для главного экрана: подписки, карты, траты за месяц
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  SummaryResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /dashboard [get]
func (h *handler) getSummary(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	ctx := c.Context()

	// Подписки
	subs, err := h.subsRepo.ListByUser(ctx, userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении подписок")
	}

	var totalSubs, active, paused, canceled int
	for _, s := range subs {
		totalSubs++
		switch s.Status {
		case "active":
			active++
		case "paused":
			paused++
		case "canceled":
			canceled++
		}
	}

	// Карты
	cards, err := h.cardsRepo.ListByUser(ctx, userID)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении карт")
	}

	var totalCards int
	var defaultCardLast4 *string
	for _, card := range cards {
		totalCards++
		if card.IsDefault && defaultCardLast4 == nil {
			lc := card.LastFourDigits
			defaultCardLast4 = &lc
		}
	}

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := now

	summary, err := h.analytics.GetSummary(ctx, userID, from, to)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении аналитики")
	}

	categories, err := h.analytics.GetByCategories(ctx, userID, from, to)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении аналитики по категориям")
	}

	services, err := h.analytics.GetByServices(ctx, userID, from, to)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении аналитики по сервисам")
	}

	resp := SummaryResponse{
		TotalSubscriptions:    totalSubs,
		ActiveSubscriptions:   active,
		PausedSubscriptions:   paused,
		CanceledSubscriptions: canceled,
		TotalCards:            totalCards,
		DefaultCardLast4:      defaultCardLast4,
		MonthSpend:            summary.TotalAmount,
		// В MVP считаем, что все суммы в RUB.
		Currency:      "RUB",
		TopCategories: categories,
		TopServices:   services,
	}

	return c.JSON(resp)
}
