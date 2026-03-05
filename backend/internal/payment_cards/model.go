package payment_cards

import "time"

// PaymentCard описывает платежную карту пользователя.
type PaymentCard struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	LastFourDigits string    `json:"lastFourDigits"`       // последние 4 цифры
	CardMask       string    `json:"cardMask"`             // маска карты (**** **** **** 1234)
	CardType       string    `json:"cardType"`             // Visa, Mastercard, Mir и т.д.
	ExpiryMonth    int       `json:"expiryMonth"`          // месяц (1-12)
	ExpiryYear     int       `json:"expiryYear"`           // год (2024, 2025 и т.д.)
	HolderName     *string   `json:"holderName,omitempty"` // имя держателя
	IsDefault      bool      `json:"isDefault"`            // карта по умолчанию
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
