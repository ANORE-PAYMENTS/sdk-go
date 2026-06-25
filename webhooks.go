package anore

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// VerifyWebhook reports whether signature matches the HMAC-SHA256 of rawBody
// keyed by secret. Pass the RAW request body bytes (never the re-serialized JSON).
//
//	ok := anore.VerifyWebhook(rawBody, r.Header.Get("Anore-Signature"), secret)
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

// ParseWebhook verifies the signature and returns the decoded event. It returns a
// *SignatureError if the signature does not match, or a JSON error if the body is
// malformed.
func ParseWebhook(rawBody []byte, signature, secret string) (*WebhookEvent, error) {
	if !VerifyWebhook(rawBody, signature, secret) {
		return nil, &SignatureError{}
	}
	var ev WebhookEvent
	if err := json.Unmarshal(rawBody, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
