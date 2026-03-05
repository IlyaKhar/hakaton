package sources

import "time"

// SubscriptionSource описывает источник данных (банк, почта, ручной ввод).
// @Description Источник данных для автоматического сбора информации о подписках
type SubscriptionSource struct {
	ID        string         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string         `json:"userId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type      string         `json:"type" example:"email"` // "email", "bank", "manual"
	Provider  string         `json:"provider" example:"mailru"` // "mailru", "yandex", "gmail" для email
	Status    string         `json:"status" example:"pending"` // "pending", "connected", "error"
	Meta      map[string]any `json:"meta"` // Для email: {"email": "test@mail.ru"}
	CreatedAt time.Time      `json:"createdAt" example:"2024-03-01T12:00:00Z"`
	UpdatedAt time.Time      `json:"updatedAt" example:"2024-03-01T12:00:00Z"`
}
