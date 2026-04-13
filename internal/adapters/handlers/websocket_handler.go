package handlers

import (
	"net/http"

	"duskforge-api/pkg/auth"
	ws "duskforge-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	hub               *ws.Hub
	accessTokenSecret string
}

func NewWebSocketHandler(hub *ws.Hub, accessTokenSecret string) *WebSocketHandler {
	return &WebSocketHandler{hub: hub, accessTokenSecret: accessTokenSecret}
}

// @Summary      WebSocket connection
// @Description  Establishes a WebSocket connection for real-time notifications. The connection is read-only — actions are performed via REST endpoints. Events pushed: message.new, message.updated, message.deleted, reaction.added, reaction.removed, conversation.read, messaging.blocked, messaging.unblocked, import.progress. The import.progress event is sent during a Letterboxd import with fields: status (processing|completed|failed), phase (resolving|importing|done), resolved, total, and result (when completed).
// @Tags         websocket
// @Produce      json
// @Param        token query string true "JWT access token"
// @Success      101 "Switching Protocols — WebSocket connection established"
// @Failure      401 {object} response.Response "Missing or invalid token"
// @Router       /ws [get]
func (h *WebSocketHandler) Connect(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	claims, err := auth.ValidateAccessToken(tokenStr, h.accessTokenSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(h.hub, conn, claims.UserID)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
