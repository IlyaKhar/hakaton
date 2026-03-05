package users

import "time"

// User описывает структуру пользователя, как в таблице users в Supabase.
type User struct {
	ID                   string    `json:"id"`
	Name                 *string   `json:"name,omitempty"`
	Email                *string   `json:"email,omitempty"`
	PasswordHash         string    `json:"-"` // не отдаём наружу
	CloudPasswordHash    string    `json:"-"` // не отдаём наружу
	CloudPasswordEnabled bool      `json:"cloudPasswordEnabled"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}
