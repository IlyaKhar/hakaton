package payment_cards

import (
	"context"
	"database/sql"
	"log"
)

// Repository инкапсулирует работу с таблицей payment_cards.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий для работы с картами.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListByUser возвращает все карты пользователя.
func (r *Repository) ListByUser(ctx context.Context, userID string) ([]PaymentCard, error) {
	const query = `
		select id, user_id, last_four_digits, card_mask, card_type, expiry_month, expiry_year, holder_name, is_default, created_at, updated_at
		from public.payment_cards
		where user_id = $1
		order by is_default desc, created_at desc
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []PaymentCard
	for rows.Next() {
		var card PaymentCard
		if err := rows.Scan(
			&card.ID,
			&card.UserID,
			&card.LastFourDigits,
			&card.CardMask,
			&card.CardType,
			&card.ExpiryMonth,
			&card.ExpiryYear,
			&card.HolderName,
			&card.IsDefault,
			&card.CreatedAt,
			&card.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cards, nil
}

// Create создаёт новую карту.
func (r *Repository) Create(ctx context.Context, card *PaymentCard) (*PaymentCard, error) {
	log.Printf("repository: creating card for user_id=%s, last_four=%s", card.UserID, card.LastFourDigits)

	const query = `
		insert into public.payment_cards (user_id, last_four_digits, card_mask, card_type, expiry_month, expiry_year, holder_name, is_default)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		returning id, user_id, last_four_digits, card_mask, card_type, expiry_month, expiry_year, holder_name, is_default, created_at, updated_at
	`

	row := r.db.QueryRowContext(ctx, query,
		card.UserID,
		card.LastFourDigits,
		card.CardMask,
		card.CardType,
		card.ExpiryMonth,
		card.ExpiryYear,
		card.HolderName,
		card.IsDefault,
	)

	var created PaymentCard
	if err := row.Scan(
		&created.ID,
		&created.UserID,
		&created.LastFourDigits,
		&created.CardMask,
		&created.CardType,
		&created.ExpiryMonth,
		&created.ExpiryYear,
		&created.HolderName,
		&created.IsDefault,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		log.Printf("repository: error scanning created card: %v", err)
		return nil, err
	}

	log.Printf("repository: card created successfully: id=%s, user_id=%s", created.ID, created.UserID)

	// Если эта карта помечена как default, снимаем флаг с остальных
	if card.IsDefault {
		if err := r.unsetOtherDefaults(ctx, card.UserID, created.ID); err != nil {
			// Логируем, но не падаем
			_ = err
		}
	}

	return &created, nil
}

// unsetOtherDefaults снимает флаг is_default с других карт пользователя.
func (r *Repository) unsetOtherDefaults(ctx context.Context, userID, excludeCardID string) error {
	const query = `
		update public.payment_cards
		set is_default = false, updated_at = now()
		where user_id = $1 and id != $2 and is_default = true
	`
	_, err := r.db.ExecContext(ctx, query, userID, excludeCardID)
	return err
}

// Delete удаляет карту.
func (r *Repository) Delete(ctx context.Context, userID, cardID string) error {
	const query = `
		delete from public.payment_cards
		where user_id = $1 and id = $2
	`
	_, err := r.db.ExecContext(ctx, query, userID, cardID)
	return err
}
