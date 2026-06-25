# anore — Go SDK

Официальный SDK для приёма платежей через [anore](https://anore.cc). Без зависимостей, только stdlib, Go 1.18+.

## Установка

```bash
go get github.com/roditsya/sdk-go
```

```go
import anore "github.com/roditsya/sdk-go"
```

## Быстрый старт

```go
package main

import (
	"fmt"
	"log"

	anore "github.com/roditsya/sdk-go"
)

func main() {
	c, err := anore.New("an_live_xxxxxxxxxxxxxxxx")
	if err != nil {
		log.Fatal(err)
	}

	// 1. создать счёт
	p, err := c.CreatePayment(anore.CreatePaymentParams{
		Amount:      1500,
		Description: "Подписка Pro",
		OrderID:     "order_42",
		ShopID:      1, // обязателен для аккаунтовых ключей (an_*)
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.PaymentURL) // отправьте клиента сюда

	// 2. проверить статус
	s, err := c.GetPayment(p.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.Status, s.Paid) // paid true
}
```

## Проверка вебхука

При оплате anore шлёт `POST` на ваш URL с заголовком `Anore-Signature`.
Проверяйте подпись по **сырому** телу запроса (не декодированному JSON):

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	sig := r.Header.Get("Anore-Signature")

	ev, err := anore.ParseWebhook(raw, sig, os.Getenv("ANORE_WEBHOOK_SECRET"))
	if err != nil {
		w.WriteHeader(http.StatusForbidden) // подпись не сошлась
		return
	}
	if ev.IsSucceeded() {
		// отгрузить заказ ev.ID / ev.OrderID
	}
	w.WriteHeader(http.StatusOK)
}
```

## Обработка ошибок

```go
p, err := c.CreatePayment(params)
if err != nil {
	var apiErr *anore.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.IsValidation():     // 400 — кривой запрос
		case apiErr.IsAuthentication(): // 401 — неверный ключ
		case apiErr.IsServer():         // 5xx
		}
	}
	var connErr *anore.ConnectionError
	if errors.As(err, &connErr) {
		// сеть недоступна
	}
}
```

## Справка

| API | Описание |
|-----|----------|
| `anore.New(apiKey, ...Option)` | клиент; опции `WithSecret`, `WithBaseURL`, `WithMaxRetries`, `WithHTTPClient` |
| `CreatePayment(CreatePaymentParams)` | создать счёт → `*Payment` |
| `GetPayment(id)` | статус → `*Payment` (`.Status`, `.Paid`) |
| `VerifyWebhook(rawBody, signature, secret)` | проверка подписи → `bool` |
| `ParseWebhook(rawBody, signature, secret)` | проверка + разбор → `*WebhookEvent` (возвращает `*SignatureError`) |

Ошибки: `*APIError` (с `.Status`, `.Kind` и предикатами `IsValidation/IsAuthentication/IsForbidden/IsNotFound/IsServer`), `*ConnectionError` (сеть), `*SignatureError` (подпись).

Полная документация: https://anore.cc/docs
