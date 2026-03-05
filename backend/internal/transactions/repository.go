package transactions

import (
	"context"
	"database/sql"
	"strconv"
	"time"
)

// Repository инкапсулирует работу с таблицей transactions.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий транзакций.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create создаёт новую транзакцию.
func (r *Repository) Create(ctx context.Context, t *Transaction) (*Transaction, error) {
	const query = `
		insert into public.transactions (user_id, source_id, subscription_id, amount, currency, charged_at, raw_description)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, user_id, source_id, subscription_id, amount, currency, charged_at, raw_description
	`

	row := r.db.QueryRowContext(ctx, query,
		t.UserID,
		t.SourceID,
		t.SubscriptionID,
		t.Amount,
		t.Currency,
		t.ChargedAt,
		t.RawDescription,
	)

	var created Transaction
	var sourceID, subscriptionID sql.NullString
	if err := row.Scan(
		&created.ID,
		&created.UserID,
		&sourceID,
		&subscriptionID,
		&created.Amount,
		&created.Currency,
		&created.ChargedAt,
		&created.RawDescription,
	); err != nil {
		return nil, err
	}

	if sourceID.Valid {
		created.SourceID = &sourceID.String
	}
	if subscriptionID.Valid {
		created.SubscriptionID = &subscriptionID.String
	}

	return &created, nil
}

// CreateBatch создаёт несколько транзакций за раз.
func (r *Repository) CreateBatch(ctx context.Context, transactions []Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	const query = `
		insert into public.transactions (user_id, source_id, subscription_id, amount, currency, charged_at, raw_description)
		values ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, t := range transactions {
		_, err := r.db.ExecContext(ctx, query,
			t.UserID,
			t.SourceID,
			t.SubscriptionID,
			t.Amount,
			t.Currency,
			t.ChargedAt,
			t.RawDescription,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListByUser возвращает транзакции пользователя.
func (r *Repository) ListByUser(ctx context.Context, userID string, from, to *time.Time) ([]Transaction, error) {
	query := `
		select id, user_id, source_id, subscription_id, amount, currency, charged_at, raw_description
		from public.transactions
		where user_id = $1
	`
	args := []interface{}{userID}
	argIdx := 2

	if from != nil {
		query += ` and charged_at >= $` + strconv.Itoa(argIdx)
		args = append(args, *from)
		argIdx++
	}
	if to != nil {
		query += ` and charged_at <= $` + strconv.Itoa(argIdx)
		args = append(args, *to)
	}

	query += ` order by charged_at desc`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Transaction
	for rows.Next() {
		var t Transaction
		var sourceID, subscriptionID sql.NullString
		if err := rows.Scan(
			&t.ID,
			&t.UserID,
			&sourceID,
			&subscriptionID,
			&t.Amount,
			&t.Currency,
			&t.ChargedAt,
			&t.RawDescription,
		); err != nil {
			return nil, err
		}

		if sourceID.Valid {
			t.SourceID = &sourceID.String
		}
		if subscriptionID.Valid {
			t.SubscriptionID = &subscriptionID.String
		}

		result = append(result, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetByID возвращает одну транзакцию по id и userID.
func (r *Repository) GetByID(ctx context.Context, userID, id string) (*Transaction, error) {
	const query = `
		select id, user_id, source_id, subscription_id, amount, currency, charged_at, raw_description
		from public.transactions
		where id = $1 and user_id = $2
	`

	row := r.db.QueryRowContext(ctx, query, id, userID)

	var t Transaction
	var sourceID, subscriptionID sql.NullString
	if err := row.Scan(
		&t.ID,
		&t.UserID,
		&sourceID,
		&subscriptionID,
		&t.Amount,
		&t.Currency,
		&t.ChargedAt,
		&t.RawDescription,
	); err != nil {
		return nil, err
	}

	if sourceID.Valid {
		t.SourceID = &sourceID.String
	}
	if subscriptionID.Valid {
		t.SubscriptionID = &subscriptionID.String
	}

	return &t, nil
}

