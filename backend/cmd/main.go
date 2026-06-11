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
	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/repository/redis"
	"qris-latency-optimizer/usecase"
	"qris-latency-optimizer/worker"
)

func setupInfrastructure() *rabbitmq.Broker {
	config.Load()

	postgres.ConnectDB()
	fmt.Println("PostgreSQL connected & migrated")

	redis.ConnectRedis()
	redis.WarmUpCache()
	fmt.Println("Redis connected & cache warmed")

	return rabbitmq.ConnectRabbitMQ()
}

func main() {

	broker := setupInfrastructure()
	websocket.InitWSConfig()

	merchantRepo := postgres.NewMerchantRepository(postgres.DB)
	txRepo := postgres.NewTransactionRepository(postgres.DB)
	merchantCache := redis.NewMerchantCache()
	txCache := redis.NewTransactionCache()
	merchantPrefetcher := redis.NewMerchantPrefetcher()
	qrisCodec := qris.NewCodec()

	merchantUsecase := usecase.NewMerchantUsecase(merchantRepo)
	qrisUsecase := usecase.NewQRISUsecase(merchantRepo, merchantCache, merchantPrefetcher, qrisCodec)
	txUsecase := usecase.NewTransactionUsecase(
		txRepo,
		merchantRepo,
		txCache,
		merchantCache,
		broker,
		broker,
		qrisCodec,
	)

	handlers := &handler.Handlers{
		Merchant:    handler.NewMerchantHandler(merchantUsecase),
		QRIS:        handler.NewQRISHandler(qrisUsecase),
		Transaction: handler.NewTransactionHandler(txUsecase),
		Ping:        handler.NewPingHandler(),
		Health:      handler.NewHealthHandler(broker),
	}

	wsHub := websocket.NewHub()
	go wsHub.Run()
	fmt.Println("WebSocket Hub initialized")

	worker.StartPaymentConsumer(broker, txUsecase)
	worker.StartNotificationConsumer(broker, wsHub)
	fmt.Println("RabbitMQ workers started")

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

	broker.Close()
	fmt.Println("Shutdown complete")
}
