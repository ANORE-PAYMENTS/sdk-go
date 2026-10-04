# anore — Go SDK

Официальный SDK для приёма платежей через [anore](https://anore.cc). Без зависимостей, только stdlib, Go 1.18+.

## Установка

```bash
go get github.com/anore-payments/sdk-go
```

```go
import anore "github.com/anore-payments/sdk-go"
```

## Быстрый старт

```go
package main

import (
	"fmt"
	"log"

	anore "github.com/anore-payments/sdk-go"
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

	page, _ := c.ListPayments(anore.ListPaymentsParams{ShopID: 1, Status: "paid", Limit: 20})
	balance, _ := c.GetBalance(1)
	fmt.Println(page.Total, balance.Available)

	payout, _ := c.CreatePayout(anore.CreatePayoutParams{
		Amount: 5000, Method: "usdt_ton", Address: "UQ...", ShopID: 1, ExternalID: "payout_42",
	})
	fmt.Println(payout.ID, payout.Status)
}
```

## Проверка вебхука

При оплате anore шлёт `POST` на ваш URL с заголовком `Anore-Signature`.
Проверяйте подпись по **сырому** телу запроса (не декодированному JSON):

Тот же обработчик принимает `payout.created`, `payout.processing`, `payout.succeeded`, `payout.failed` и `payout.updated` (ручная корректировка статуса, см. `statusRevision`); используйте `ev.IsPayout()`.

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
| `ListPayments(ListPaymentsParams)` | страница платежей → `*PaymentList` |
| `GetBalance(shopID)` | баланс → `*Balance` |
| `GetPayoutFees(shopID)` | комиссии → `*PayoutFees` |
| `GetPayoutRates(shopID)` | курсы → `*PayoutRates` |
| `CreatePayout(CreatePayoutParams)` | заявка → `*Payout` |
| `GetPayout(id)` | статус выплаты → `*Payout` |
| `VerifyWebhook(rawBody, signature, secret)` | проверка подписи → `bool` |
| `ParseWebhook(rawBody, signature, secret)` | проверка + разбор → `*WebhookEvent` (возвращает `*SignatureError`) |

Ошибки: `*APIError` (с `.Status`, `.Kind` и предикатами `IsValidation/IsAuthentication/IsForbidden/IsNotFound/IsServer`), `*ConnectionError` (сеть), `*SignatureError` (подпись).

Полная документация: https://anore.cc/docs
