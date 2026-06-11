package handler

import (
	"context"
	"net/http"
	"time"

	"qris-latency-optimizer/delivery/middleware"
	"qris-latency-optimizer/repository/postgres"
	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/repository/redis"

	"github.com/gin-gonic/gin"
)

// HealthHandler checks the liveness of all system dependencies.
type HealthHandler struct {
	broker *rabbitmq.Broker
}

func NewHealthHandler(broker *rabbitmq.Broker) *HealthHandler {
	return &HealthHandler{broker: broker}
}

type serviceStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (h *HealthHandler) Health(c *gin.Context) {
	overall := "ok"
	checks := map[string]serviceStatus{}

	// --- PostgreSQL ---
	sqlDB, err := postgres.DB.DB()
	if err != nil || sqlDB.PingContext(context.Background()) != nil {
		checks["postgres"] = serviceStatus{Status: "error", Message: "unreachable"}
		overall = "degraded"
		middleware.RecordServiceHealth("postgres", false)
	} else {
		checks["postgres"] = serviceStatus{Status: "ok"}
		middleware.RecordServiceHealth("postgres", true)
	}

	// --- Redis ---
	if !redis.RedisAvailable {
		checks["redis"] = serviceStatus{Status: "error", Message: "unavailable"}
		overall = "degraded"
		middleware.RecordServiceHealth("redis", false)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := redis.RedisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = serviceStatus{Status: "error", Message: err.Error()}
			overall = "degraded"
			middleware.RecordServiceHealth("redis", false)
		} else {
			checks["redis"] = serviceStatus{Status: "ok"}
			middleware.RecordServiceHealth("redis", true)
		}
	}

	// --- RabbitMQ ---
	if !h.broker.IsConnected() {
		checks["rabbitmq"] = serviceStatus{Status: "error", Message: "disconnected"}
		overall = "degraded"
		middleware.RecordServiceHealth("rabbitmq", false)
	} else {
		checks["rabbitmq"] = serviceStatus{Status: "ok"}
		middleware.RecordServiceHealth("rabbitmq", true)
	}

	httpStatus := http.StatusOK
	if overall != "ok" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":    overall,
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  checks,
	})
}

// PingHandler is kept for a lightweight liveness check.
type PingHandler struct{}

func NewPingHandler() *PingHandler { return &PingHandler{} }

func (h *PingHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong", "timestamp": time.Now().Format(time.RFC3339)})
}
