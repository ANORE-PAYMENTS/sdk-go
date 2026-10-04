package anore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func testClient(t *testing.T, server *httptest.Server, opts ...Option) *Client {
	t.Helper()
	options := []Option{WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithMaxRetries(0)}
	options = append(options, opts...)
	c, err := New("test-api-key", options...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func independentSignature(raw []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestBaseURLs(t *testing.T) {
	c, err := New("test-api-key")
	if err != nil || c.baseURL != "https://api.anore.cc/api/v1" {
		t.Fatalf("default base = %v, err = %v", c, err)
	}
	for _, tc := range []struct{ suffix, path string }{
		{"", "/api/v1/payments/payment-1"},
		{"/", "/api/v1/payments/payment-1"},
		{"/api", "/api/v1/payments/payment-1"},
		{"/api/v1/", "/api/v1/payments/payment-1"},
		{"/v1/", "/v1/payments/payment-1"},
		{"/proxy/api/v1", "/proxy/api/v1/payments/payment-1"},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path || r.Header.Get("Authorization") != "Bearer test-api-key" {
					t.Errorf("unexpected request %s, auth %q", r.URL.Path, r.Header.Get("Authorization"))
				}
				io.WriteString(w, `{"success":true,"id":"payment-1","status":"new"}`)
			}))
			defer server.Close()
			client := testClient(t, server, WithBaseURL(server.URL+tc.suffix))
			if _, err := client.GetPayment("payment-1"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreatePaymentContractAndExactSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/payments" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.Header.Get("X-ZPay-Signature") != independentSignature(raw, "api-secret") {
			t.Error("request signature does not match the exact bytes received")
		}
		if r.Header.Get("Anore-Signature") != "" {
			t.Error("webhook signature header sent on API request")
		}
		if r.Header.Get("User-Agent") != "anore-go/1.2.0" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected headers %v", r.Header)
		}
		var body map[string]interface{}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Error(err)
		}
		want := map[string]interface{}{
			"amount": 12.34, "description": "Подписка <Pro> & доступ", "orderId": "order-42", "shopId": float64(7),
			"currency": "usd", "methods": []interface{}{"sbp", "card", "crypto"}, "email": "buyer@example.com",
			"getbackurl": "https://merchant.example/back", "successurl": "https://merchant.example/success",
			"failurl": "https://merchant.example/fail", "callbackUrl": "https://merchant.example/webhook",
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body = %#v, want %#v", body, want)
		}
		io.WriteString(w, `{"success":true,"id":"payment-42","orderId":"order-42","amount":12.34,"baseAmount":1110.6,"rubAmount":1110.6,"currency":"usd","currencyRate":90,"status":"new","paymentUrl":"https://anore.cc/pay/payment-42","sbpUrl":"https://qr.nspk.ru/example","expiresIn":14400,"test":true}`)
	}))
	defer server.Close()
	client := testClient(t, server, WithSecret("api-secret"))
	payment, err := client.CreatePayment(CreatePaymentParams{
		Amount: 12.34, Description: "Подписка <Pro> & доступ", OrderID: "order-42", ShopID: 7,
		Currency: "usd", Methods: []string{"sbp", "card", "crypto"}, Email: "buyer@example.com",
		GetbackURL: "https://merchant.example/back", SuccessURL: "https://merchant.example/success",
		FailURL: "https://merchant.example/fail", CallbackURL: "https://merchant.example/webhook",
	})
	if err != nil {
		t.Fatal(err)
	}
	if payment.Amount != 12.34 || payment.RubAmount != 1110.6 || payment.CurrencyRate != 90 || payment.ExpiresIn != 14400 || !payment.Test || payment.SBPURL == "" {
		t.Fatalf("unexpected payment %#v", payment)
	}
}

func TestListPaymentsPagination(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" || r.URL.Path != "/api/v1/payments" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if requests == 1 {
			if query.Get("shopId") != "7" || query.Get("status") != "written_off" || query.Get("from") != "2026-10-01" || query.Get("to") != "2026-10-03" || query.Get("limit") != "20" || query.Get("offset") != "40" {
				t.Errorf("unexpected filters %v", query)
			}
		} else if query.Get("limit") != "50" || query.Get("offset") != "0" || query.Get("shopId") != "" {
			t.Errorf("unexpected defaults %v", query)
		}
		io.WriteString(w, `{"success":true,"shopId":7,"total":61,"limit":20,"offset":40,"payments":[{"id":"payment-42","orderId":null,"amount":15,"baseAmount":1350,"rubAmount":1350,"currency":"usd","currencyRate":90,"description":"Invoice","status":"written_off","paid":false,"method":null,"createdAt":"2026-10-03T12:00:00Z","paidAt":null}]}`)
	}))
	defer server.Close()
	client := testClient(t, server)
	page, err := client.ListPayments(ListPaymentsParams{ShopID: 7, Status: "written_off", From: "2026-10-01", To: "2026-10-03", Limit: 20, Offset: 40})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 61 || page.Offset != 40 || len(page.Payments) != 1 || page.Payments[0].Status != "written_off" || page.Payments[0].Amount != 15 || page.Payments[0].RubAmount != 1350 {
		t.Fatalf("unexpected page %#v", page)
	}
	if _, err := client.ListPayments(ListPaymentsParams{}); err != nil {
		t.Fatal(err)
	}
}

func TestPayoutEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path != "/api/v1/payouts/WD-42" && r.URL.Query().Get("shopId") != "7" {
			t.Errorf("shopId not propagated: %s", r.URL)
		}
		switch r.URL.Path {
		case "/api/v1/balance":
			io.WriteString(w, `{"currency":"RUB","available":5000,"shopLocalBalance":7000,"shopLocalAvailable":6000,"accountAvailable":5000,"hold":400,"frozen":100,"matured":6500,"paidAmount":7000,"withdrawn":1000,"reserved":500,"legacyPayoutsUnassigned":false,"fee":{"flatRub":240,"thresholdRub":25000,"percentAbove":1,"trc20SurchargeRub":60,"methods":[{"method":"usdt_trc20","flatRub":300,"percentAbove":1,"surchargeRub":60}]},"minAmount":500,"minAmountByMethod":{"card_rub":3000},"maxAmount":1500000,"maxAmountByMethod":{"card_rub":500000},"usdtRateRub":90,"rapiraMarketUsdtRub":88,"rapiraMarkupPercent":2,"cbrRateRub":91,"rateBasis":"fixed_0900_msk","rapiraFixing":"09:00 Europe/Moscow","sbpBanks":[{"id":"sber","name":"Сбербанк"}]}`)
		case "/api/v1/payouts/fees":
			io.WriteString(w, `{"shopId":7,"currency":"RUB","thresholdRub":25000,"minAmountRub":500,"minAmountRubByMethod":{"card_rub":3000},"maxAmountRub":1500000,"maxAmountRubByMethod":{"card_rub":500000},"methods":[{"method":"usdt_trc20","label":"USDT TRC20","flatRub":300,"percentAbove":1,"surchargeRub":60}]}`)
		case "/api/v1/payouts/rates":
			io.WriteString(w, `{"shopId":7,"ratePolicy":"approval","rateBasis":"fixed_0900_msk","sbpRatePolicy":"execution","rapiraUsdtRub":90,"rapiraMarketUsdtRub":88,"rapiraMarkupPercent":2,"cbrUsdRub":91,"rubToUsdt":0.011111,"usdtToRub":91,"rubToRubSettlement":1.011111,"rapiraFixing":"09:00 Europe/Moscow","fees":[{"method":"card_rub","label":"Карта РФ","flatRub":240,"percentAbove":1,"surchargeRub":0}]}`)
		case "/api/v1/payouts":
			if r.Method != "POST" {
				t.Errorf("unexpected payout method %s", r.Method)
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Header.Get("X-ZPay-Signature") != independentSignature(raw, "api-secret") {
				t.Error("payout body signature mismatch")
			}
			var body map[string]interface{}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Error(err)
			}
			want := map[string]interface{}{"amount": float64(5000), "method": "sbp", "address": "+79990000000", "bank": "sber", "shopId": float64(7), "externalId": "payout-42"}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("unexpected payout body %#v", body)
			}
			io.WriteString(w, `{"id":"WD-42","shopId":7,"status":"pending","amount":5000,"method":"sbp","address":"+79990000000","bank":"sber","fee":240,"netRub":4760,"amountUsdt":52.888889,"rapiraRate":90,"cbrRate":91,"settlementRub":4812.89,"quoteType":"estimate","ratePolicy":"execution","rateBasis":"fixed_0900_msk","externalId":"payout-42"}`)
		case "/api/v1/payouts/WD-42":
			io.WriteString(w, `{"id":"WD-42","shopId":7,"legacy":false,"status":"paid","manualCorrection":true,"statusRevision":2,"amount":5000,"method":"СБП","methodCode":"sbp","address":"+79990000000","bank":"sber","fee":240,"netRub":4760,"amountUsdt":52.888889,"rapiraRate":90,"cbrRate":91,"settlementRub":4812.89,"quoteType":"executed","ratePolicy":"fixed","quoteAppliedAt":"2026-10-03T12:00:00Z","externalId":"payout-42","createdAt":"2026-10-03T10:00:00Z","processedAt":"2026-10-03T12:05:00Z","txHash":"tx-42"}`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := testClient(t, server, WithSecret("api-secret"))
	balance, err := client.GetBalance(7)
	if err != nil || balance.Available != 5000 || balance.MinAmountByMethod["card_rub"] != 3000 || balance.RateBasis != "fixed_0900_msk" {
		t.Fatalf("balance = %#v, err = %v", balance, err)
	}
	fees, err := client.GetPayoutFees(7)
	if err != nil || len(fees.Methods) != 1 || fees.Methods[0].SurchargeRub != 60 || fees.MaxAmountRubByMethod["card_rub"] != 500000 {
		t.Fatalf("fees = %#v, err = %v", fees, err)
	}
	rates, err := client.GetPayoutRates(7)
	if err != nil || len(rates.Fees) != 1 || rates.Fees[0].Method != "card_rub" || rates.SBPRatePolicy != "execution" || rates.RateBasis != "fixed_0900_msk" {
		t.Fatalf("rates = %#v, err = %v", rates, err)
	}
	payout, err := client.CreatePayout(CreatePayoutParams{Amount: 5000, Method: "sbp", Address: "+79990000000", ShopID: 7, Bank: "sber", ExternalID: "payout-42"})
	if err != nil || payout.ID != "WD-42" || payout.NetRub != 4760 || payout.RateBasis != "fixed_0900_msk" {
		t.Fatalf("payout = %#v, err = %v", payout, err)
	}
	payout, err = client.GetPayout("WD-42")
	if err != nil || !payout.ManualCorrection || payout.StatusRevision != 2 || payout.QuoteAppliedAt == "" || payout.MethodCode != "sbp" || payout.TxHash != "tx-42" {
		t.Fatalf("payout status = %#v, err = %v", payout, err)
	}
}

func TestGETRetries(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			http.Error(w, `{"message":"temporarily unavailable"}`, 503)
			return
		}
		io.WriteString(w, `{"success":true,"id":"payment-1","status":"paid","paid":true}`)
	}))
	defer server.Close()
	client := testClient(t, server, WithMaxRetries(1))
	payment, err := client.GetPayment("payment-1")
	if err != nil || !payment.Paid || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("payment = %#v, err = %v, calls = %d", payment, err, calls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPOSTNeverRetries(t *testing.T) {
	for _, operation := range []string{"payment", "payout"} {
		for _, failure := range []string{"server", "network"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				calls := 0
				client, err := New("test-api-key", WithMaxRetries(3), WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if failure == "network" {
							return nil, errors.New("connection lost after request write")
						}
						return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"message":"temporary failure"}`))}, nil
					}),
				}))
				if err != nil {
					t.Fatal(err)
				}
				if operation == "payment" {
					_, err = client.CreatePayment(CreatePaymentParams{Amount: 100, Description: "Order"})
				} else {
					_, err = client.CreatePayout(CreatePayoutParams{Amount: 5000, Method: "usdt_ton", Address: "UQ-address", ExternalID: "reference"})
				}
				if err == nil || calls != 1 {
					t.Fatalf("err = %v, calls = %d", err, calls)
				}
			})
		}
	}
}

func TestAPIErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		kind   string
	}{{400, "validation"}, {401, "authentication"}, {403, "forbidden"}, {404, "not_found"}, {429, "api"}, {503, "server"}} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", "request-42")
				w.WriteHeader(tc.status)
				io.WriteString(w, `{"message":"API отказал"}`)
			}))
			defer server.Close()
			_, err := testClient(t, server).GetPayment("payment-1")
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Status != tc.status || apiErr.Kind != tc.kind || apiErr.Message != "API отказал" || apiErr.RequestID != "request-42" {
				t.Fatalf("unexpected error %#v", err)
			}
		})
	}
}

func TestInvalidOptions(t *testing.T) {
	for _, base := range []string{"", "not-a-url", "ftp://example.com", "https://user:password@example.com", "https://example.com?key=secret", "https://example.com#fragment"} {
		if _, err := New("key", WithBaseURL(base)); err == nil {
			t.Errorf("accepted invalid base URL %q", base)
		}
	}
	if _, err := New(" "); err == nil {
		t.Error("accepted empty API key")
	}
	if _, err := New("key", WithMaxRetries(-1)); err == nil {
		t.Error("accepted negative retry count")
	}
	if _, err := New("key", WithHTTPClient(nil)); err == nil {
		t.Error("accepted nil HTTP client")
	}
}

func TestInvalidFinancialInputs(t *testing.T) {
	client, _ := New("key", WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Error("invalid input reached transport")
		return nil, errors.New("unexpected request")
	})}))
	for _, amount := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := client.CreatePayment(CreatePaymentParams{Amount: amount, Description: "Order"}); err == nil {
			t.Errorf("accepted payment amount %v", amount)
		}
		if _, err := client.CreatePayout(CreatePayoutParams{Amount: amount, Method: "card", Address: "card-number"}); err == nil {
			t.Errorf("accepted payout amount %v", amount)
		}
	}
	if _, err := client.CreatePayment(CreatePaymentParams{Amount: 1, Description: " \t"}); err == nil {
		t.Error("accepted blank description")
	}
}

func TestInvalidSuccessfulResponse(t *testing.T) {
	for _, body := range []string{"", "null", "{", "[]"} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, body)
			}))
			defer server.Close()
			if _, err := testClient(t, server).GetPayment("payment-1"); err == nil {
				t.Errorf("accepted invalid API response %q", body)
			}
		})
	}
}

type failedBody struct{}

func (failedBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failedBody) Close() error             { return nil }

func TestPOSTResponseReadFailure(t *testing.T) {
	calls := 0
	client, _ := New("key", WithMaxRetries(3), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: failedBody{}}, nil
	})}))
	_, err := client.CreatePayment(CreatePaymentParams{Amount: 100, Description: "Order"})
	var connectionErr *ConnectionError
	if !errors.As(err, &connectionErr) || !errors.Is(err, io.ErrUnexpectedEOF) || calls != 1 {
		t.Fatalf("err = %v, calls = %d", err, calls)
	}
}
