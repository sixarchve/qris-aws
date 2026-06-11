package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qris-latency-optimizer/config"
	"qris-latency-optimizer/delivery/handler"
	"qris-latency-optimizer/internal/qris"
	"qris-latency-optimizer/internal/websocket"
	"qris-latency-optimizer/repository/postgres"
	"qris-latency-optimizer/repository/redis"
	"qris-latency-optimizer/usecase"
)

type websocketNotificationPublisher struct {
	hub *websocket.Hub
}

func (p websocketNotificationPublisher) PublishNotification(txID, merchantID, merchantName string, amount float64) error {
	if p.hub == nil {
		return nil
	}

	notification := map[string]interface{}{
		"type":           "transaction_notification",
		"transaction_id": txID,
		"merchant_id":    merchantID,
		"merchant_name":  merchantName,
		"amount":         amount,
		"status":         "SUCCESS",
		"timestamp":      time.Now(),
	}

	return p.hub.SendToMerchant(merchantID, notification)
}

func setupInfrastructure() {
	config.Load()

	postgres.ConnectDB()
	fmt.Println("PostgreSQL connected & migrated")

	redis.ConnectRedis()
	redis.WarmUpCache()
	fmt.Println("Redis connected & cache warmed")
}

func main() {

	setupInfrastructure()
	websocket.InitWSConfig()

	merchantRepo := postgres.NewMerchantRepository(postgres.DB)
	txRepo := postgres.NewTransactionRepository(postgres.DB)
	merchantCache := redis.NewMerchantCache()
	txCache := redis.NewTransactionCache()
	merchantPrefetcher := redis.NewMerchantPrefetcher()
	qrisCodec := qris.NewCodec()

	merchantUsecase := usecase.NewMerchantUsecase(merchantRepo)
	qrisUsecase := usecase.NewQRISUsecase(merchantRepo, merchantCache, merchantPrefetcher, qrisCodec)
	wsHub := websocket.NewHub()
	txUsecase := usecase.NewTransactionUsecase(
		txRepo,
		merchantRepo,
		txCache,
		merchantCache,
		websocketNotificationPublisher{hub: wsHub},
		qrisCodec,
	)

	handlers := &handler.Handlers{
		Merchant:    handler.NewMerchantHandler(merchantUsecase),
		QRIS:        handler.NewQRISHandler(qrisUsecase),
		Transaction: handler.NewTransactionHandler(txUsecase),
		Ping:        handler.NewPingHandler(),
		Health:      handler.NewHealthHandler(),
	}

	go wsHub.Run()
	fmt.Println("WebSocket Hub initialized")

	r := handler.SetupRouter(handlers, wsHub)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		fmt.Println("Server running on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Shutdown complete")
}

var _ usecase.NotificationPublisher = websocketNotificationPublisher{}
