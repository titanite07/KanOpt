package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost in development
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:3000" || origin == "http://localhost:3001"
	},
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool
	mutex      sync.RWMutex
	logger     *logrus.Logger
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	roomID string
}

type Message struct {
	Type      string                 `json:"type"`
	BoardID   string                 `json:"boardId,omitempty"`
	UserID    string                 `json:"userId,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			
			// Add to room if specified
			if client.roomID != "" {
				if h.rooms[client.roomID] == nil {
					h.rooms[client.roomID] = make(map[*Client]bool)
				}
				h.rooms[client.roomID][client] = true
			}
			h.mutex.Unlock()
			
			h.logger.WithFields(logrus.Fields{
				"user_id": client.userID,
				"room_id": client.roomID,
				"total_clients": len(h.clients),
			}).Info("Client connected")

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				
				// Remove from room
				if client.roomID != "" && h.rooms[client.roomID] != nil {
					delete(h.rooms[client.roomID], client)
					if len(h.rooms[client.roomID]) == 0 {
						delete(h.rooms, client.roomID)
					}
				}
			}
			h.mutex.Unlock()
			
			h.logger.WithFields(logrus.Fields{
				"user_id": client.userID,
				"room_id": client.roomID,
				"total_clients": len(h.clients),
			}).Info("Client disconnected")

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	if room, exists := h.rooms[roomID]; exists {
		for client := range room {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
				delete(room, client)
			}
		}
	}
}

func (h *Hub) GetConnectionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request, logger *logrus.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to upgrade connection")
		return
	}

	userID := r.URL.Query().Get("userId")
	roomID := r.URL.Query().Get("boardId")
	
	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		roomID: roomID,
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		var message Message
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			c.hub.logger.WithError(err).Error("Failed to unmarshal message")
			continue
		}

		message.UserID = c.userID
		message.Timestamp = time.Now()

		// Process different message types
		switch message.Type {
		case "ping":
			// Respond with pong
			pongMessage := Message{
				Type:      "pong",
				Timestamp: time.Now(),
			}
			if pongBytes, err := json.Marshal(pongMessage); err == nil {
				select {
				case c.send <- pongBytes:
				default:
					close(c.send)
					return
				}
			}
		case "join_board":
			// Client wants to join a specific board room
			if boardID, ok := message.Data["boardId"].(string); ok {
				c.roomID = boardID
				c.hub.register <- c // Re-register with new room
			}
		case "cursor_move":
			// Broadcast cursor position to other clients in the same room
			if c.roomID != "" {
				if responseBytes, err := json.Marshal(message); err == nil {
					c.hub.BroadcastToRoom(c.roomID, responseBytes)
				}
			}
		case "task_drag":
			// Broadcast task dragging state
			if c.roomID != "" {
				if responseBytes, err := json.Marshal(message); err == nil {
					c.hub.BroadcastToRoom(c.roomID, responseBytes)
				}
			}
		default:
			c.hub.logger.WithField("message_type", message.Type).Warn("Unknown message type")
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
