package recommendations

import (
	"context"
	"database/sql"
)

// Repository отвечает за выборку альтернативных сервисов.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий рекомендаций.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListByCategory возвращает альтернативы по категории с ограничением цены.
func (r *Repository) ListByCategory(ctx context.Context, category string, maxPrice float64) ([]RecommendationAlternative, error) {
	const query = `
		select id, category, service_name, price, billing_period, description
		from public.recommendation_alternatives
		where category = $1 and price <= $2
		order by price asc
	`

	rows, err := r.db.QueryContext(ctx, query, category, maxPrice)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RecommendationAlternative

	for rows.Next() {
		var rAlt RecommendationAlternative
		if err := rows.Scan(
			&rAlt.ID,
			&rAlt.Category,
			&rAlt.ServiceName,
			&rAlt.Price,
			&rAlt.BillingPeriod,
			&rAlt.Description,
		); err != nil {
			return nil, err
		}

		result = append(result, rAlt)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

