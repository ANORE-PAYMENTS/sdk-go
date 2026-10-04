package anore

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestWebhookSignatureExactBytes(t *testing.T) {
	raw := []byte("{\n  \"event\":\"payment.succeeded\",\"id\":\"payment-42\",\"description\":\"Подписка\"\n}")
	signature := independentSignature(raw, "webhook-secret")
	if !VerifyWebhook(raw, " "+strings.ToUpper(signature)+" ", "webhook-secret") {
		t.Error("valid signature rejected")
	}
	for _, tc := range []struct {
		body              []byte
		signature, secret string
	}{
		{bytes.ReplaceAll(raw, []byte("\n"), nil), signature, "webhook-secret"},
		{raw, signature, "other-secret"},
		{raw, "invalid", "webhook-secret"},
		{raw, signature, ""},
		{nil, signature, "webhook-secret"},
	} {
		if VerifyWebhook(tc.body, tc.signature, tc.secret) {
			t.Error("invalid signature accepted")
		}
		_, err := ParseWebhook(tc.body, tc.signature, tc.secret)
		var signatureErr *SignatureError
		if !errors.As(err, &signatureErr) {
			t.Errorf("expected SignatureError, got %v", err)
		}
	}
}

func TestWebhookContractsAndRoundTrip(t *testing.T) {
	for _, event := range []string{"payment.succeeded", "payment.expired", "payment.test", "payout.created", "payout.processing", "payout.succeeded", "payout.failed", "payout.updated"} {
		t.Run(event, func(t *testing.T) {
			payout := strings.HasPrefix(event, "payout.")
			fields := map[string]interface{}{
				"event": event, "id": "payment-42", "orderId": "order-42", "amount": 10.5,
				"rubAmount": 945.0, "currency": "usd", "currencyRate": 90.0, "status": "paid",
				"description": "Подписка", "shop": "Store", "createdAt": "2026-10-03T12:00:00Z",
			}
			if payout {
				fields = map[string]interface{}{
					"event": event, "id": "WD-42", "externalId": "external-42", "shopId": float64(7),
					"status": "paid", "amount": 5000.0, "currency": "RUB", "method": "usdt_ton",
					"fee": 240.0, "netRub": 4760.0, "amountUsdt": 52.888889, "settlementRub": 4812.89,
					"txHash": "tx-42", "createdAt": "2026-10-03T10:00:00Z", "processedAt": "2026-10-03T12:00:00Z",
					"manualCorrection": true, "statusRevision": float64(2),
				}
			}
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			model, err := ParseWebhook(raw, independentSignature(raw, "secret"), "secret")
			if err != nil {
				t.Fatal(err)
			}
			if model.IsPayout() != payout || model.IsSucceeded() != (event == "payment.succeeded" || event == "payout.succeeded") {
				t.Errorf("wrong event classification: %#v", model)
			}
			roundTrip, err := json.Marshal(model)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(roundTrip, &decoded); err != nil {
				t.Fatal(err)
			}
			for key, want := range fields {
				if decoded[key] != want {
					t.Errorf("field %s lost or changed: got %v, want %v", key, decoded[key], want)
				}
			}
		})
	}
}

func TestInvalidSignedWebhookPayload(t *testing.T) {
	for _, raw := range []string{"null", "[]", "{", "{}", `{"event":"payment.succeeded"}`, `{"id":"payment-42"}`} {
		if _, err := ParseWebhook([]byte(raw), independentSignature([]byte(raw), "secret"), "secret"); err == nil {
			t.Errorf("accepted invalid signed webhook %s", raw)
		}
	}
}
