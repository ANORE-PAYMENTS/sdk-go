package anore

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func VerifyWebhook(rawBody []byte, signature, secret string) bool {
	if len(rawBody) == 0 || signature == "" || secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	got := strings.ToLower(strings.TrimSpace(signature))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}

func ParseWebhook(rawBody []byte, signature, secret string) (*WebhookEvent, error) {
	if !VerifyWebhook(rawBody, signature, secret) {
		return nil, &SignatureError{}
	}
	if !bytes.HasPrefix(bytes.TrimSpace(rawBody), []byte("{")) {
		return nil, fmt.Errorf("anore: webhook payload must be a JSON object")
	}
	var ev WebhookEvent
	if err := json.Unmarshal(rawBody, &ev); err != nil {
		return nil, err
	}
	if ev.Event == "" || ev.ID == "" {
		return nil, fmt.Errorf("anore: webhook event and id are required")
	}
	return &ev, nil
}
