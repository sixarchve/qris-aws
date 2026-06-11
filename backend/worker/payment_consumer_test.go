package worker

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/repository/rabbitmq"
)

type fakePaymentConsumer struct {
	deliveries chan rabbitmq.Delivery
}

func (c *fakePaymentConsumer) ConsumePaymentConfirmations() (<-chan rabbitmq.Delivery, error) {
	return c.deliveries, nil
}

type fakeNotificationConsumer struct {
	deliveries chan rabbitmq.Delivery
}

func (c *fakeNotificationConsumer) ConsumeMerchantNotifications() (<-chan rabbitmq.Delivery, error) {
	return c.deliveries, nil
}

type fakeWorkerTransactionUsecase struct {
	confirmed chan string
	err       error
}

func (u *fakeWorkerTransactionUsecase) ConfirmPayment(transactionIDStr string) (*entity.TransactionResponse, error) {
	u.confirmed <- transactionIDStr
	return nil, u.err
}

type fakeNotifier struct {
	merchantIDs chan string
	err         error
}

func (n *fakeNotifier) SendToMerchant(merchantID string, notification interface{}) error {
	n.merchantIDs <- merchantID
	return n.err
}

func TestStartPaymentConsumerConfirmsTransaction(t *testing.T) {
	consumer := &fakePaymentConsumer{deliveries: make(chan rabbitmq.Delivery, 1)}
	txUsecase := &fakeWorkerTransactionUsecase{confirmed: make(chan string, 1)}

	StartPaymentConsumer(consumer, txUsecase)
	consumer.deliveries <- rabbitmq.Delivery{Body: []byte(`{"transaction_id":"tx-123"}`)}

	select {
	case txID := <-txUsecase.confirmed:
		if txID != "tx-123" {
			t.Fatalf("expected tx-123, got %s", txID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected transaction confirmation")
	}
}

func TestStartNotificationConsumerAcksValidMessageAndNotifies(t *testing.T) {
	consumer := &fakeNotificationConsumer{deliveries: make(chan rabbitmq.Delivery, 1)}
	notifier := &fakeNotifier{merchantIDs: make(chan string, 1)}
	acked := make(chan struct{}, 1)

	StartNotificationConsumer(consumer, notifier)
	body, err := json.Marshal(NotificationPayload{
		TransactionID: "tx-123",
		MerchantID:    "merchant-123",
		MerchantName:  "Kantin",
		Amount:        15000,
		Status:        "SUCCESS",
		Timestamp:     time.Now(),
	})
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}
	consumer.deliveries <- rabbitmq.Delivery{
		Body: body,
		Ack: func() error {
			acked <- struct{}{}
			return nil
		},
		Nack: func(requeue bool) error {
			return errors.New("unexpected nack")
		},
	}

	select {
	case merchantID := <-notifier.merchantIDs:
		if merchantID != "merchant-123" {
			t.Fatalf("expected merchant-123, got %s", merchantID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected notification")
	}

	select {
	case <-acked:
	case <-time.After(time.Second):
		t.Fatal("expected ack")
	}
}

func TestStartNotificationConsumerNacksMalformedMessage(t *testing.T) {
	consumer := &fakeNotificationConsumer{deliveries: make(chan rabbitmq.Delivery, 1)}
	notifier := &fakeNotifier{merchantIDs: make(chan string, 1)}
	nacked := make(chan bool, 1)

	StartNotificationConsumer(consumer, notifier)
	consumer.deliveries <- rabbitmq.Delivery{
		Body: []byte(`{bad-json`),
		Ack: func() error {
			return errors.New("unexpected ack")
		},
		Nack: func(requeue bool) error {
			nacked <- requeue
			return nil
		},
	}

	select {
	case requeue := <-nacked:
		if requeue {
			t.Fatal("expected malformed message to be nacked without requeue")
		}
	case <-time.After(time.Second):
		t.Fatal("expected nack")
	}
}
