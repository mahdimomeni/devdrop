package hub

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"devdrop/internal/database"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 32,
	WriteBufferSize: 1024 * 32,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all LAN connections
	},
}

var adjectives = []string{
	"Hyper", "Quantum", "Turbo", "Neon", "Cyber", "Atomic", "Solar", "Lunar",
	"Cosmic", "Vivid", "Static", "Binary", "Neural", "Shadow", "Vector", "Apex",
	"Echo", "Flux", "Omega", "Zero", "Alpha", "Pixel", "Nova", "Zenith",
}

var nouns = []string{
	"Gopher", "Falcon", "Otter", "Panda", "Viper", "Lynx", "Badger", "Fox",
	"Raven", "Hawk", "Wolf", "Beaver", "Cheetah", "Cobra", "Jaguar", "Lemur",
	"Osprey", "Condor", "Mantis", "Orca", "Bison", "Eagle", "Hornet", "Gecko",
}

// GenerateMoniker creates a memorable moniker like "HyperGopher" or fallback
func GenerateMoniker(ip string) string {
	adjIdx, err1 := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	nounIdx, err2 := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	if err1 != nil || err2 != nil {
		cleanIP := strings.ReplaceAll(ip, ":", ".")
		return fmt.Sprintf("Dev-%s", cleanIP)
	}
	return fmt.Sprintf("%s%s", adjectives[adjIdx.Int64()], nouns[nounIdx.Int64()])
}

// ExtractClientIP retrieves remote IP without port, taking proxies into account
func ExtractClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	if host == "::1" || host == "127.0.0.1" {
		return "127.0.0.1"
	}
	return host
}

type WSMessage struct {
	Type    string          `json:"type"`              // 'init', 'peers', 'message', 'presence', 'user_updated', 'file_expired', 'ping', 'pong'
	Payload json.RawMessage `json:"payload,omitempty"` // polymorphic payload
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	ip       string
	userName string
}

type Hub struct {
	db         *database.DB
	clients    map[*Client]bool
	userCounts map[string]int // userID -> active connection count
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub(db *database.DB) *Hub {
	return &Hub{
		db:         db,
		clients:    make(map[*Client]bool),
		userCounts: make(map[string]int),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.userCounts[client.userID]++
			firstConn := h.userCounts[client.userID] == 1
			h.mu.Unlock()

			if firstConn {
				h.BroadcastPresence(client.userID, true)
			}
			// Send updated peers list to all
			h.BroadcastPeerList()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.userCounts[client.userID]--
				lastConn := h.userCounts[client.userID] <= 0
				if lastConn {
					delete(h.userCounts, client.userID)
				}
				h.mu.Unlock()

				if lastConn {
					h.BroadcastPresence(client.userID, false)
				}
				h.BroadcastPeerList()
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.userCounts[userID] > 0
}

func (h *Hub) BroadcastMessage(msg *database.Message) {
	data, err := json.Marshal(map[string]any{
		"type":    "message",
		"payload": msg,
	})
	if err != nil {
		log.Printf("Error encoding message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// If receiver is "all" or "broadcast", send to all
	if msg.ReceiverID == "all" || msg.ReceiverID == "broadcast" || msg.ReceiverID == "" {
		for client := range h.clients {
			select {
			case client.send <- data:
			default:
			}
		}
		return
	}

	// 1-to-1: send to sender and receiver
	for client := range h.clients {
		if client.userID == msg.SenderID || client.userID == msg.ReceiverID {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) BroadcastPresence(userID string, isOnline bool) {
	data, err := json.Marshal(map[string]any{
		"type": "presence",
		"payload": map[string]any{
			"user_id":   userID,
			"is_online": isOnline,
		},
	})
	if err != nil {
		return
	}
	h.broadcast <- data
}

func (h *Hub) BroadcastPeerList() {
	users, err := h.db.GetAllUsers()
	if err != nil {
		log.Printf("Error getting users for peer list: %v", err)
		return
	}

	h.mu.RLock()
	for i := range users {
		users[i].IsOnline = h.userCounts[users[i].ID] > 0
	}
	h.mu.RUnlock()

	data, err := json.Marshal(map[string]any{
		"type":    "peers",
		"payload": users,
	})
	if err != nil {
		return
	}

	h.broadcast <- data
}

func (h *Hub) BroadcastUserUpdate(user *database.User) {
	data, err := json.Marshal(map[string]any{
		"type":    "user_updated",
		"payload": user,
	})
	if err != nil {
		return
	}
	h.broadcast <- data
	h.BroadcastPeerList()
}

func (h *Hub) BroadcastFileExpired(messageID string) {
	data, err := json.Marshal(map[string]any{
		"type": "file_expired",
		"payload": map[string]any{
			"message_id": messageID,
		},
	})
	if err != nil {
		return
	}
	h.broadcast <- data
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	ip := ExtractClientIP(r)

	// Fetch or create user
	user, err := h.db.GetUser(userID)
	if err != nil {
		// New user, generate moniker
		moniker := GenerateMoniker(ip)
		user, err = h.db.UpsertUser(userID, ip, moniker)
		if err != nil {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}
	} else {
		// Update IP and last seen
		_ = h.db.UpdateUserLastSeen(userID)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		ip:       ip,
		userName: user.DisplayName,
	}

	client.hub.register <- client

	// Send initial greeting with user info and current peer list
	allUsers, _ := h.db.GetAllUsers()
	h.mu.RLock()
	for i := range allUsers {
		allUsers[i].IsOnline = h.userCounts[allUsers[i].ID] > 0
	}
	h.mu.RUnlock()

	initMsg, _ := json.Marshal(map[string]any{
		"type": "init",
		"payload": map[string]any{
			"me":    user,
			"peers": allUsers,
		},
	})
	client.send <- initMsg

	go client.writePump()
	go client.readPump()
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS error: %v", err)
			}
			break
		}

		var incoming struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(message, &incoming); err != nil {
			continue
		}

		switch incoming.Type {
		case "ping":
			pong, _ := json.Marshal(map[string]string{"type": "pong"})
			select {
			case c.send <- pong:
			default:
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Drain queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
