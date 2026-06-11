package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/usecase"
)

type NotificationPayload struct {
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

type PaymentConsumer interface {
	ConsumePaymentConfirmations() (<-chan rabbitmq.Delivery, error)
}

type NotificationConsumer interface {
	ConsumeMerchantNotifications() (<-chan rabbitmq.Delivery, error)
}

type MerchantNotifier interface {
	SendToMerchant(merchantID string, notification interface{}) error
}

// StartPaymentConsumer runs a background goroutine to process async payment confirmations.
func StartPaymentConsumer(consumer PaymentConsumer, txUsecase usecase.PaymentWorkerUsecase) {
	msgs, err := consumer.ConsumePaymentConfirmations()
	if err != nil {
		log.Fatalf("Failed to register RabbitMQ consumer: %v", err)
	}

	go func() {
		for d := range msgs {
			var event map[string]string
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("[Worker] Error unmarshalling message: %v | Body: %s", err, string(d.Body))
				continue
			}

			transactionID := event["transaction_id"]
			if transactionID == "" {
				log.Printf("[Worker] Skipping message with empty transaction_id")
				continue
			}

			start := time.Now()
			if _, err := txUsecase.ConfirmPayment(transactionID); err != nil {
				log.Printf("[Worker] Failed to update transaction %s: %v", transactionID, err)
				continue
			}

			log.Printf("[Worker] Confirmed payment %s in %v", transactionID, time.Since(start))
		}
	}()

	fmt.Println("RabbitMQ Worker is running and waiting for messages...")
}

func StartNotificationConsumer(consumer NotificationConsumer, notifier MerchantNotifier) {
	go func() {
		msgs, err := consumer.ConsumeMerchantNotifications()
		if err != nil {
			log.Fatalf("Failed to register RabbitMQ notification consumer: %v", err)
		}

		log.Println("RabbitMQ notification consumer is running and waiting for messages...")

		for msg := range msgs {
			var payload NotificationPayload
			err := json.Unmarshal(msg.Body, &payload)
			if err != nil {
				log.Printf("[NotificationWorker] Failed to unmarshal message: %v", err)
				if nackErr := msg.Nack(false); nackErr != nil {
					log.Printf("[NotificationWorker] Failed to nack message: %v", nackErr)
				}
				continue
			}

			if notifier != nil {
				notification := map[string]interface{}{
					"type":           "transaction_notification",
					"transaction_id": payload.TransactionID,
					"merchant_name":  payload.MerchantName,
					"merchant_id":    payload.MerchantID,
					"amount":         payload.Amount,
					"status":         payload.Status,
					"timestamp":      payload.Timestamp,
				}

				err := notifier.SendToMerchant(payload.MerchantID, notification)
				if err != nil {
					log.Printf("[NotificationWorker] Failed to send via WebSocket: %v", err)
				}
			} else {
				log.Println("[NotificationWorker] WebSocket hub not initialized")
			}

			if ackErr := msg.Ack(); ackErr != nil {
				log.Printf("[NotificationWorker] Failed to ack message: %v", ackErr)
			}
		}
	}()
}
