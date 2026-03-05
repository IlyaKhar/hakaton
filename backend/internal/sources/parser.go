package sources

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hakaton/subscriptions-backend/internal/subscriptions"
	"github.com/hakaton/subscriptions-backend/internal/transactions"
)

// ParserService парсит файлы выписок и создаёт транзакции.
type ParserService struct {
	db              *sql.DB
	transactionsRepo *transactions.Repository
	subscriptionsRepo *subscriptions.Repository
}

// NewParserService создаёт сервис парсинга.
func NewParserService(db *sql.DB) *ParserService {
	return &ParserService{
		db:                db,
		transactionsRepo:  transactions.NewRepository(db),
		subscriptionsRepo: subscriptions.NewRepository(db),
	}
}

// ParsedTransaction описывает распарсенную транзакцию из файла.
type ParsedTransaction struct {
	Amount         float64
	Currency       string
	ChargedAt      time.Time
	RawDescription string
	ServiceName    string
	Category       string
}

// ParseTextFile парсит текстовый файл (CSV, простой текст) и извлекает транзакции.
func (s *ParserService) ParseTextFile(ctx context.Context, content string, userID, sourceID string) ([]ParsedTransaction, error) {
	var result []ParsedTransaction

	// Разбиваем на строки
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Пытаемся распарсить строку как транзакцию
		// Форматы:
		// 1. CSV: дата, сумма, описание
		// 2. Простой текст: ищем паттерны "дата сумма описание"
		// 3. Email-уведомление: ищем паттерны типа "списание", "оплата", суммы и даты

		parsed := s.parseLine(line)
		if parsed != nil {
			result = append(result, *parsed)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	return result, nil
}

// parseLine пытается распарсить одну строку файла.
func (s *ParserService) parseLine(line string) *ParsedTransaction {
	// Паттерны для поиска транзакций
	// 1. Дата в формате DD.MM.YYYY или YYYY-MM-DD
	datePattern := regexp.MustCompile(`(\d{1,2}[./-]\d{1,2}[./-]\d{2,4})`)
	// 2. Сумма (число с возможными пробелами/запятыми)
	amountPattern := regexp.MustCompile(`(\d+[\s,.]?\d*)\s*(руб|RUB|₽|USD|\$)`)
	// 3. Названия сервисов
	servicePatterns := map[string]string{
		"netflix":           "Netflix",
		"spotify":           "Spotify",
		"yandex.plus":       "Yandex Plus",
		"яндекс.плюс":       "Yandex Plus",
		"kinopoisk":         "Kinopoisk",
		"окко":              "Okko",
		"ivi":               "IVI",
		"apple":             "Apple",
		"google":            "Google",
		"microsoft":         "Microsoft",
		"adobe":             "Adobe",
		"яндекс.диск":       "Yandex Disk",
		"dropbox":           "Dropbox",
		"icloud":            "iCloud",
		"telegram":          "Telegram Premium",
		"discord":           "Discord Nitro",
		"twitch":            "Twitch",
		"youtube":           "YouTube Premium",
		"яндекс.музыка":     "Yandex Music",
		"deezer":            "Deezer",
		"soundcloud":        "SoundCloud",
	}

	// Ищем дату
	dateMatches := datePattern.FindStringSubmatch(line)
	if len(dateMatches) == 0 {
		return nil
	}

	// Парсим дату
	var chargedAt time.Time
	var err error
	dateStr := dateMatches[1]
	
	// Пробуем разные форматы
	formats := []string{"02.01.2006", "02/01/2006", "2006-01-02", "02.01.06", "2.1.2006"}
	for _, format := range formats {
		chargedAt, err = time.Parse(format, dateStr)
		if err == nil {
			break
		}
	}
	
	if err != nil {
		// Если не удалось распарсить, используем текущую дату
		chargedAt = time.Now()
	}

	// Ищем сумму
	amountMatches := amountPattern.FindStringSubmatch(line)
	if len(amountMatches) == 0 {
		return nil
	}

	amountStr := strings.ReplaceAll(strings.ReplaceAll(amountMatches[1], " ", ""), ",", ".")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil
	}

	currency := "RUB"
	if strings.Contains(strings.ToLower(amountMatches[0]), "usd") || strings.Contains(amountMatches[0], "$") {
		currency = "USD"
	}

	// Ищем сервис в описании
	lineLower := strings.ToLower(line)
	var serviceName, category string

	for pattern, service := range servicePatterns {
		if strings.Contains(lineLower, pattern) {
			serviceName = service
			// Определяем категорию
			if strings.Contains(pattern, "netflix") || strings.Contains(pattern, "spotify") || 
			   strings.Contains(pattern, "yandex.plus") || strings.Contains(pattern, "kinopoisk") ||
			   strings.Contains(pattern, "окко") || strings.Contains(pattern, "ivi") ||
			   strings.Contains(pattern, "youtube") || strings.Contains(pattern, "twitch") {
				category = "streaming"
			} else if strings.Contains(pattern, "apple") || strings.Contains(pattern, "google") ||
			          strings.Contains(pattern, "microsoft") || strings.Contains(pattern, "adobe") {
				category = "software"
			} else if strings.Contains(pattern, "диск") || strings.Contains(pattern, "dropbox") ||
			          strings.Contains(pattern, "icloud") {
				category = "storage"
			} else if strings.Contains(pattern, "telegram") || strings.Contains(pattern, "discord") {
				category = "communication"
			} else {
				category = "other"
			}
			break
		}
	}

	// Если сервис не найден, используем описание
	if serviceName == "" {
		serviceName = "Неизвестный сервис"
		category = "other"
	}

	return &ParsedTransaction{
		Amount:         amount,
		Currency:       currency,
		ChargedAt:      chargedAt,
		RawDescription: line,
		ServiceName:    serviceName,
		Category:       category,
	}
}

// ProcessParsedTransactions обрабатывает распарсенные транзакции:
// создаёт/обновляет подписки и создаёт транзакции.
func (s *ParserService) ProcessParsedTransactions(ctx context.Context, parsed []ParsedTransaction, userID, sourceID string) error {
	for _, p := range parsed {
		// Ищем существующую подписку по сервису
		subs, err := s.subscriptionsRepo.ListByUser(ctx, userID)
		if err != nil {
			log.Printf("warning: failed to list subscriptions: %v", err)
		}

		var subscriptionID *string
		var found bool
		for _, sub := range subs {
			if strings.EqualFold(sub.ServiceName, p.ServiceName) && sub.Status == "active" {
				subscriptionID = &sub.ID
				found = true
				break
			}
		}

		// Если подписка не найдена, создаём новую
		if !found {
			// Определяем период оплаты (по умолчанию месяц)
			billingPeriod := "month"
			nextChargeAt := p.ChargedAt.AddDate(0, 1, 0) // через месяц

			newSub := &subscriptions.Subscription{
				UserID:        userID,
				ServiceName:   p.ServiceName,
				Category:      p.Category,
				Price:         p.Amount,
				Currency:      p.Currency,
				BillingPeriod: billingPeriod,
				NextChargeAt:  &nextChargeAt,
				Status:        "active",
			}

			created, err := s.subscriptionsRepo.Create(ctx, newSub)
			if err != nil {
				log.Printf("warning: failed to create subscription for %s: %v", p.ServiceName, err)
			} else {
				subscriptionID = &created.ID
			}
		}

		// Создаём транзакцию
		tx := &transactions.Transaction{
			UserID:         userID,
			SourceID:       &sourceID,
			SubscriptionID: subscriptionID,
			Amount:         p.Amount,
			Currency:       p.Currency,
			ChargedAt:      p.ChargedAt,
			RawDescription: p.RawDescription,
		}

		if _, err := s.transactionsRepo.Create(ctx, tx); err != nil {
			log.Printf("warning: failed to create transaction: %v", err)
		}
	}

	return nil
}
