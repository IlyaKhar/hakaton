package transactions

import "time"

// Transaction описывает транзакцию из источника (банк, почта).
type Transaction struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	SourceID       *string   `json:"sourceId,omitempty"`
	SubscriptionID *string   `json:"subscriptionId,omitempty"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	ChargedAt      time.Time `json:"chargedAt"`
	RawDescription string    `json:"rawDescription"`
}
