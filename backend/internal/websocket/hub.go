package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

const maxPendingPerMerchant = 20

type Client struct {
	MerchantID string
	Conn       *websocket.Conn
	Send       chan interface{}
	Hub        *Hub
}

type Hub struct {
	clients    map[string]map[*Client]bool
	pending    map[string][]interface{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan interface{}
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		pending:    make(map[string][]interface{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan interface{}, 256),
	}
}

// Run starts the hub event loop. Call it in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.MerchantID] == nil {
				h.clients[client.MerchantID] = make(map[*Client]bool)
			}
			h.clients[client.MerchantID][client] = true

			pending := h.pending[client.MerchantID]
			delete(h.pending, client.MerchantID)
			merchantConnections := len(h.clients[client.MerchantID])
			totalConnections := h.countConnectionsLocked()
			h.mu.Unlock()

			log.Printf("Merchant %s WebSocket connected [merchant connections: %d, total: %d]", client.MerchantID, merchantConnections, totalConnections)
			h.flushPending(client, pending)

		case client := <-h.unregister:
			h.mu.Lock()
			if merchantClients, exists := h.clients[client.MerchantID]; exists {
				if _, registered := merchantClients[client]; registered {
					delete(merchantClients, client)
					close(client.Send)
				}
				if len(merchantClients) == 0 {
					delete(h.clients, client.MerchantID)
				}
			}
			merchantConnections := len(h.clients[client.MerchantID])
			totalConnections := h.countConnectionsLocked()
			h.mu.Unlock()

			log.Printf("Merchant %s WebSocket disconnected [merchant connections: %d, total: %d]", client.MerchantID, merchantConnections, totalConnections)

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, merchantClients := range h.clients {
				for client := range merchantClients {
					select {
					case client.Send <- message:
					default:
						log.Printf("Merchant %s WebSocket channel full, broadcast skipped", client.MerchantID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) countConnectionsLocked() int {
	total := 0
	for _, merchantClients := range h.clients {
		total += len(merchantClients)
	}
	return total
}

func (h *Hub) queuePendingLocked(merchantID string, notification interface{}) {
	pending := append(h.pending[merchantID], notification)
	if len(pending) > maxPendingPerMerchant {
		pending = pending[len(pending)-maxPendingPerMerchant:]
	}
	h.pending[merchantID] = pending
}

func (h *Hub) flushPending(client *Client, pending []interface{}) {
	if len(pending) == 0 {
		return
	}

	sent := 0
	for _, notification := range pending {
		select {
		case client.Send <- notification:
			sent++
		default:
			h.mu.Lock()
			h.queuePendingLocked(client.MerchantID, notification)
			h.mu.Unlock()
		}
	}

	if sent > 0 {
		log.Printf("Delivered %d pending notification(s) to merchant %s", sent, client.MerchantID)
	}
}

// SendToMerchant sends a notification to every active WebSocket for a merchant.
// If the merchant is temporarily offline, keep a small in-memory backlog for
// delivery on the next WebSocket registration.
func (h *Hub) SendToMerchant(merchantID string, notification interface{}) error {
	h.mu.Lock()
	merchantClients := h.clients[merchantID]
	if len(merchantClients) == 0 {
		h.queuePendingLocked(merchantID, notification)
		pendingCount := len(h.pending[merchantID])
		h.mu.Unlock()
		log.Printf("Merchant %s not connected, queued notification [pending: %d]", merchantID, pendingCount)
		return nil
	}

	delivered := 0
	for client := range merchantClients {
		select {
		case client.Send <- notification:
			delivered++
		default:
			log.Printf("Merchant %s WebSocket channel full, one connection skipped", merchantID)
		}
	}

	if delivered == 0 {
		h.queuePendingLocked(merchantID, notification)
		pendingCount := len(h.pending[merchantID])
		h.mu.Unlock()
		log.Printf("Merchant %s has no writable WebSocket, queued notification [pending: %d]", merchantID, pendingCount)
		return nil
	}

	connectionCount := len(merchantClients)
	h.mu.Unlock()
	log.Printf("Notification sent to merchant %s [delivered connections: %d/%d]", merchantID, delivered, connectionCount)
	return nil
}

// GetConnectedCount returns the number of active WebSocket connections.
func (h *Hub) GetConnectedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.countConnectionsLocked()
}

func (h *Hub) IsMerchantConnected(merchantID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[merchantID]) > 0
}

func (h *Hub) GetMerchantConnectionCount(merchantID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[merchantID])
}

func (h *Hub) GetPendingCount(merchantID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.pending[merchantID])
}
