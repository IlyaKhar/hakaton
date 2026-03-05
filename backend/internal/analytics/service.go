package analytics

import (
	"context"
	"database/sql"
	"time"
)

// SummaryResult представляет краткую сводку трат.
type SummaryResult struct {
	TotalAmount       float64 `json:"totalAmount"`
	PreviousTotal     float64 `json:"previousTotal"`
	ChangeAbsolute    float64 `json:"changeAbsolute"`
	ChangePercentage  float64 `json:"changePercentage"`
}

// Service инкапсулирует бизнес-логику аналитики.
type Service struct {
	db *sql.DB
}

// NewService создаёт сервис аналитики.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CategoryStat описывает сумму трат по категории.
type CategoryStat struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

// ServiceStat описывает сумму трат по сервису.
type ServiceStat struct {
	ServiceName string  `json:"serviceName"`
	Amount      float64 `json:"amount"`
}

// GetSummary возвращает сумму трат за период и за предыдущий такой же период.
// Важно: это пример, дальше можно усложнить или оптимизировать.
func (s *Service) GetSummary(ctx context.Context, userID string, from, to time.Time) (*SummaryResult, error) {
	const baseQuery = `
		select coalesce(sum(amount), 0)
		from public.transactions
		where user_id = $1 and charged_at >= $2 and charged_at < $3
	`

	var current float64
	if err := s.db.QueryRowContext(ctx, baseQuery, userID, from, to).Scan(&current); err != nil {
		return nil, err
	}

	// Предыдущий период такой же длины.
	prevFrom := from.Add(from.Sub(to))
	prevTo := from

	var prev float64
	if err := s.db.QueryRowContext(ctx, baseQuery, userID, prevFrom, prevTo).Scan(&prev); err != nil {
		return nil, err
	}

	changeAbs := current - prev
	var changePct float64
	if prev != 0 {
		changePct = (changeAbs / prev) * 100
	}

	return &SummaryResult{
		TotalAmount:      current,
		PreviousTotal:    prev,
		ChangeAbsolute:   changeAbs,
		ChangePercentage: changePct,
	}, nil
}

// GetByCategories возвращает суммы трат по категориям за период.
func (s *Service) GetByCategories(ctx context.Context, userID string, from, to time.Time) ([]CategoryStat, error) {
	const query = `
		select coalesce(category, '') as category, coalesce(sum(amount), 0) as amount
		from public.transactions t
		join public.subscriptions s on t.subscription_id = s.id
		where t.user_id = $1 and t.charged_at >= $2 and t.charged_at < $3
		group by category
		order by amount desc
	`

	rows, err := s.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CategoryStat
	for rows.Next() {
		var cs CategoryStat
		if err := rows.Scan(&cs.Category, &cs.Amount); err != nil {
			return nil, err
		}
		result = append(result, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetByServices возвращает суммы трат по сервисам за период.
func (s *Service) GetByServices(ctx context.Context, userID string, from, to time.Time) ([]ServiceStat, error) {
	const query = `
		select coalesce(s.service_name, '') as service_name, coalesce(sum(t.amount), 0) as amount
		from public.transactions t
		join public.subscriptions s on t.subscription_id = s.id
		where t.user_id = $1 and t.charged_at >= $2 and t.charged_at < $3
		group by s.service_name
		order by amount desc
	`

	rows, err := s.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ServiceStat
	for rows.Next() {
		var ss ServiceStat
		if err := rows.Scan(&ss.ServiceName, &ss.Amount); err != nil {
			return nil, err
		}
		result = append(result, ss)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
