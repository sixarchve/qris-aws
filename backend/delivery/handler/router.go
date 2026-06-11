package handler

import (
	"qris-latency-optimizer/delivery/middleware"
	"qris-latency-optimizer/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handlers struct {
	Merchant    *MerchantHandler
	QRIS        *QRISHandler
	Transaction *TransactionHandler
	Ping        *PingHandler
	Health      *HealthHandler
}

func SetupRouter(h *Handlers, wsHub *websocket.Hub) *gin.Engine {
	r := gin.Default()
	middleware.CorsHandler(r)
	r.Use(middleware.PrometheusMiddleware())

	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(middleware.Registry, promhttp.HandlerOpts{})))

	r.GET("/api/ping", h.Ping.Ping)
	r.GET("/api/health", h.Health.Health)
	r.GET("/api/qris", h.QRIS.GenerateDynamic)
	r.GET("/api/merchants", h.Merchant.GetMerchants)
	r.GET("/api/transactions/:id", h.Transaction.GetTransactionStatus)
	r.POST("/api/transactions/scan", h.Transaction.ScanQR)
	r.POST("/api/transactions/:id/confirm", h.Transaction.ConfirmPaymentAsync)

	r.GET("/api/ws/status", func(c *gin.Context) {
		merchantID := c.Query("merchant_id")
		response := gin.H{
			"connected_count": wsHub.GetConnectedCount(),
		}
		if merchantID != "" {
			response["merchant_id"] = merchantID
			response["merchant_connected"] = wsHub.IsMerchantConnected(merchantID)
			response["merchant_connection_count"] = wsHub.GetMerchantConnectionCount(merchantID)
			response["pending_notifications"] = wsHub.GetPendingCount(merchantID)
		}
		c.JSON(200, response)
	})
	r.GET("/ws", wsHub.HandleWebSocket)

	return r
}
