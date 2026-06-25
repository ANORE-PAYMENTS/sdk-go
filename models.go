package anore

// Payment is a payment / invoice — the response of CreatePayment and GetPayment.
type Payment struct {
	Success    bool    `json:"success"`
	ID         string  `json:"id"`
	OrderID    string  `json:"orderId,omitempty"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Status     string  `json:"status"` // "new" | "paid" | "expired"
	Paid       bool    `json:"paid"`
	PaymentURL string  `json:"paymentUrl,omitempty"`
	ExpiresIn  int64   `json:"expiresIn,omitempty"`
}

// WebhookEvent is a parsed (and verified) webhook payload.
type WebhookEvent struct {
	Event       string  `json:"event"` // e.g. "payment.succeeded"
	ID          string  `json:"id"`
	OrderID     string  `json:"orderId,omitempty"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	Description string  `json:"description,omitempty"`
	Shop        string  `json:"shop,omitempty"`
	CreatedAt   string  `json:"createdAt,omitempty"`
}

// IsSucceeded reports whether this is a successful-payment event.
func (e *WebhookEvent) IsSucceeded() bool { return e.Event == "payment.succeeded" }
