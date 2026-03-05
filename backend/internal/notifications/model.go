package notifications

import "time"

// NotificationSetting описывает настройки одного типа уведомлений.
type NotificationSetting struct {
	ID       string         `json:"id"`
	UserID   string         `json:"userId"`
	Type     string         `json:"type"`
	Channels map[string]any `json:"channels"`
	Enabled  bool           `json:"enabled"`
}

// Notification описывает одно уведомление.
type Notification struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	SubscriptionID *string    `json:"subscriptionId,omitempty"`
	Type           string     `json:"type"`
	Payload        map[string]any `json:"payload"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

