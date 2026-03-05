package subscriptions

import (
	"context"
	"database/sql"
)

// Repository инкапсулирует работу с таблицей subscriptions.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий подписок.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create создаёт новую подписку.
func (r *Repository) Create(ctx context.Context, s *Subscription) (*Subscription, error) {
	const query = `
		insert into public.subscriptions (
			user_id,
			service_name,
			category,
			price,
			currency,
			billing_period,
			next_charge_at,
			status,
			cancel_url,
			support_email
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		returning id, user_id, service_name, category, price, currency,
		          billing_period, next_charge_at, status, cancel_url,
		          support_email, created_at, updated_at
	`

	row := r.db.QueryRowContext(
		ctx,
		query,
		s.UserID,
		s.ServiceName,
		s.Category,
		s.Price,
		s.Currency,
		s.BillingPeriod,
		s.NextChargeAt,
		s.Status,
		s.CancelURL,
		s.SupportEmail,
	)

	var created Subscription
	if err := row.Scan(
		&created.ID,
		&created.UserID,
		&created.ServiceName,
		&created.Category,
		&created.Price,
		&created.Currency,
		&created.BillingPeriod,
		&created.NextChargeAt,
		&created.Status,
		&created.CancelURL,
		&created.SupportEmail,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &created, nil
}

// ListByUser возвращает все подписки пользователя.
func (r *Repository) ListByUser(ctx context.Context, userID string) ([]Subscription, error) {
	const query = `
		select id, user_id, service_name, category, price, currency,
		       billing_period, next_charge_at, status, cancel_url,
		       support_email, created_at, updated_at
		from public.subscriptions
		where user_id = $1
		order by created_at desc
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Subscription

	for rows.Next() {
		var s Subscription
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.ServiceName,
			&s.Category,
			&s.Price,
			&s.Currency,
			&s.BillingPeriod,
			&s.NextChargeAt,
			&s.Status,
			&s.CancelURL,
			&s.SupportEmail,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetByID возвращает подписку по id и принадлежности пользователю.
func (r *Repository) GetByID(ctx context.Context, userID, id string) (*Subscription, error) {
	const query = `
		select id, user_id, service_name, category, price, currency,
		       billing_period, next_charge_at, status, cancel_url,
		       support_email, created_at, updated_at
		from public.subscriptions
		where id = $1 and user_id = $2
	`

	row := r.db.QueryRowContext(ctx, query, id, userID)

	var s Subscription
	if err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.ServiceName,
		&s.Category,
		&s.Price,
		&s.Currency,
		&s.BillingPeriod,
		&s.NextChargeAt,
		&s.Status,
		&s.CancelURL,
		&s.SupportEmail,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &s, nil
}

// Update обновляет основные поля подписки.
func (r *Repository) Update(ctx context.Context, s *Subscription) (*Subscription, error) {
	const query = `
		update public.subscriptions
		set service_name = $1,
		    category = $2,
		    price = $3,
		    currency = $4,
		    billing_period = $5,
		    next_charge_at = $6,
		    status = $7,
		    cancel_url = $8,
		    support_email = $9,
		    updated_at = now()
		where id = $10 and user_id = $11
		returning id, user_id, service_name, category, price, currency,
		          billing_period, next_charge_at, status, cancel_url,
		          support_email, created_at, updated_at
	`

	row := r.db.QueryRowContext(
		ctx,
		query,
		s.ServiceName,
		s.Category,
		s.Price,
		s.Currency,
		s.BillingPeriod,
		s.NextChargeAt,
		s.Status,
		s.CancelURL,
		s.SupportEmail,
		s.ID,
		s.UserID,
	)

	var updated Subscription
	if err := row.Scan(
		&updated.ID,
		&updated.UserID,
		&updated.ServiceName,
		&updated.Category,
		&updated.Price,
		&updated.Currency,
		&updated.BillingPeriod,
		&updated.NextChargeAt,
		&updated.Status,
		&updated.CancelURL,
		&updated.SupportEmail,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &updated, nil
}

// SetStatus меняет статус подписки.
func (r *Repository) SetStatus(ctx context.Context, userID, id, status string) error {
	const query = `
		update public.subscriptions
		set status = $1,
		    updated_at = now()
		where id = $2 and user_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, id, userID)
	return err
}
