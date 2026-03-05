package auth

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/gofiber/fiber/v2"
	authjwt "github.com/hakaton/subscriptions-backend/internal/auth"
	"github.com/hakaton/subscriptions-backend/internal/config"
	"github.com/hakaton/subscriptions-backend/internal/email"
	"github.com/hakaton/subscriptions-backend/internal/payment_cards"
	"github.com/hakaton/subscriptions-backend/internal/users"
	"golang.org/x/crypto/bcrypt"
)

// registerRequest описывает входные данные для регистрации.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handler хранит зависимости для auth-роутов.
// Через эту структуру мы передаём в хэндлеры db, config и репозитории.
type handler struct {
	db               *sql.DB
	cfg              *config.Config
	usersRepo        *users.Repository
	emailSvc         *email.Service
	paymentCardsRepo *payment_cards.Repository
}

// RegisterPublicRoutes регистрирует публичные auth-роуты (без JWT).
func RegisterPublicRoutes(r fiber.Router, db *sql.DB, cfg *config.Config) {
	emailSvc := email.NewService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	h := &handler{
		db:               db,
		cfg:              cfg,
		usersRepo:        users.NewRepository(db),
		emailSvc:         emailSvc,
		paymentCardsRepo: payment_cards.NewRepository(db),
	}

	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/forgot-password", h.forgotPassword)
	r.Post("/verify-code", h.verifyCode)
	r.Post("/reset-password", h.resetPassword)
}

// RegisterProtectedRoutes регистрирует защищённые auth-роуты (требуют JWT).
func RegisterProtectedRoutes(r fiber.Router, db *sql.DB, cfg *config.Config) {
	emailSvc := email.NewService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	h := &handler{
		db:               db,
		cfg:              cfg,
		usersRepo:        users.NewRepository(db),
		emailSvc:         emailSvc,
		paymentCardsRepo: payment_cards.NewRepository(db),
	}

	r.Get("/me", h.me)
	r.Put("/me", h.updateProfile)
	r.Post("/change-password", h.changePassword)
	r.Post("/logout", h.logout)
	r.Post("/cloud-password", h.setupCloudPassword)
	r.Post("/verify-cloud-password", h.verifyCloudPassword)
}

// generateAccessToken — вспомогательная функция для создания access-токена.
func (h *handler) generateAccessToken(userID string) (string, error) {
	if h.cfg.JwtSecret == "" {
		return "", errors.New("JWT_SECRET не задан")
	}

	// TTL access-токена: 15 минут
	return authjwt.GenerateToken(userID, h.cfg.JwtSecret, 15*time.Minute)
}

// register godoc
// @Summary      Регистрация пользователя
// @Description  Создаёт нового пользователя и возвращает JWT токен
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body      registerRequest true "Данные для регистрации"
// @Success      201     {object}  map[string]any
// @Failure      400     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/register [post]
func (h *handler) register(c *fiber.Ctx) error {
	var req registerRequest

	// Пытаемся распарсить JSON из тела запроса
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	// Проверяем обязательные поля
	if req.Email == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "email и password обязательны",
		})
	}

	// Проверяем, что пользователя с таким email ещё нет
	// Проверяем, что пользователя с таким email ещё нет
	existing, err := h.usersRepo.GetUserByEmail(c.Context(), req.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return c.Status(500).JSON(fiber.Map{
			// ВАЖНО: добавляем err.Error() внутрь сообщения
			"message": "ошибка при проверке пользователя: " + err.Error(),
		})
	}

	if existing != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "пользователь с таким email уже существует",
		})
	}

	// Хешируем пароль через bcrypt
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при хешировании пароля",
		})
	}

	// Создаём пользователя в БД
	user, err := h.usersRepo.CreateUser(c.Context(), req.Email, string(hashed))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при создании пользователя: " + err.Error(),
		})
	}

	// Генерируем access-токен
	token, err := h.generateAccessToken(user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при генерации токена",
		})
	}

	// Возвращаем пользователя и токен
	return c.Status(201).JSON(fiber.Map{
		"user":  user,  // PasswordHash не уйдёт наружу, у него json:"-"
		"token": token, // access-токен
	})
}

// loginRequest можно использовать тот же, что и для регистрации (email+password),
// но вынесем в отдельный тип на будущее.
type loginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CloudPassword string `json:"cloudPassword,omitempty"` // опционально, требуется если установлен
}

// login godoc
// @Summary      Авторизация пользователя
// @Description  Проверяет email/пароль и возвращает JWT токен. Если установлен облачный пароль, требуется также cloudPassword
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body      loginRequest true "Данные для авторизации (cloudPassword опционален, требуется если установлен облачный пароль)"
// @Success      200     {object}  map[string]any
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/login [post]
func (h *handler) login(c *fiber.Ctx) error {
	var req loginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "email и password обязательны",
		})
	}

	// Ищем пользователя по email
	user, err := h.usersRepo.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(401).JSON(fiber.Map{
				"message": "неверный email или пароль",
			})
		}

		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при поиске пользователя",
		})
	}

	// Сравниваем основной пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{
			"message": "неверный email или пароль",
		})
	}

	// Проверяем облачный пароль, если он установлен
	if user.CloudPasswordEnabled {
		if req.CloudPassword == "" {
			return c.Status(401).JSON(fiber.Map{
				"message":               "требуется облачный пароль",
				"requiresCloudPassword": true,
			})
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.CloudPasswordHash), []byte(req.CloudPassword)); err != nil {
			return c.Status(401).JSON(fiber.Map{
				"message": "неверный облачный пароль",
			})
		}
	}

	// Генерируем access-токен
	token, err := h.generateAccessToken(user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при генерации токена",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"user":  user,
		"token": token,
	})
}

// me — метод-обработчик, который будет возвращать текущего пользователя.
// Здесь мы предполагаем, что auth-middleware уже положил userID в c.Locals("userID").

// me godoc
// @Summary      Текущий пользователь
// @Description  Возвращает данные текущего пользователя по JWT токену, включая привязанные платежные карты
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200     {object}  map[string]any "user: объект пользователя, cards: массив карт"
// @Failure      401     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/me [get]
func (h *handler) me(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{
			"message": "не авторизован",
		})
	}

	user, err := h.usersRepo.GetUserByID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(404).JSON(fiber.Map{
				"message": "пользователь не найден",
			})
		}

		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при получении пользователя",
		})
	}

	// Получаем карты пользователя
	cards, err := h.paymentCardsRepo.ListByUser(c.Context(), userID)
	if err != nil {
		// Логируем, но не падаем - карты опциональны
		log.Printf("warning: failed to get payment cards for user %s: %v", userID, err)
		cards = []payment_cards.PaymentCard{} // пустой массив вместо nil
	}

	// Возвращаем пользователя с картами
	return c.Status(200).JSON(fiber.Map{
		"user":  user,
		"cards": cards,
	})
}

// updateProfileRequest описывает запрос на обновление профиля.
type updateProfileRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

// updateProfile godoc
// @Summary      Обновление профиля
// @Description  Обновляет имя и/или email пользователя
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload body      updateProfileRequest true "Новые данные профиля"
// @Success      200     {object}  users.User
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/me [put]
func (h *handler) updateProfile(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{
			"message": "не авторизован",
		})
	}

	var req updateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	// Обновляем профиль
	if err := h.usersRepo.UpdateProfile(c.Context(), userID, req.Name, req.Email); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при обновлении профиля",
		})
	}

	// Возвращаем обновлённого пользователя
	user, err := h.usersRepo.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при получении пользователя",
		})
	}

	return c.Status(200).JSON(user)
}

// changePasswordRequest описывает запрос на изменение пароля.
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// changePassword godoc
// @Summary      Изменение пароля
// @Description  Изменяет основной пароль пользователя
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload body      changePasswordRequest true "Старый и новый пароль"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/change-password [post]
func (h *handler) changePassword(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{
			"message": "не авторизован",
		})
	}

	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "oldPassword и newPassword обязательны",
		})
	}

	// Получаем пользователя
	user, err := h.usersRepo.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при получении пользователя",
		})
	}

	// Проверяем старый пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return c.Status(401).JSON(fiber.Map{
			"message": "неверный старый пароль",
		})
	}

	// Хешируем новый пароль
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при хешировании пароля",
		})
	}

	// Обновляем пароль
	if err := h.usersRepo.UpdateUserPassword(c.Context(), userID, string(hashed)); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при обновлении пароля",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "пароль успешно изменён",
	})
}

// logout godoc
// @Summary      Выход из аккаунта
// @Description  Выход из аккаунта (JWT stateless, поэтому просто возвращает успех)
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Router       /auth/logout [post]
func (h *handler) logout(c *fiber.Ctx) error {
	// JWT stateless, поэтому просто возвращаем успех
	// На клиенте нужно удалить токен из localStorage/sessionStorage
	return c.Status(200).JSON(fiber.Map{
		"message": "выход выполнен успешно",
	})
}

// setupCloudPasswordRequest описывает запрос на установку облачного пароля.
type setupCloudPasswordRequest struct {
	CloudPassword string `json:"cloudPassword"`
}

// setupCloudPassword godoc
// @Summary      Установка облачного пароля
// @Description  Отправляет код подтверждения на email для установки облачного пароля
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload body      setupCloudPasswordRequest true "Облачный пароль"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/cloud-password [post]
func (h *handler) setupCloudPassword(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{
			"message": "не авторизован",
		})
	}

	var req setupCloudPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.CloudPassword == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "cloudPassword обязателен",
		})
	}

	// Получаем пользователя
	user, err := h.usersRepo.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при получении пользователя",
		})
	}

	if user.Email == nil || *user.Email == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "email не указан",
		})
	}

	// Хешируем облачный пароль
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.CloudPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при хешировании пароля",
		})
	}

	// Генерируем код подтверждения
	code := generateResetCode()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Сохраняем код и хеш пароля в БД
	if err := h.usersRepo.CreateCloudPasswordVerificationCode(c.Context(), userID, code, string(hashed), expiresAt); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при создании кода подтверждения",
		})
	}

	// Отправляем код на email
	if err := h.emailSvc.SendCloudPasswordVerificationCode(*user.Email, code); err != nil {
		log.Printf("warning: failed to send cloud password verification email to %s: %v", *user.Email, err)
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "код подтверждения отправлен на email",
	})
}

// verifyCloudPasswordRequest описывает запрос на подтверждение облачного пароля.
type verifyCloudPasswordRequest struct {
	Code string `json:"code"`
}

// verifyCloudPassword godoc
// @Summary      Подтверждение облачного пароля
// @Description  Проверяет код из email и устанавливает облачный пароль
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload body      verifyCloudPasswordRequest true "Код подтверждения и облачный пароль"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/verify-cloud-password [post]
func (h *handler) verifyCloudPassword(c *fiber.Ctx) error {
	raw := c.Locals("userID")
	userID, ok := raw.(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{
			"message": "не авторизован",
		})
	}

	var req verifyCloudPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.Code == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "code обязателен",
		})
	}

	// Проверяем код и получаем хеш пароля
	valid, hash, err := h.usersRepo.ValidateCloudPasswordVerificationCode(c.Context(), userID, req.Code)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при проверке кода",
		})
	}

	if !valid {
		return c.Status(401).JSON(fiber.Map{
			"message": "неверный или просроченный код",
		})
	}

	// Устанавливаем облачный пароль (хеш уже сохранён в коде)
	if err := h.usersRepo.SetCloudPassword(c.Context(), userID, hash); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при установке облачного пароля",
		})
	}

	// Помечаем код как использованный
	if err := h.usersRepo.MarkCloudPasswordVerificationCodeAsUsed(c.Context(), userID, req.Code); err != nil {
		log.Printf("warning: failed to mark cloud password verification code as used: %v", err)
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "облачный пароль успешно установлен",
	})
}

// forgotPasswordRequest описывает запрос на восстановление пароля.
type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// forgotPassword godoc
// @Summary      Запрос на восстановление пароля
// @Description  Отправляет код восстановления на email пользователя
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body      forgotPasswordRequest true "Email пользователя"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/forgot-password [post]
func (h *handler) forgotPassword(c *fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.Email == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "email обязателен",
		})
	}

	// Ищем пользователя по email
	user, err := h.usersRepo.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Для безопасности не говорим, что пользователя нет
			return c.Status(200).JSON(fiber.Map{
				"message": "если пользователь с таким email существует, код отправлен",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при поиске пользователя",
		})
	}

	// Генерируем 6-значный код
	code := generateResetCode()
	expiresAt := time.Now().Add(15 * time.Minute) // код действителен 15 минут

	// Сохраняем код в БД
	if err := h.usersRepo.CreatePasswordResetCode(c.Context(), user.ID, code, expiresAt); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при создании кода восстановления",
		})
	}

	// Отправляем код на email
	if err := h.emailSvc.SendPasswordResetCode(req.Email, code); err != nil {
		// Логируем ошибку с деталями для отладки
		log.Printf("warning: failed to send password reset email to %s: %v", req.Email, err)
		log.Printf("hint: проверь SMTP настройки в .env (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD)")
		log.Printf("hint: для Gmail попробуй порт 465 (SSL) вместо 587 (TLS)")
		// В режиме разработки (без SMTP) код всё равно будет в логах
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "если пользователь с таким email существует, код отправлен",
	})
}

// verifyCodeRequest описывает запрос на проверку кода.
type verifyCodeRequest struct {
	Code string `json:"code"`
}

// verifyCode godoc
// @Summary      Проверка кода восстановления
// @Description  Проверяет код восстановления пароля и возвращает токен для сброса пароля
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body      verifyCodeRequest true "Код восстановления"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/verify-code [post]
func (h *handler) verifyCode(c *fiber.Ctx) error {
	var req verifyCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.Code == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "код обязателен",
		})
	}

	// Проверяем код
	userID, err := h.usersRepo.ValidatePasswordResetCode(c.Context(), req.Code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(401).JSON(fiber.Map{
				"message": "неверный или просроченный код",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при проверке кода",
		})
	}

	// Помечаем код как использованный
	if err := h.usersRepo.MarkPasswordResetCodeAsUsed(c.Context(), req.Code); err != nil {
		// Логируем, но не падаем - код уже проверен
		log.Printf("warning: failed to mark code as used: %v", err)
	}

	// Генерируем временный токен для сброса пароля (действителен 10 минут)
	resetToken, err := authjwt.GenerateToken(userID, h.cfg.JwtSecret, 10*time.Minute)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при генерации токена",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"resetToken": resetToken,
		"message":    "код подтверждён",
	})
}

// resetPasswordRequest описывает запрос на сброс пароля.
type resetPasswordRequest struct {
	ResetToken string `json:"resetToken"`
	Password   string `json:"password"`
}

// resetPassword godoc
// @Summary      Сброс пароля
// @Description  Устанавливает новый пароль по токену из verify-code
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body      resetPasswordRequest true "Токен сброса и новый пароль"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  map[string]string
// @Failure      401     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /auth/reset-password [post]
func (h *handler) resetPassword(c *fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "не удалось прочитать тело запроса",
		})
	}

	if req.ResetToken == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "resetToken и password обязательны",
		})
	}

	// Проверяем токен
	userID, err := authjwt.ParseToken(req.ResetToken, h.cfg.JwtSecret)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"message": "неверный или просроченный токен",
		})
	}

	// Хешируем новый пароль
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при хешировании пароля",
		})
	}

	// Обновляем пароль
	if err := h.usersRepo.UpdateUserPassword(c.Context(), userID, string(hashed)); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "ошибка при обновлении пароля",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "пароль успешно изменён",
	})
}

// generateResetCode генерирует 6-значный код восстановления.
func generateResetCode() string {
	// Генерируем случайное число от 100000 до 999999
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback на менее безопасный метод, если crypto/rand не работает
		return fmt.Sprintf("%06d", time.Now().UnixNano()%900000+100000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}
