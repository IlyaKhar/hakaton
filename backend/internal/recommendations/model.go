package recommendations

// RecommendationAlternative описывает альтернативный сервис внутри категории.
type RecommendationAlternative struct {
	ID           string  `json:"id"`
	Category     string  `json:"category"`
	ServiceName  string  `json:"serviceName"`
	Price        float64 `json:"price"`
	BillingPeriod string `json:"billingPeriod"`
	Description  string  `json:"description"`
}

