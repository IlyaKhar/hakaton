package users

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Repository инкапсулирует работу с таблицей users.
// Это простой слой над *sql.DB, чтобы не размазывать SQL по хэндлерам.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт репозиторий пользователей.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser создаёт нового пользователя в БД.
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	const query = `
		insert into public.users (email, password_hash)
		values ($1, $2)
		returning id, name, email, password_hash, cloud_password_hash, cloud_password_enabled, created_at, updated_at
	`

	row := r.db.QueryRowContext(ctx, query, email, passwordHash)

	var u User
	var cloudPasswordHash sql.NullString
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &cloudPasswordHash, &u.CloudPasswordEnabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}

	if cloudPasswordHash.Valid {
		u.CloudPasswordHash = cloudPasswordHash.String
	}

	return &u, nil
}

// GetUserByEmail возвращает пользователя по email.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const query = `
		select id, name, email, password_hash, cloud_password_hash, cloud_password_enabled, created_at, updated_at
		from public.users
		where email = $1
	`

	row := r.db.QueryRowContext(ctx, query, email)

	var u User
	var cloudPasswordHash sql.NullString
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &cloudPasswordHash, &u.CloudPasswordEnabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}

	if cloudPasswordHash.Valid {
		u.CloudPasswordHash = cloudPasswordHash.String
	}

	return &u, nil
}

// GetUserByID возвращает пользователя по его id.
func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	const query = `
		select id, name, email, password_hash, cloud_password_hash, cloud_password_enabled, created_at, updated_at
		from public.users
		where id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)

	var u User
	var cloudPasswordHash sql.NullString
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &cloudPasswordHash, &u.CloudPasswordEnabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}

	if cloudPasswordHash.Valid {
		u.CloudPasswordHash = cloudPasswordHash.String
	}

	return &u, nil
}

// CreatePasswordResetCode создаёт код восстановления пароля.
func (r *Repository) CreatePasswordResetCode(ctx context.Context, userID, code string, expiresAt time.Time) error {
	const query = `
		insert into public.password_reset_codes (user_id, code, expires_at)
		values ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, userID, code, expiresAt)
	return err
}

// ValidatePasswordResetCode проверяет код восстановления пароля.
// Возвращает userID, если код валиден и не использован.
func (r *Repository) ValidatePasswordResetCode(ctx context.Context, code string) (string, error) {
	const query = `
		select user_id
		from public.password_reset_codes
		where code = $1
		  and used = false
		  and expires_at > now()
		order by created_at desc
		limit 1
	`
	var userID string
	err := r.db.QueryRowContext(ctx, query, code).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// MarkPasswordResetCodeAsUsed помечает код как использованный.
func (r *Repository) MarkPasswordResetCodeAsUsed(ctx context.Context, code string) error {
	const query = `
		update public.password_reset_codes
		set used = true
		where code = $1
	`
	_, err := r.db.ExecContext(ctx, query, code)
	return err
}

// UpdateUserPassword обновляет пароль пользователя.
func (r *Repository) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	const query = `
		update public.users
		set password_hash = $1, updated_at = now()
		where id = $2
	`
	_, err := r.db.ExecContext(ctx, query, passwordHash, userID)
	return err
}

// UpdateProfile обновляет профиль пользователя (имя, email).
func (r *Repository) UpdateProfile(ctx context.Context, userID string, name, email *string) error {
	const query = `
		update public.users
		set name = coalesce($1, name),
		    email = coalesce($2, email),
		    updated_at = now()
		where id = $3
	`
	_, err := r.db.ExecContext(ctx, query, name, email, userID)
	return err
}

// SetCloudPassword устанавливает облачный пароль пользователя.
func (r *Repository) SetCloudPassword(ctx context.Context, userID, cloudPasswordHash string) error {
	const query = `
		update public.users
		set cloud_password_hash = $1,
		    cloud_password_enabled = true,
		    updated_at = now()
		where id = $2
	`
	_, err := r.db.ExecContext(ctx, query, cloudPasswordHash, userID)
	return err
}

// VerifyCloudPassword проверяет облачный пароль пользователя.
func (r *Repository) VerifyCloudPassword(ctx context.Context, userID string) (bool, string, error) {
	const query = `
		select cloud_password_enabled, cloud_password_hash
		from public.users
		where id = $1
	`
	var enabled bool
	var hash string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&enabled, &hash)
	if err != nil {
		return false, "", err
	}
	return enabled, hash, nil
}

// DisableCloudPassword отключает облачный пароль.
func (r *Repository) DisableCloudPassword(ctx context.Context, userID string) error {
	const query = `
		update public.users
		set cloud_password_enabled = false,
		    cloud_password_hash = null,
		    updated_at = now()
		where id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// CreateCloudPasswordVerificationCode создаёт код подтверждения email для облачного пароля.
func (r *Repository) CreateCloudPasswordVerificationCode(ctx context.Context, userID, code, cloudPasswordHash string, expiresAt time.Time) error {
	const query = `
		insert into public.cloud_password_verification_codes (user_id, code, cloud_password_hash, expires_at)
		values ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, userID, code, cloudPasswordHash, expiresAt)
	return err
}

// ValidateCloudPasswordVerificationCode проверяет код подтверждения email и возвращает хеш пароля.
func (r *Repository) ValidateCloudPasswordVerificationCode(ctx context.Context, userID, code string) (bool, string, error) {
	const query = `
		select cloud_password_hash
		from public.cloud_password_verification_codes
		where user_id = $1
		  and code = $2
		  and used = false
		  and expires_at > now()
		limit 1
	`
	var hash string
	err := r.db.QueryRowContext(ctx, query, userID, code).Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, hash, nil
}

// MarkCloudPasswordVerificationCodeAsUsed помечает код как использованный.
func (r *Repository) MarkCloudPasswordVerificationCodeAsUsed(ctx context.Context, userID, code string) error {
	const query = `
		update public.cloud_password_verification_codes
		set used = true
		where user_id = $1 and code = $2
	`
	_, err := r.db.ExecContext(ctx, query, userID, code)
	return err
}
