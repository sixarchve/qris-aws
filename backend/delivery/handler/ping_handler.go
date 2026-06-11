package handler

import (
	"context"
	"net/http"
	"time"

	"qris-latency-optimizer/repository/postgres"
	"qris-latency-optimizer/repository/redis"

	"github.com/gin-gonic/gin"
)

// HealthHandler checks the liveness of all system dependencies.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
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
	} else {
		checks["postgres"] = serviceStatus{Status: "ok"}
	}

	// --- Redis ---
	if !redis.RedisAvailable {
		checks["redis"] = serviceStatus{Status: "error", Message: "unavailable"}
		overall = "degraded"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := redis.RedisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = serviceStatus{Status: "error", Message: err.Error()}
			overall = "degraded"
		} else {
			checks["redis"] = serviceStatus{Status: "ok"}
		}
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
