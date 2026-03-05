package forecast

import (
	"context"
	"database/sql"
)

// YearForecast описывает результат прогноза на год.
type YearForecast struct {
	TotalYearCost          float64 `json:"totalYearCost"`
	BaselineMonthlyCost    float64 `json:"baselineMonthlyCost"`
	EconomyIfCancelLowUsed float64 `json:"economyIfCancelLowUsed"`
}

// Service — обёртка над *sql.DB для логики прогноза.
type Service struct {
	db *sql.DB
}

// NewService создаёт сервис прогноза.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// BuildYearForecast считает простой прогноз:
// - нормализует все активные подписки до стоимости в месяц
// - умножает сумму на 12
// TODO: добавить логику economyIfCancelLowUsed (по usage_events).
func (s *Service) BuildYearForecast(ctx context.Context, userID string) (*YearForecast, error) {
	const query = `
		select
			price,
			billing_period
		from public.subscriptions
		where user_id = $1 and status = 'active'
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monthlySum float64

	for rows.Next() {
		var (
			price         float64
			billingPeriod string
		)

		if err := rows.Scan(&price, &billingPeriod); err != nil {
			return nil, err
		}

		switch billingPeriod {
		case "year":
			monthlySum += price / 12
		default:
			monthlySum += price
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &YearForecast{
		TotalYearCost:       monthlySum * 12,
		BaselineMonthlyCost: monthlySum,
		// economy пока 0 — это задел на дальнейшую реализацию.
		EconomyIfCancelLowUsed: 0,
	}, nil
}

