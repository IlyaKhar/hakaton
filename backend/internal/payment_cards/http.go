package payment_cards

import (
	"database/sql"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/common"
)

// RegisterRoutes регистрирует HTTP-роуты для работы с платежными картами.
func RegisterRoutes(r fiber.Router, db *sql.DB) {
	repo := NewRepository(db)
	h := &handler{repo: repo}

	r.Get("/", h.listCards)
	r.Post("/", h.createCard)
	r.Delete("/:id", h.deleteCard)
}

type handler struct {
	repo *Repository
}

// listCards godoc
// @Summary      Список карт
// @Description  Возвращает все платежные карты текущего пользователя
// @Tags         payment-cards
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   PaymentCard
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /payment-cards [get]
func (h *handler) listCards(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	log.Printf("listing payment cards for user_id=%s", userID)

	cards, err := h.repo.ListByUser(c.Context(), userID)
	if err != nil {
		log.Printf("error listing payment cards: %v", err)
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	log.Printf("found %d payment cards for user_id=%s", len(cards), userID)

	return c.JSON(cards)
}

// createCardRequest описывает запрос на добавление карты.
type createCardRequest struct {
	CardNumber  string  `json:"cardNumber"`  // полный номер карты (будет обработан и сохранены только последние 4 цифры)
	CardType    string  `json:"cardType"`   // Visa, Mastercard, Mir и т.д.
	ExpiryMonth int     `json:"expiryMonth"` // 1-12
	ExpiryYear  int     `json:"expiryYear"`  // 2024, 2025 и т.д.
	HolderName  *string `json:"holderName,omitempty"`
	IsDefault   bool    `json:"isDefault"`   // сделать карту по умолчанию
}

// createCard godoc
// @Summary      Добавить карту
// @Description  Добавляет новую платежную карту для текущего пользователя
// @Tags         payment-cards
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload body      createCardRequest true "Данные карты"
// @Success      201     {object}  PaymentCard
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /payment-cards [post]
func (h *handler) createCard(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	var req createCardRequest
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, http.StatusBadRequest, "не удалось прочитать тело запроса")
	}

	// Валидация
	if req.CardNumber == "" {
		return common.JSONError(c, http.StatusBadRequest, "cardNumber обязателен")
	}

	// Очищаем номер карты от пробелов и дефисов
	cardNumber := strings.ReplaceAll(strings.ReplaceAll(req.CardNumber, " ", ""), "-", "")

	// Проверяем, что номер карты состоит только из цифр и имеет правильную длину (13-19 цифр)
	if matched, _ := regexp.MatchString(`^\d{13,19}$`, cardNumber); !matched {
		return common.JSONError(c, http.StatusBadRequest, "неверный формат номера карты")
	}

	// Извлекаем последние 4 цифры
	lastFour := cardNumber[len(cardNumber)-4:]

	// Формируем маску карты (**** **** **** 1234)
	cardMask := "**** **** **** " + lastFour

	// Валидация типа карты
	if req.CardType == "" {
		// Определяем тип карты автоматически, если не указан
		req.CardType = detectCardType(cardNumber)
	}

	// Валидация срока действия
	if req.ExpiryMonth < 1 || req.ExpiryMonth > 12 {
		return common.JSONError(c, http.StatusBadRequest, "expiryMonth должен быть от 1 до 12")
	}

	if req.ExpiryYear < 2020 || req.ExpiryYear > 2100 {
		return common.JSONError(c, http.StatusBadRequest, "expiryYear должен быть от 2020 до 2100")
	}

	log.Printf("creating payment card for user_id=%s, last_four=%s, card_type=%s", userID, lastFour, req.CardType)

	card := &PaymentCard{
		UserID:        userID,
		LastFourDigits: lastFour,
		CardMask:      cardMask,
		CardType:      req.CardType,
		ExpiryMonth:   req.ExpiryMonth,
		ExpiryYear:    req.ExpiryYear,
		HolderName:    req.HolderName,
		IsDefault:     req.IsDefault,
	}

	created, err := h.repo.Create(c.Context(), card)
	if err != nil {
		log.Printf("error creating payment card for user_id=%s: %v", userID, err)
		return common.JSONError(c, http.StatusInternalServerError, "ошибка при создании карты: "+err.Error())
	}

	log.Printf("payment card created successfully: id=%s, user_id=%s, last_four=%s", created.ID, created.UserID, created.LastFourDigits)

	return c.Status(http.StatusCreated).JSON(created)
}

// deleteCard godoc
// @Summary      Удалить карту
// @Description  Удаляет платежную карту пользователя
// @Tags         payment-cards
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "ID карты"
// @Success      204 "No Content"
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /payment-cards/{id} [delete]
func (h *handler) deleteCard(c *fiber.Ctx) error {
	cardID := c.Params("id")
	if cardID == "" {
		return common.JSONError(c, http.StatusBadRequest, "id обязателен")
	}

	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return common.JSONError(c, http.StatusUnauthorized, "не авторизован")
	}

	if err := h.repo.Delete(c.Context(), userID, cardID); err != nil {
		return common.JSONError(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}

// detectCardType определяет тип карты по номеру (вспомогательная функция).
func detectCardType(cardNumber string) string {
	// Убираем пробелы и дефисы
	cleaned := strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")
	
	if len(cleaned) == 0 {
		return "Unknown"
	}

	firstDigit := cleaned[0]
	firstTwoDigits := ""
	if len(cleaned) >= 2 {
		firstTwoDigits = cleaned[:2]
	}

	// Visa: начинается с 4
	if firstDigit == '4' {
		return "Visa"
	}

	// Mastercard: начинается с 51-55 или 2221-2720
	if firstTwoDigits != "" {
		if num, err := strconv.Atoi(firstTwoDigits); err == nil {
			if (num >= 51 && num <= 55) || (num >= 2221 && num <= 2720) {
				return "Mastercard"
			}
		}
	}

	// Mir: начинается с 2
	if firstDigit == '2' {
		return "Mir"
	}

	return "Unknown"
}
