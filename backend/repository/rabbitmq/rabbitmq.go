package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"qris-latency-optimizer/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Broker struct {
	conn              *amqp.Connection
	channel           *amqp.Channel
	queue             amqp.Queue
	notificationQueue amqp.Queue
}

type Delivery struct {
	Body []byte
	Ack  func() error
	Nack func(requeue bool) error
}

// ConnectRabbitMQ connects to RabbitMQ with retry logic (3 attempts).
func ConnectRabbitMQ() *Broker {
	url := config.App.RabbitMQURL()
	var conn *amqp.Connection
	var err error

	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("RabbitMQ connection attempt %d/%d failed: %v", attempt, maxRetries, err)
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			log.Printf("Retrying in %v...", backoff)
			time.Sleep(backoff)
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %v", maxRetries, err)
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}

	queue, err := channel.QueueDeclare(
		"payment_confirmations",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	notificationQueue, err := channel.QueueDeclare(
		"merchant_notifications",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare notification queue: %v", err)
	}

	fmt.Println("RabbitMQ connected successfully & queues declared")
	return &Broker{
		conn:              conn,
		channel:           channel,
		queue:             queue,
		notificationQueue: notificationQueue,
	}
}

func (b *Broker) PublishPaymentConfirmation(transactionID string) error {
	event := map[string]string{
		"transaction_id": transactionID,
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return b.channel.PublishWithContext(ctx,
		"",
		b.queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        eventJSON,
		})
}

// NotificationPayload - struktur data untuk merchant notifications.
type NotificationPayload struct {
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

// PublishNotification - publish merchant notification ke queue.
func (b *Broker) PublishNotification(txID, merchantID, merchantName string, amount float64) error {
	if !b.IsConnected() {
		return fmt.Errorf("RabbitMQ not connected")
	}

	payload := NotificationPayload{
		TransactionID: txID,
		MerchantID:    merchantID,
		MerchantName:  merchantName,
		Amount:        amount,
		Status:        "SUCCESS",
		Timestamp:     time.Now(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = b.channel.PublishWithContext(ctx,
		"",
		b.notificationQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		log.Printf("Failed to publish notification: %v", err)
		return err
	}

	log.Printf("Notification published [TX: %s, Merchant: %s]", txID, merchantName)
	return nil
}

func (b *Broker) ConsumePaymentConfirmations() (<-chan Delivery, error) {
	msgs, err := b.channel.Consume(
		b.queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return convertDeliveries(msgs), nil
}

func (b *Broker) ConsumeMerchantNotifications() (<-chan Delivery, error) {
	msgs, err := b.channel.Consume(
		b.notificationQueue.Name,
		"merchant-notification-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return convertDeliveries(msgs), nil
}

func convertDeliveries(msgs <-chan amqp.Delivery) <-chan Delivery {
	deliveries := make(chan Delivery)
	go func() {
		defer close(deliveries)
		for msg := range msgs {
			delivery := msg
			deliveries <- Delivery{
				Body: delivery.Body,
				Ack: func() error {
					return delivery.Ack(false)
				},
				Nack: func(requeue bool) error {
					return delivery.Nack(false, requeue)
				},
			}
		}
	}()
	return deliveries
}

// IsConnected returns true if RabbitMQ connection is alive.
func (b *Broker) IsConnected() bool {
	if b == nil || b.conn == nil || b.conn.IsClosed() {
		return false
	}
	return b.channel != nil
}

// Close gracefully closes the RabbitMQ channel and connection.
func (b *Broker) Close() {
	if b == nil {
		return
	}
	if b.channel != nil {
		b.channel.Close()
	}
	if b.conn != nil {
		b.conn.Close()
	}
	log.Println("RabbitMQ closed")
}
