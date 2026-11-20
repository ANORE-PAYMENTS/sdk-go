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
		UserIdentity: &anore.UserIdentity{
			Type: anore.UserIdentityTelegramID, Value: "123456789", TelegramUsername: "@customer_name",
		},
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

## Identity и франшиза

`UserIdentity` сейчас может быть необязательным для конкретной кассы, но скоро станет обязательным. Передавайте его уже сейчас. Telegram ID сопоставляется глобально; `email`, `username` и `external_id` — только внутри исходной кассы. `TelegramUsername` разрешён только с `telegram_id` и не участвует в matching.

```go
_, err := c.CreatePayment(anore.CreatePaymentParams{
	Amount: 1500, Description: "Заказ филиала",
	UserIdentity: &anore.UserIdentity{Type: anore.UserIdentityEmail, Value: "buyer@example.com"},
	Franchise: &anore.Franchise{
		Type: anore.FranchiseWebsite, Address: "https://branch.example.com",
		OwnerTelegramID: "987654321", OwnerTelegramUsername: "branch_owner",
	},
})
```

`Franchise` обязателен при включённом franchise mode и запрещён при выключенном. Оба объекта принадлежат только ANORE, не отправляются Payso и не возвращаются через публичные responses, webhooks, логи или exports.

## Рекуррентные платежи

```go
catalog, _ := c.GetSubscriptionCatalog()
binding, err := c.CreateSubscription(anore.CreateSubscriptionParams{
	PlanID: "plan_id", CustomerID: "customer-42", CustomerEmail: "buyer@example.com",
	IdempotencyKey: "binding-customer-42", ShopID: 1,
})
_ = catalog
_ = binding
```

Subscription POST никогда не повторяется SDK автоматически. При `503 binding_reconciling` извлеките `subscriptionId` из `APIError.Data` и используйте `GetSubscription(id)`; сохраните тот же key/body и не отправляйте новый POST.

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
| `CreatePayment(CreatePaymentParams)` | создать счёт → `*Payment`; identity скоро станет обязательным, franchise зависит от режима кассы |
| `GetPayment(id)` | статус → `*Payment` (`.Status`, `.Paid`) |
| `ListPayments(ListPaymentsParams)` | страница платежей → `*PaymentList` |
| `GetSubscriptionCatalog()` | доступные кассы и планы → `*SubscriptionCatalog` |
| `CreateSubscription(CreateSubscriptionParams)` | QR привязки → `*SubscriptionBinding` |
| `ChargeSubscription(id, idempotencyKey)` | начать списание → `*SubscriptionCharge` |
| `GetSubscription(id)` | безопасный статус → `*SubscriptionStatus` |
| `GetBalance(shopID)` | баланс → `*Balance` |
| `GetPayoutFees(shopID)` | комиссии → `*PayoutFees` |
| `GetPayoutRates(shopID)` | курсы → `*PayoutRates` |
| `ListPayouts(ListPayoutsParams)` | история выплат → `*PayoutList` |
| `CreatePayout(CreatePayoutParams)` | заявка → `*Payout` |
| `GetPayout(id)` | статус выплаты → `*Payout` |
| `VerifyWebhook(rawBody, signature, secret)` | проверка подписи → `bool` |
| `ParseWebhook(rawBody, signature, secret)` | проверка + разбор → `*WebhookEvent` (возвращает `*SignatureError`) |

Ошибки: `*APIError` (с `.Status`, `.Kind` и предикатами `IsValidation/IsAuthentication/IsForbidden/IsNotFound/IsServer`), `*ConnectionError` (сеть), `*SignatureError` (подпись).

Полная документация: https://anore.cc/docs
