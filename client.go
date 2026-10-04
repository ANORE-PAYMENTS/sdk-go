package anore

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.anore.cc/api/v1"

const userAgent = "anore-go/1.2.0"

type Client struct {
	apiKey     string
	secret     string
	baseURL    string
	maxRetries int
	httpClient *http.Client
}

type Option func(*Client)

func WithSecret(secret string) Option { return func(c *Client) { c.secret = secret } }

func WithBaseURL(baseURL string) Option { return func(c *Client) { c.baseURL = baseURL } }

func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

func New(apiKey string, opts ...Option) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("anore: apiKey is required")
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		maxRetries: 2,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	baseURL, err := normalizeBaseURL(c.baseURL)
	if err != nil {
		return nil, err
	}
	c.baseURL = baseURL
	if c.maxRetries < 0 {
		return nil, fmt.Errorf("anore: maxRetries must be >= 0")
	}
	if c.httpClient == nil {
		return nil, fmt.Errorf("anore: HTTP client is required")
	}
	return c, nil
}

func normalizeBaseURL(baseURL string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return "", fmt.Errorf("anore: baseURL must be an HTTP or HTTPS URL without credentials, query or fragment")
	}
	base.Path = strings.TrimRight(base.Path, "/")
	base.RawPath = strings.TrimRight(base.RawPath, "/")
	if base.Path == "" {
		base.Path = "/api/v1"
	} else if base.Path == "/api" {
		base.Path = "/api/v1"
	}
	return base.String(), nil
}

type CreatePaymentParams struct {
	Amount      float64
	Description string
	OrderID     string
	ShopID      int64
	Currency    string
	Methods     []string
	GetbackURL  string
	SuccessURL  string
	FailURL     string
	CallbackURL string
	Email       string
}

func (c *Client) CreatePayment(p CreatePaymentParams) (*Payment, error) {
	if !(p.Amount > 0) || math.IsInf(p.Amount, 0) {
		return nil, fmt.Errorf("anore: amount must be > 0")
	}
	if strings.TrimSpace(p.Description) == "" {
		return nil, fmt.Errorf("anore: description is required")
	}
	body := map[string]interface{}{"amount": p.Amount, "description": p.Description}
	if p.OrderID != "" {
		body["orderId"] = p.OrderID
	}
	if p.ShopID != 0 {
		body["shopId"] = p.ShopID
	}
	if p.Currency != "" {
		body["currency"] = p.Currency
	}
	if len(p.Methods) > 0 {
		body["methods"] = p.Methods
	}
	if p.GetbackURL != "" {
		body["getbackurl"] = p.GetbackURL
	}
	if p.SuccessURL != "" {
		body["successurl"] = p.SuccessURL
	}
	if p.FailURL != "" {
		body["failurl"] = p.FailURL
	}
	if p.CallbackURL != "" {
		body["callbackUrl"] = p.CallbackURL
	}
	if p.Email != "" {
		body["email"] = p.Email
	}
	var out Payment
	if err := c.request("POST", "/payments", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPayment(id string) (*Payment, error) {
	if id == "" {
		return nil, fmt.Errorf("anore: id is required")
	}
	var out Payment
	if err := c.request("GET", "/payments/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type ListPaymentsParams struct {
	ShopID int64
	Status string
	From   string
	To     string
	Limit  int
	Offset int
}

func (c *Client) ListPayments(p ListPaymentsParams) (*PaymentList, error) {
	query := url.Values{}
	if p.ShopID != 0 {
		query.Set("shopId", fmt.Sprint(p.ShopID))
	}
	if p.Status != "" {
		query.Set("status", p.Status)
	}
	if p.From != "" {
		query.Set("from", p.From)
	}
	if p.To != "" {
		query.Set("to", p.To)
	}
	if p.Limit == 0 {
		p.Limit = 50
	}
	query.Set("limit", fmt.Sprint(p.Limit))
	query.Set("offset", fmt.Sprint(p.Offset))
	var out PaymentList
	if err := c.request("GET", "/payments?"+query.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBalance(shopID int64) (*Balance, error) {
	var out Balance
	if err := c.request("GET", shopPath("/balance", shopID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPayoutFees(shopID int64) (*PayoutFees, error) {
	var out PayoutFees
	if err := c.request("GET", shopPath("/payouts/fees", shopID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPayoutRates(shopID int64) (*PayoutRates, error) {
	var out PayoutRates
	if err := c.request("GET", shopPath("/payouts/rates", shopID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreatePayoutParams struct {
	Amount     float64
	Method     string
	Address    string
	ShopID     int64
	Bank       string
	ExternalID string
}

func (c *Client) CreatePayout(p CreatePayoutParams) (*Payout, error) {
	if !(p.Amount > 0) || math.IsInf(p.Amount, 0) {
		return nil, fmt.Errorf("anore: payout amount must be > 0")
	}
	if strings.TrimSpace(p.Method) == "" {
		return nil, fmt.Errorf("anore: payout method is required")
	}
	if strings.TrimSpace(p.Address) == "" {
		return nil, fmt.Errorf("anore: payout address is required")
	}
	body := map[string]interface{}{
		"amount": p.Amount, "method": p.Method, "address": p.Address,
	}
	if p.ShopID != 0 {
		body["shopId"] = p.ShopID
	}
	if p.Bank != "" {
		body["bank"] = p.Bank
	}
	if p.ExternalID != "" {
		body["externalId"] = p.ExternalID
	}
	var out Payout
	if err := c.request("POST", "/payouts", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPayout(id string) (*Payout, error) {
	if id == "" {
		return nil, fmt.Errorf("anore: payout id is required")
	}
	var out Payout
	if err := c.request("GET", "/payouts/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func shopPath(path string, shopID int64) string {
	if shopID == 0 {
		return path
	}
	return path + "?shopId=" + url.QueryEscape(fmt.Sprint(shopID))
}

func (c *Client) request(method, path string, body interface{}, out interface{}) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("anore: marshal request: %w", err)
		}
	}

	backoff := 500 * time.Millisecond
	retries := 0
	if method == http.MethodGet {
		retries = c.maxRetries
	}
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		var reqBody io.Reader
		if payload != nil {
			reqBody = bytes.NewReader(payload)
		}
		req, err := http.NewRequest(method, c.baseURL+path, reqBody)
		if err != nil {
			return fmt.Errorf("anore: build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
			if c.secret != "" {
				signature := hmacHex(c.secret, payload)
				req.Header.Set("X-ZPay-Signature", signature)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &ConnectionError{Err: err}
			if attempt < retries {
				time.Sleep(backoff)
				backoff = capDur(backoff*2, 4*time.Second)
				continue
			}
			return lastErr
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = &ConnectionError{Err: readErr}
			if attempt < retries {
				time.Sleep(backoff)
				backoff = capDur(backoff*2, 4*time.Second)
				continue
			}
			return lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil {
				if len(bytes.TrimSpace(data)) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
					return fmt.Errorf("anore: API returned an empty response")
				}
				if err := json.Unmarshal(data, out); err != nil {
					return fmt.Errorf("anore: decode response: %w", err)
				}
			}
			return nil
		}

		if resp.StatusCode >= 500 && attempt < retries {
			time.Sleep(backoff)
			backoff = capDur(backoff*2, 4*time.Second)
			continue
		}

		msg := errorMessage(data)
		return apiErrorForStatus(resp.StatusCode, msg, resp.Header.Get("X-Request-Id"))
	}

	if lastErr != nil {
		return lastErr
	}
	return &ConnectionError{Err: fmt.Errorf("request failed")}
}

func errorMessage(data []byte) string {
	var m map[string]interface{}
	if json.Unmarshal(data, &m) == nil {
		if s, ok := m["message"].(string); ok && s != "" {
			return s
		}
		if s, ok := m["error"].(string); ok && s != "" {
			return s
		}
	}
	return "request failed"
}

func hmacHex(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func capDur(d, max time.Duration) time.Duration {
	if d > max {
		return max
	}
	return d
}
