package websocket

import (
	"sync"

	"duskforge-api/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Hub struct {
	clients    map[uuid.UUID]map[*Client]struct{}
	mu         sync.RWMutex
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Logger.Error("websocket-hub panic", zap.Any("panic", r))
				}
			}()
			select {
			case client := <-h.register:
				h.mu.Lock()
				if h.clients[client.UserID] == nil {
					h.clients[client.UserID] = make(map[*Client]struct{})
				}
				h.clients[client.UserID][client] = struct{}{}
				h.mu.Unlock()

			case client := <-h.unregister:
				h.mu.Lock()
				if conns, ok := h.clients[client.UserID]; ok {
					if _, present := conns[client]; present {
						delete(conns, client)
						if len(conns) == 0 {
							delete(h.clients, client.UserID)
						}
						close(client.send)
					}
				}
				h.mu.Unlock()
			}
		}()
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) SendToUser(userID uuid.UUID, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.clients[userID]
	if !ok {
		return
	}

	for client := range conns {
		select {
		case client.send <- event:
		default:
		}
	}
}
