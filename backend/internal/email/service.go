package email

import (
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
)

// Service инкапсулирует логику отправки email.
type Service struct {
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
}

// NewService создаёт сервис для отправки email.
// Если SMTP настройки не заданы, сервис будет работать в режиме логирования (для разработки).
func NewService(smtpHost, smtpPort, smtpUser, smtpPassword, fromEmail string) *Service {
	port, _ := strconv.Atoi(smtpPort)
	if port == 0 {
		port = 587 // дефолтный порт для TLS
	}

	return &Service{
		smtpHost:     smtpHost,
		smtpPort:     port,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
	}
}

// SendPasswordResetCode отправляет код восстановления пароля на email.
func (s *Service) SendPasswordResetCode(toEmail, code string) error {
	// Если SMTP не настроен, просто логируем (для разработки)
	if s.smtpHost == "" || s.smtpUser == "" {
		fmt.Printf("[EMAIL] Password reset code for %s: %s\n", toEmail, code)
		return nil
	}

	subject := "Код восстановления пароля"
	body := fmt.Sprintf(`
Здравствуйте!

Вы запросили восстановление пароля для вашего аккаунта.

Ваш код восстановления: %s

Код действителен в течение 15 минут.

Если вы не запрашивали восстановление пароля, просто проигнорируйте это письмо.

С уважением,
Команда Subscriptions
`, code)

	return s.sendEmail(toEmail, subject, body)
}

// SendCloudPasswordVerificationCode отправляет код подтверждения для установки облачного пароля.
func (s *Service) SendCloudPasswordVerificationCode(toEmail, code string) error {
	// Если SMTP не настроен, просто логируем (для разработки)
	if s.smtpHost == "" || s.smtpUser == "" {
		fmt.Printf("[EMAIL] Cloud password verification code for %s: %s\n", toEmail, code)
		return nil
	}

	subject := "Подтверждение установки облачного пароля"
	body := fmt.Sprintf(`
Здравствуйте!

Вы запросили установку облачного пароля для вашего аккаунта.

Ваш код подтверждения: %s

Код действителен в течение 15 минут.

Если вы не запрашивали установку облачного пароля, просто проигнорируйте это письмо.

С уважением,
Команда Subscriptions
`, code)

	return s.sendEmail(toEmail, subject, body)
}

// sendEmail отправляет email через SMTP.
func (s *Service) sendEmail(toEmail, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.fromEmail)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	// Настраиваем dialer в зависимости от порта
	d := gomail.NewDialer(s.smtpHost, s.smtpPort, s.smtpUser, s.smtpPassword)

	// Порт 465 использует SSL, порт 587 использует STARTTLS (TLS)
	// В gomail.v2 для порта 587 STARTTLS включается автоматически
	if s.smtpPort == 465 {
		d.SSL = true
	} else {
		d.SSL = false
		// Для порта 587 STARTTLS включается автоматически
	}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("ошибка при отправке email: %w", err)
	}

	return nil
}
