package sources

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Repository инкапсулирует работу с таблицей subscription_sources.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий источников.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create создаёт новый источник данных.
func (r *Repository) Create(ctx context.Context, s *SubscriptionSource) (*SubscriptionSource, error) {
	const query = `
		insert into public.subscription_sources (user_id, type, provider, status, meta)
		values ($1, $2, $3, $4, $5)
		returning id, user_id, type, provider, status, meta, created_at, updated_at
	`

	metaBytes, err := json.Marshal(s.Meta)
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRowContext(ctx, query, s.UserID, s.Type, s.Provider, s.Status, metaBytes)

	var created SubscriptionSource
	var raw []byte
	if err := row.Scan(
		&created.ID,
		&created.UserID,
		&created.Type,
		&created.Provider,
		&created.Status,
		&raw,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &created.Meta)
	}

	return &created, nil
}

// UpdateStatus меняет статус источника (pending/connected/error).
func (r *Repository) UpdateStatus(ctx context.Context, userID, id, status string) error {
	const query = `
		update public.subscription_sources
		set status = $1,
		    updated_at = now()
		where id = $2 and user_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, id, userID)
	return err
}

// ListByUser возвращает источники пользователя.
func (r *Repository) ListByUser(ctx context.Context, userID string) ([]SubscriptionSource, error) {
	const query = `
		select id, user_id, type, provider, status, meta, created_at, updated_at
		from public.subscription_sources
		where user_id = $1
		order by created_at desc
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SubscriptionSource

	for rows.Next() {
		var (
			s   SubscriptionSource
			raw []byte
		)

		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Type,
			&s.Provider,
			&s.Status,
			&raw,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &s.Meta)
		}

		result = append(result, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetByID возвращает источник по ID (с проверкой user_id).
func (r *Repository) GetByID(ctx context.Context, userID, id string) (*SubscriptionSource, error) {
	const query = `
		select id, user_id, type, provider, status, meta, created_at, updated_at
		from public.subscription_sources
		where id = $1 and user_id = $2
	`

	row := r.db.QueryRowContext(ctx, query, id, userID)

	var s SubscriptionSource
	var raw []byte
	if err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.Type,
		&s.Provider,
		&s.Status,
		&raw,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.Meta)
	}

	return &s, nil
}
