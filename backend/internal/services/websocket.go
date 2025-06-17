package services

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"utunnel-pro/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

// WebSocketService handles real-time WebSocket connections
type WebSocketService struct {
	clients    map[string]*WebSocketClient
	clientsMux sync.RWMutex
	upgrader   websocket.Upgrader
	authService *AuthService
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID       string
	UserID   uuid.UUID
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *WebSocketService
	LastSeen time.Time
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService(authService *AuthService) *WebSocketService {
	return &WebSocketService{
		clients: make(map[string]*WebSocketClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		authService: authService,
	}
}

// HandleWebSocket handles WebSocket connection upgrades
func (ws *WebSocketService) HandleWebSocket(c *gin.Context) {
	// Get token from query parameter or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
		return
	}

	// Validate token
	user, err := ws.authService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Upgrade connection
	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &WebSocketClient{
		ID:       uuid.New().String(),
		UserID:   user.ID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      ws,
		LastSeen: time.Now(),
	}

	// Register client
	ws.registerClient(client)

	// Start goroutines
	go client.writePump()
	go client.readPump()

	log.Printf("WebSocket client connected: %s (User: %s)", client.ID, user.Username)
}

// registerClient registers a new WebSocket client
func (ws *WebSocketService) registerClient(client *WebSocketClient) {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()
	
	ws.clients[client.ID] = client
	
	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id": client.ID,
			"message":   "Connected to UTunnel Pro",
		},
		Timestamp: time.Now(),
	}
	
	client.sendMessage(welcomeMsg)
}

// unregisterClient removes a WebSocket client
func (ws *WebSocketService) unregisterClient(client *WebSocketClient) {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()
	
	if _, exists := ws.clients[client.ID]; exists {
		delete(ws.clients, client.ID)
		close(client.Send)
		log.Printf("WebSocket client disconnected: %s", client.ID)
	}
}

// BroadcastToUser sends a message to all connections of a specific user
func (ws *WebSocketService) BroadcastToUser(userID uuid.UUID, message WebSocketMessage) {
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()
	
	for _, client := range ws.clients {
		if client.UserID == userID {
			client.sendMessage(message)
		}
	}
}

// BroadcastToAll sends a message to all connected clients
func (ws *WebSocketService) BroadcastToAll(message WebSocketMessage) {
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()
	
	for _, client := range ws.clients {
		client.sendMessage(message)
	}
}

// BroadcastTunnelUpdate sends tunnel status updates to relevant users
func (ws *WebSocketService) BroadcastTunnelUpdate(tunnel *models.Tunnel, updateType string) {
	message := WebSocketMessage{
		Type: "tunnel_update",
		Data: map[string]interface{}{
			"update_type": updateType,
			"tunnel":      tunnel,
		},
		Timestamp: time.Now(),
	}
	
	ws.BroadcastToUser(tunnel.UserID, message)
}

// BroadcastMetrics sends real-time metrics to connected clients
func (ws *WebSocketService) BroadcastMetrics(metrics map[string]interface{}) {
	message := WebSocketMessage{
		Type:      "metrics_update",
		Data:      metrics,
		Timestamp: time.Now(),
	}
	
	ws.BroadcastToAll(message)
}

// BroadcastAlert sends alert notifications to connected clients
func (ws *WebSocketService) BroadcastAlert(alert map[string]interface{}) {
	message := WebSocketMessage{
		Type:      "alert",
		Data:      alert,
		Timestamp: time.Now(),
	}
	
	ws.BroadcastToAll(message)
}

// GetConnectedClients returns the number of connected clients
func (ws *WebSocketService) GetConnectedClients() int {
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()
	return len(ws.clients)
}

// GetUserConnections returns the number of connections for a specific user
func (ws *WebSocketService) GetUserConnections(userID uuid.UUID) int {
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()
	
	count := 0
	for _, client := range ws.clients {
		if client.UserID == userID {
			count++
		}
	}
	return count
}

// StartCleanup starts the cleanup routine for stale connections
func (ws *WebSocketService) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.cleanupStaleConnections()
		}
	}
}

// cleanupStaleConnections removes stale WebSocket connections
func (ws *WebSocketService) cleanupStaleConnections() {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()
	
	now := time.Now()
	for id, client := range ws.clients {
		if now.Sub(client.LastSeen) > 5*time.Minute {
			client.Conn.Close()
			delete(ws.clients, id)
			close(client.Send)
			log.Printf("Cleaned up stale WebSocket connection: %s", id)
		}
	}
}

// Client methods

// sendMessage sends a message to the WebSocket client
func (c *WebSocketClient) sendMessage(message WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}
	
	select {
	case c.Send <- data:
	default:
		c.Hub.unregisterClient(c)
	}
}

// readPump handles reading messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Hub.unregisterClient(c)
		c.Conn.Close()
	}()
	
	// Set read deadline and pong handler
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.LastSeen = time.Now()
		return nil
	})
	
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		
		c.LastSeen = time.Now()
		
		// Handle incoming messages
		var wsMessage WebSocketMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			continue
		}
		
		c.handleMessage(wsMessage)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			
			// Add queued messages to the current message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}
			
			if err := w.Close(); err != nil {
				return
			}
			
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages from clients
func (c *WebSocketClient) handleMessage(message WebSocketMessage) {
	switch message.Type {
	case "ping":
		// Respond with pong
		pongMsg := WebSocketMessage{
			Type:      "pong",
			Data:      map[string]interface{}{"timestamp": time.Now()},
			Timestamp: time.Now(),
		}
		c.sendMessage(pongMsg)
		
	case "subscribe":
		// Handle subscription to specific events
		if data, ok := message.Data.(map[string]interface{}); ok {
			if channel, ok := data["channel"].(string); ok {
				log.Printf("Client %s subscribed to channel: %s", c.ID, channel)
				// Store subscription info if needed
			}
		}
		
	case "unsubscribe":
		// Handle unsubscription from events
		if data, ok := message.Data.(map[string]interface{}); ok {
			if channel, ok := data["channel"].(string); ok {
				log.Printf("Client %s unsubscribed from channel: %s", c.ID, channel)
				// Remove subscription info if needed
			}
		}
		
	default:
		log.Printf("Unknown WebSocket message type: %s", message.Type)
	}
}
