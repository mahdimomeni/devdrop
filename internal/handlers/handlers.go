package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devdrop/internal/database"
	"devdrop/internal/hub"
	"devdrop/internal/transfer"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ServerHandler struct {
	db              *database.DB
	hub             *hub.Hub
	transferManager *transfer.Manager
}

func NewServerHandler(db *database.DB, h *hub.Hub, tm *transfer.Manager) *ServerHandler {
	return &ServerHandler{
		db:              db,
		hub:             h,
		transferManager: tm,
	}
}

// RegisterRoutes configures HTTP routes on a chi.Router
func (s *ServerHandler) RegisterRoutes(r chi.Router) {
	r.Get("/ws", s.hub.HandleWebSocket)

	r.Route("/api", func(api chi.Router) {
		// User & Profile
		api.Get("/profile", s.handleGetProfile)
		api.Put("/profile/name", s.handleUpdateName)
		api.Get("/peers", s.handleGetPeers)

		// Messaging & History
		api.Get("/messages", s.handleGetMessages)
		api.Post("/messages", s.handleSendMessage)

		// File Transfers
		api.Post("/upload", s.handleUpload)
		api.Get("/transfers/{id}/download", s.handleDownload)
		api.Get("/transfers/{id}/meta", s.handleGetTransferMeta)
	})
}

func (s *ServerHandler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	ip := hub.ExtractClientIP(r)
	user, err := s.db.GetUser(userID)
	if err != nil {
		// Create initial user
		moniker := hub.GenerateMoniker(ip)
		user, err = s.db.UpsertUser(userID, ip, moniker)
		if err != nil {
			http.Error(w, "failed to initialize user profile", http.StatusInternalServerError)
			return
		}
	} else {
		// Update IP address if changed
		user, _ = s.db.UpsertUser(userID, ip, user.DisplayName)
	}

	user.IsOnline = s.hub.IsUserOnline(user.ID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (s *ServerHandler) handleUpdateName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		http.Error(w, "display_name cannot be empty", http.StatusBadRequest)
		return
	}
	if len(req.DisplayName) > 40 {
		req.DisplayName = req.DisplayName[:40]
	}

	if err := s.db.UpdateUserDisplayName(req.UserID, req.DisplayName); err != nil {
		http.Error(w, "failed to update display name", http.StatusInternalServerError)
		return
	}

	user, err := s.db.GetUser(req.UserID)
	if err == nil {
		user.IsOnline = s.hub.IsUserOnline(user.ID)
		s.hub.BroadcastUserUpdate(user)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"user":    user,
	})
}

func (s *ServerHandler) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetAllUsers()
	if err != nil {
		http.Error(w, "failed to query peers", http.StatusInternalServerError)
		return
	}

	for i := range users {
		users[i].IsOnline = s.hub.IsUserOnline(users[i].ID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

func (s *ServerHandler) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	peerID := r.URL.Query().Get("peer_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	messages, err := s.db.GetMessages(userID, peerID, limit)
	if err != nil {
		http.Error(w, "failed to load messages", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []database.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(messages)
}

func (s *ServerHandler) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SenderID    string `json:"sender_id"`
		ReceiverID  string `json:"receiver_id"`
		Type        string `json:"type"` // "text" or "code"
		Body        string `json:"body"`
		Language    string `json:"language,omitempty"`
		CodeContent string `json:"code_content,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SenderID == "" || req.ReceiverID == "" {
		http.Error(w, "sender_id and receiver_id are required", http.StatusBadRequest)
		return
	}

	msgID := uuid.New().String()
	createdAt := time.Now().UTC()

	var msg *database.Message
	var err error

	if req.Type == "code" {
		if req.CodeContent == "" {
			http.Error(w, "code_content cannot be empty", http.StatusBadRequest)
			return
		}
		if req.Language == "" {
			req.Language = "plaintext"
		}
		msg, err = s.db.SaveCodeMessage(msgID, req.SenderID, req.ReceiverID, req.Body, req.Language, req.CodeContent, createdAt)
	} else {
		if strings.TrimSpace(req.Body) == "" {
			http.Error(w, "body cannot be empty", http.StatusBadRequest)
			return
		}
		msg, err = s.db.SaveTextMessage(msgID, req.SenderID, req.ReceiverID, req.Body, createdAt)
	}

	if err != nil {
		http.Error(w, "failed to save message", http.StatusInternalServerError)
		return
	}

	// Broadcast message via WebSocket
	s.hub.BroadcastMessage(msg)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(msg)
}

func (s *ServerHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	res, err := s.transferManager.HandleUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *ServerHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	transferID := chi.URLParam(r, "id")
	if transferID == "" {
		http.Error(w, "missing transfer ID", http.StatusBadRequest)
		return
	}

	s.transferManager.ServeDownload(w, r, transferID)
}

func (s *ServerHandler) handleGetTransferMeta(w http.ResponseWriter, r *http.Request) {
	transferID := chi.URLParam(r, "id")
	if transferID == "" {
		http.Error(w, "missing transfer ID", http.StatusBadRequest)
		return
	}

	t, err := s.db.GetTransfer(transferID)
	if err != nil {
		http.Error(w, "transfer not found", http.StatusNotFound)
		return
	}

	// Don't leak raw disk path
	t.FilePath = ""

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}
