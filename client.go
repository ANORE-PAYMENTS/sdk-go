package anore

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is the production API base.
const DefaultBaseURL = "https://api.anore.cc/v1"

const userAgent = "anore-go/1.0.0"

// Client is the anore payments API client. Create it with New.
//
//	c, _ := anore.New("an_live_xxxxxxxxxxxxxxxx")
//	p, err := c.CreatePayment(anore.CreatePaymentParams{
//	    Amount: 1500, Description: "Подписка Pro", OrderID: "order_42", ShopID: 1,
//	})
//	if err != nil { /* handle */ }
//	fmt.Println(p.PaymentURL)
type Client struct {
	apiKey     string
	secret     string
	baseURL    string
	maxRetries int
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithSecret signs outgoing requests with an Anore-Signature header (the server
// verifies it if present).
func WithSecret(secret string) Option { return func(c *Client) { c.secret = secret } }

// WithBaseURL overrides the API base (default DefaultBaseURL).
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.baseURL = baseURL } }

// WithMaxRetries sets retries on network errors / 5xx (default 2).
func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

// WithHTTPClient supplies a custom *http.Client (e.g. with a proxy or timeout).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// New creates a Client. apiKey is the key from the dashboard (an_live_… / an_test_…).
func New(apiKey string, opts ...Option) (*Client, error) {
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
	return c, nil
}

// CreatePaymentParams are the inputs to CreatePayment.
type CreatePaymentParams struct {
	Amount      float64 // in rubles, must be > 0
	Description string  // shown to the customer
	OrderID     string  // your own order reference (optional)
	ShopID      int64   // required for account-level keys (an_*); omit (0) to skip
}

// CreatePayment creates a payment / invoice (POST /payments). Use Payment.PaymentURL
// to redirect the customer.
func (c *Client) CreatePayment(p CreatePaymentParams) (*Payment, error) {
	if !(p.Amount > 0) {
		return nil, fmt.Errorf("anore: amount must be > 0")
	}
	if p.Description == "" {
		return nil, fmt.Errorf("anore: description is required")
	}
	body := map[string]interface{}{"amount": p.Amount, "description": p.Description}
	if p.OrderID != "" {
		body["orderId"] = p.OrderID
	}
	if p.ShopID != 0 {
		body["shopId"] = p.ShopID
	}
	var out Payment
	if err := c.request("POST", "/payments", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPayment fetches payment status (GET /payments/{id}). Payment.Status is
// "new" | "paid" | "expired".
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
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
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
				req.Header.Set("Anore-Signature", hmacHex(c.secret, payload))
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &ConnectionError{Err: err}
			if attempt < c.maxRetries {
				time.Sleep(backoff)
				backoff = capDur(backoff*2, 4*time.Second)
				continue
			}
			return lastErr
		}

		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil && len(data) > 0 {
				if err := json.Unmarshal(data, out); err != nil {
					return fmt.Errorf("anore: decode response: %w", err)
				}
			}
			return nil
		}

		if resp.StatusCode >= 500 && attempt < c.maxRetries {
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
