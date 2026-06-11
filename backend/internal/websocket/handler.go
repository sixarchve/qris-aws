package websocket

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSConfig struct {
	ReadDeadline      time.Duration
	WriteDeadline     time.Duration
	IdleCheckInterval time.Duration
	IdleThreshold     time.Duration
	MaxMessageSize    int64
}

func LoadWSConfig() WSConfig {
	readDeadline := getEnvDuration("WEBSOCKET_READ_DEADLINE", 5*time.Minute)
	writeDeadline := getEnvDuration("WEBSOCKET_WRITE_DEADLINE", 10*time.Second)
	idleCheckInterval := getEnvDuration("WEBSOCKET_IDLE_CHECK_INTERVAL", 60*time.Second)
	idleThreshold := getEnvDuration("WEBSOCKET_IDLE_THRESHOLD", 4*time.Minute)
	maxMessageSize := getEnvInt64("WEBSOCKET_MAX_MESSAGE_SIZE", 65536)

	cfg := WSConfig{
		ReadDeadline:      readDeadline,
		WriteDeadline:     writeDeadline,
		IdleCheckInterval: idleCheckInterval,
		IdleThreshold:     idleThreshold,
		MaxMessageSize:    maxMessageSize,
	}

	log.Printf("✓ WebSocket config (from .env):")
	log.Printf("  - ReadDeadline: %v", cfg.ReadDeadline)
	log.Printf("  - WriteDeadline: %v", cfg.WriteDeadline)
	log.Printf("  - IdleCheck every: %v", cfg.IdleCheckInterval)
	log.Printf("  - Send ping only if idle > %v", cfg.IdleThreshold)
	log.Printf("  - MaxMessageSize: %d bytes", cfg.MaxMessageSize)

	return cfg
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("⚠ Invalid duration for %s: %s, using default: %v", key, val, defaultVal)
		return defaultVal
	}
	return d
}

func getEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return i
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var wsConfig WSConfig

func InitWSConfig() {
	wsConfig = LoadWSConfig()
}

func (h *Hub) HandleWebSocket(c *gin.Context) {
	merchantID := c.Query("merchant_id")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "merchant_id required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}

	// ✅ CORRECT: Use SetReadLimit (available in v1.5.3)
	conn.SetReadLimit(wsConfig.MaxMessageSize)

	client := &Client{
		MerchantID: merchantID,
		Conn:       conn,
		Send:       make(chan interface{}, 256),
		Hub:        h,
	}

	h.register <- client

	go h.readPump(client)
	go h.writePump(client)
}

func (h *Hub) readPump(client *Client) {
	defer func() {
		h.unregister <- client
		client.Conn.Close()
	}()

	client.Conn.SetReadDeadline(time.Now().Add(wsConfig.ReadDeadline))

	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(wsConfig.ReadDeadline))
		return nil
	})

	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠ WebSocket error for %s: %v", client.MerchantID, err)
			}
			return
		}

		_ = msg
	}
}

func (h *Hub) writePump(client *Client) {
	lastMessageTime := time.Now()

	ticker := time.NewTicker(wsConfig.IdleCheckInterval)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Conn.SetWriteDeadline(time.Now().Add(wsConfig.WriteDeadline))
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(wsConfig.WriteDeadline))
			if err := client.Conn.WriteJSON(message); err != nil {
				log.Printf("⚠ WebSocket write error: %v", err)
				return
			}

			lastMessageTime = time.Now()

		case <-ticker.C:
			if time.Since(lastMessageTime) > wsConfig.IdleThreshold {
				client.Conn.SetWriteDeadline(time.Now().Add(wsConfig.WriteDeadline))
				if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("⚠ Failed to send ping: %v", err)
					return
				}
			}

		}
	}
}
