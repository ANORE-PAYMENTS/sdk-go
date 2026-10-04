package anore

type Payment struct {
	Success      bool    `json:"success"`
	ID           string  `json:"id"`
	OrderID      string  `json:"orderId,omitempty"`
	Amount       float64 `json:"amount"`
	BaseAmount   float64 `json:"baseAmount,omitempty"`
	Currency     string  `json:"currency"`
	CurrencyRate float64 `json:"currencyRate,omitempty"`
	RubAmount    float64 `json:"rubAmount,omitempty"`
	Description  string  `json:"description,omitempty"`
	Status       string  `json:"status"`
	Paid         bool    `json:"paid"`
	Test         bool    `json:"test,omitempty"`
	PaymentURL   string  `json:"paymentUrl,omitempty"`
	SBPURL       string  `json:"sbpUrl,omitempty"`
	ExpiresIn    int64   `json:"expiresIn,omitempty"`
	Method       string  `json:"method,omitempty"`
	CreatedAt    string  `json:"createdAt,omitempty"`
	PaidAt       string  `json:"paidAt,omitempty"`
}

type PaymentList struct {
	Success  bool      `json:"success"`
	ShopID   int64     `json:"shopId"`
	Total    int64     `json:"total"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
	Payments []Payment `json:"payments"`
}

type Balance struct {
	Currency                string                   `json:"currency"`
	Available               float64                  `json:"available"`
	ShopLocalBalance        float64                  `json:"shopLocalBalance"`
	ShopLocalAvailable      float64                  `json:"shopLocalAvailable"`
	AccountAvailable        float64                  `json:"accountAvailable"`
	Hold                    float64                  `json:"hold"`
	Frozen                  float64                  `json:"frozen"`
	Matured                 float64                  `json:"matured"`
	PaidAmount              float64                  `json:"paidAmount"`
	Withdrawn               float64                  `json:"withdrawn"`
	Reserved                float64                  `json:"reserved"`
	LegacyPayoutsUnassigned bool                     `json:"legacyPayoutsUnassigned"`
	Fee                     map[string]interface{}   `json:"fee"`
	MinAmount               float64                  `json:"minAmount"`
	MinAmountByMethod       map[string]float64       `json:"minAmountByMethod"`
	MaxAmount               float64                  `json:"maxAmount"`
	MaxAmountByMethod       map[string]float64       `json:"maxAmountByMethod"`
	UsdtRateRub             float64                  `json:"usdtRateRub"`
	RapiraMarketUsdtRub     float64                  `json:"rapiraMarketUsdtRub"`
	RapiraMarkupPercent     float64                  `json:"rapiraMarkupPercent"`
	CBRRateRub              float64                  `json:"cbrRateRub"`
	RateBasis               string                   `json:"rateBasis"`
	RapiraFixing            string                   `json:"rapiraFixing"`
	SBPBanks                []map[string]interface{} `json:"sbpBanks"`
}

type PayoutFee struct {
	Method       string  `json:"method"`
	Label        string  `json:"label"`
	FlatRub      float64 `json:"flatRub"`
	PercentAbove float64 `json:"percentAbove"`
	SurchargeRub float64 `json:"surchargeRub"`
}

type PayoutFees struct {
	ShopID               int64              `json:"shopId"`
	Currency             string             `json:"currency"`
	ThresholdRub         float64            `json:"thresholdRub"`
	MinAmountRub         float64            `json:"minAmountRub"`
	MinAmountRubByMethod map[string]float64 `json:"minAmountRubByMethod"`
	MaxAmountRub         float64            `json:"maxAmountRub"`
	MaxAmountRubByMethod map[string]float64 `json:"maxAmountRubByMethod"`
	Methods              []PayoutFee        `json:"methods"`
}

type PayoutRates struct {
	ShopID              int64       `json:"shopId"`
	RatePolicy          string      `json:"ratePolicy"`
	RateBasis           string      `json:"rateBasis"`
	SBPRatePolicy       string      `json:"sbpRatePolicy"`
	RapiraUsdtRub       float64     `json:"rapiraUsdtRub"`
	RapiraMarketUsdtRub float64     `json:"rapiraMarketUsdtRub"`
	RapiraMarkupPercent float64     `json:"rapiraMarkupPercent"`
	CBRUsdRub           float64     `json:"cbrUsdRub"`
	RubToUsdt           float64     `json:"rubToUsdt"`
	UsdtToRub           float64     `json:"usdtToRub"`
	RubToRubSettlement  float64     `json:"rubToRubSettlement"`
	RapiraFixing        string      `json:"rapiraFixing"`
	Fees                []PayoutFee `json:"fees"`
}

type Payout struct {
	ID               string  `json:"id"`
	ShopID           int64   `json:"shopId,omitempty"`
	Legacy           bool    `json:"legacy,omitempty"`
	Status           string  `json:"status"`
	Amount           float64 `json:"amount"`
	Method           string  `json:"method,omitempty"`
	MethodCode       string  `json:"methodCode,omitempty"`
	Address          string  `json:"address"`
	Bank             string  `json:"bank,omitempty"`
	Fee              float64 `json:"fee,omitempty"`
	NetRub           float64 `json:"netRub,omitempty"`
	AmountUsdt       float64 `json:"amountUsdt,omitempty"`
	RapiraRate       float64 `json:"rapiraRate,omitempty"`
	CBRRate          float64 `json:"cbrRate,omitempty"`
	SettlementRub    float64 `json:"settlementRub,omitempty"`
	QuoteType        string  `json:"quoteType,omitempty"`
	RatePolicy       string  `json:"ratePolicy,omitempty"`
	RateBasis        string  `json:"rateBasis,omitempty"`
	QuoteAppliedAt   string  `json:"quoteAppliedAt,omitempty"`
	ManualCorrection bool    `json:"manualCorrection,omitempty"`
	StatusRevision   int64   `json:"statusRevision,omitempty"`
	ExternalID       string  `json:"externalId,omitempty"`
	CreatedAt        string  `json:"createdAt,omitempty"`
	ProcessedAt      string  `json:"processedAt,omitempty"`
	TxHash           string  `json:"txHash,omitempty"`
}

type WebhookEvent struct {
	Event            string  `json:"event"`
	ID               string  `json:"id"`
	OrderID          string  `json:"orderId,omitempty"`
	Amount           float64 `json:"amount"`
	RubAmount        float64 `json:"rubAmount,omitempty"`
	Currency         string  `json:"currency"`
	CurrencyRate     float64 `json:"currencyRate,omitempty"`
	Status           string  `json:"status"`
	Description      string  `json:"description,omitempty"`
	Shop             string  `json:"shop,omitempty"`
	CreatedAt        string  `json:"createdAt,omitempty"`
	ExternalID       string  `json:"externalId,omitempty"`
	ShopID           int64   `json:"shopId,omitempty"`
	Method           string  `json:"method,omitempty"`
	Fee              float64 `json:"fee,omitempty"`
	NetRub           float64 `json:"netRub,omitempty"`
	AmountUsdt       float64 `json:"amountUsdt,omitempty"`
	SettlementRub    float64 `json:"settlementRub,omitempty"`
	TxHash           string  `json:"txHash,omitempty"`
	ProcessedAt      string  `json:"processedAt,omitempty"`
	ManualCorrection bool    `json:"manualCorrection,omitempty"`
	StatusRevision   int64   `json:"statusRevision,omitempty"`
}

func (e *WebhookEvent) IsSucceeded() bool {
	return e.Event == "payment.succeeded" || e.Event == "payout.succeeded"
}

func (e *WebhookEvent) IsPayout() bool {
	return len(e.Event) > len("payout.") && e.Event[:len("payout.")] == "payout."
}
