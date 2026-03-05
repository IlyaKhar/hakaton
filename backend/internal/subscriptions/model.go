package subscriptions

import "time"

// Subscription описывает подписку из таблицы subscriptions.
type Subscription struct {
	ID           string     `json:"id"`
	UserID       string     `json:"userId"`
	ServiceName  string     `json:"serviceName"`
	Category     string     `json:"category"`
	Price        float64    `json:"price"`
	Currency     string     `json:"currency"`
	BillingPeriod string    `json:"billingPeriod"`
	NextChargeAt *time.Time `json:"nextChargeAt,omitempty"`
	Status       string     `json:"status"`
	CancelURL    *string    `json:"cancelUrl,omitempty"`
	SupportEmail *string    `json:"supportEmail,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

