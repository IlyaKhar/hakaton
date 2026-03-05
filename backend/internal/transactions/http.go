package transactions

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
)

// RegisterRoutes регистрирует HTTP-роуты для работы с транзакциями.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	repo := NewRepository(db)

	h := &handler{
		repo: repo,
	}

	r.Get("/", h.listTransactions)
	r.Get("/:id", h.getTransaction)
}

type handler struct {
	repo *Repository
}

// listTransactions godoc
// @Summary      Список транзакций
// @Description  Возвращает список транзакций пользователя за период
// @Tags         transactions
// @Produce      json
// @Param        from  query     string  false  "Дата начала периода (YYYY-MM-DD)"
// @Param        to    query     string  false  "Дата конца периода (YYYY-MM-DD)"
// @Success      200   {array}   Transaction
// @Failure      401   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /transactions [get]
func (h *handler) listTransactions(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	var (
		fromPtr *time.Time
		toPtr   *time.Time
	)

	fromStr := c.Query("from", "")
	if fromStr != "" {
		parsed, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return common.JSONError(c, http.StatusBadRequest, "некорректный формат from, ожидается YYYY-MM-DD")
		}
		fromPtr = &parsed
	}

	toStr := c.Query("to", "")
	if toStr != "" {
		parsed, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return common.JSONError(c, http.StatusBadRequest, "некорректный формат to, ожидается YYYY-MM-DD")
		}
		// включаем весь день
		parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		toPtr = &parsed
	}

	txs, err := h.repo.ListByUser(c.Context(), userID, fromPtr, toPtr)
	if err != nil {
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении транзакций")
	}

	return c.JSON(txs)
}

// getTransaction godoc
// @Summary      Детали транзакции
// @Description  Возвращает детальную информацию по одной транзакции
// @Tags         transactions
// @Produce      json
// @Param        id   path      string  true  "ID транзакции"
// @Success      200  {object}  Transaction
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /transactions/{id} [get]
func (h *handler) getTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return common.JSONError(c, http.StatusBadRequest, "id is required")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	tx, err := h.repo.GetByID(c.Context(), userID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return common.JSONError(c, http.StatusNotFound, "транзакция не найдена")
		}
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при получении транзакции")
	}

	return c.JSON(tx)
}

