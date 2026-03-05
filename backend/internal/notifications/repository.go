package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Repository отвечает за чтение/запись уведомлений и настроек.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий уведомлений.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListSettings возвращает все настройки уведомлений пользователя.
func (r *Repository) ListSettings(ctx context.Context, userID string) ([]NotificationSetting, error) {
	const query = `
		select id, user_id, type, channels, enabled
		from public.notification_settings
		where user_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []NotificationSetting

	for rows.Next() {
		var (
			s   NotificationSetting
			raw []byte
		)

		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &raw, &s.Enabled); err != nil {
			return nil, err
		}

		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &s.Channels)
		}

		result = append(result, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// UpsertSettings сохраняет настройки уведомлений пользователя.
func (r *Repository) UpsertSettings(ctx context.Context, userID string, settings []NotificationSetting) error {
	const deleteQuery = `
		delete from public.notification_settings
		where user_id = $1
	`
	if _, err := r.db.ExecContext(ctx, deleteQuery, userID); err != nil {
		return err
	}

	const insertQuery = `
		insert into public.notification_settings (user_id, type, channels, enabled)
		values ($1, $2, $3, $4)
	`

	for _, s := range settings {
		chBytes, err := json.Marshal(s.Channels)
		if err != nil {
			return err
		}
		if _, err := r.db.ExecContext(ctx, insertQuery, userID, s.Type, chBytes, s.Enabled); err != nil {
			return err
		}
	}

	return nil
}

// ListNotifications возвращает уведомления пользователя.
func (r *Repository) ListNotifications(ctx context.Context, userID string) ([]Notification, error) {
	const query = `
		select id, user_id, subscription_id, type, payload, status, created_at, read_at
		from public.notifications
		where user_id = $1
		order by created_at desc
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Notification

	for rows.Next() {
		var (
			n   Notification
			raw []byte
		)

		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.SubscriptionID,
			&n.Type,
			&raw,
			&n.Status,
			&n.CreatedAt,
			&n.ReadAt,
		); err != nil {
			return nil, err
		}

		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &n.Payload)
		}

		result = append(result, n)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// MarkRead помечает уведомление как прочитанное.
func (r *Repository) MarkRead(ctx context.Context, userID, id string) error {
	const query = `
		update public.notifications
		set status = 'read',
		    read_at = now()
		where id = $1 and user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}
