package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devdrop/internal/database"
	"devdrop/internal/hub"
	"devdrop/internal/preview"
	"devdrop/internal/transfer"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ServerHandler struct {
	db              *database.DB
	hub             *hub.Hub
	transferManager *transfer.Manager
	previewService  *preview.Service
}

func NewServerHandler(db *database.DB, h *hub.Hub, tm *transfer.Manager) *ServerHandler {
	return &ServerHandler{
		db:              db,
		hub:             h,
		transferManager: tm,
		previewService:  preview.NewService(),
	}
}

// RegisterRoutes configures HTTP routes on a chi.Router
func (s *ServerHandler) RegisterRoutes(r chi.Router) {
	r.Get("/ws", s.handleWebSocketAuth)

	r.Route("/api", func(api chi.Router) {
		// Public Auth Endpoints
		api.Route("/auth", func(auth chi.Router) {
			auth.Get("/status", s.handleAuthStatus)
			auth.Post("/login", s.handleAuthLogin)
			auth.Post("/logout", s.handleAuthLogout)
			auth.Get("/devices", s.handleAuthGetDevices)
		})

		// Protected Workspace Endpoints
		api.Group(func(protected chi.Router) {
			protected.Use(s.AuthMiddleware)

			// User & Profile
			protected.Get("/profile", s.handleGetProfile)
			protected.Put("/profile/name", s.handleUpdateName)
			protected.Get("/peers", s.handleGetPeers)

			// Messaging & History
			protected.Get("/messages", s.handleGetMessages)
			protected.Post("/messages", s.handleSendMessage)
			protected.Post("/messages/{id}/reactions", s.handleToggleReaction)

			// Link Preview
			protected.Get("/preview", s.handleGetLinkPreview)

			// File Transfers
			protected.Post("/upload", s.handleUpload)
			protected.Get("/transfers/{id}/download", s.handleDownload)
			protected.Get("/transfers/{id}/meta", s.handleGetTransferMeta)
		})
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
	beforeID := r.URL.Query().Get("before")

	limit := 40
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	messages, hasMore, err := s.db.GetMessages(userID, peerID, limit, beforeID)
	if err != nil {
		http.Error(w, "failed to load messages", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []database.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Has-More", strconv.FormatBool(hasMore))
	_ = json.NewEncoder(w).Encode(messages)
}

func (s *ServerHandler) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SenderID    string  `json:"sender_id"`
		ReceiverID  string  `json:"receiver_id"`
		Type        string  `json:"type"` // "text" or "code"
		Body        string  `json:"body"`
		ReplyToID   *string `json:"reply_to_id,omitempty"`
		Language    string  `json:"language,omitempty"`
		CodeContent string  `json:"code_content,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SenderID == "" || req.ReceiverID == "" {
		http.Error(w, "sender_id and receiver_id are required", http.StatusBadRequest)
		return
	}

	if req.ReplyToID != nil && strings.TrimSpace(*req.ReplyToID) == "" {
		req.ReplyToID = nil
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
		msg, err = s.db.SaveCodeMessage(msgID, req.SenderID, req.ReceiverID, req.Body, req.Language, req.CodeContent, req.ReplyToID, createdAt)
	} else {
		if strings.TrimSpace(req.Body) == "" {
			http.Error(w, "body cannot be empty", http.StatusBadRequest)
			return
		}
		msg, err = s.db.SaveTextMessage(msgID, req.SenderID, req.ReceiverID, req.Body, req.ReplyToID, createdAt)
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

func (s *ServerHandler) handleGetLinkPreview(w http.ResponseWriter, r *http.Request) {
	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if targetURL == "" {
		http.Error(w, "missing url query parameter", http.StatusBadRequest)
		return
	}

	previewData, err := s.previewService.Fetch(r.Context(), targetURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(previewData)
}

func (s *ServerHandler) handleToggleReaction(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		http.Error(w, "missing message ID", http.StatusBadRequest)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Emoji  string `json:"emoji"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.Emoji = strings.TrimSpace(req.Emoji)

	if req.UserID == "" || req.Emoji == "" {
		http.Error(w, "user_id and emoji are required", http.StatusBadRequest)
		return
	}

	reactions, added, senderID, receiverID, err := s.db.ToggleReaction(messageID, req.UserID, req.Emoji)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "message not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update reaction", http.StatusInternalServerError)
		return
	}

	action := "removed"
	if added {
		action = "added"
	}

	// Broadcast via WebSocket
	s.hub.BroadcastReaction(hub.ReactionEventPayload{
		MessageID:  messageID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		UserID:     req.UserID,
		Emoji:      req.Emoji,
		Action:     action,
		Reactions:  reactions,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":   true,
		"action":    action,
		"reactions": reactions,
	})
}

// ExtractToken extracts authentication token from header, cookie, or query parameter
func ExtractToken(r *http.Request) string {
	// 1. Authorization: Bearer <token>
	if auth := r.Header.Get("Authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	// 2. Custom headers
	if t := r.Header.Get("X-Device-Token"); t != "" {
		return strings.TrimSpace(t)
	}
	if t := r.Header.Get("X-Auth-Token"); t != "" {
		return strings.TrimSpace(t)
	}
	// 3. Cookie
	if c, err := r.Cookie("devdrop_auth_token"); err == nil && c.Value != "" {
		return strings.TrimSpace(c.Value)
	}
	// 4. Query param
	if t := r.URL.Query().Get("token"); t != "" {
		return strings.TrimSpace(t)
	}
	if t := r.URL.Query().Get("auth_token"); t != "" {
		return strings.TrimSpace(t)
	}
	return ""
}

// AuthMiddleware protects routes when password protection is active
func (s *ServerHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasPass, err := s.db.HasPassword()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// If no password set on this instance, allow traffic
		if !hasPass {
			next.ServeHTTP(w, r)
			return
		}

		token := ExtractToken(r)
		if token == "" {
			http.Error(w, "unauthorized: password required", http.StatusUnauthorized)
			return
		}

		valid, _, err := s.db.ValidateTrustedDevice(token)
		if err != nil || !valid {
			http.Error(w, "unauthorized: invalid or revoked device token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleWebSocketAuth enforces auth before upgrading WS connections
func (s *ServerHandler) handleWebSocketAuth(w http.ResponseWriter, r *http.Request) {
	hasPass, err := s.db.HasPassword()
	if err == nil && hasPass {
		token := ExtractToken(r)
		if token == "" {
			http.Error(w, "unauthorized: password required", http.StatusUnauthorized)
			return
		}
		valid, _, err := s.db.ValidateTrustedDevice(token)
		if err != nil || !valid {
			http.Error(w, "unauthorized: invalid or revoked device token", http.StatusUnauthorized)
			return
		}
	}
	s.hub.HandleWebSocket(w, r)
}

func (s *ServerHandler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	hasPass, err := s.db.HasPassword()
	if err != nil {
		http.Error(w, "failed to check auth status", http.StatusInternalServerError)
		return
	}

	if !hasPass {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"has_password":  false,
			"authenticated": false,
			"trusted":       false,
		})
		return
	}

	token := ExtractToken(r)
	valid, dev, _ := s.db.ValidateTrustedDevice(token)
	deviceName := ""
	if dev != nil {
		deviceName = dev.DeviceName
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"has_password":  true,
		"authenticated": valid,
		"trusted":       valid,
		"device_name":   deviceName,
	})
}

func (s *ServerHandler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	hasPass, err := s.db.HasPassword()
	if err != nil {
		http.Error(w, "failed to check auth status", http.StatusInternalServerError)
		return
	}
	if !hasPass {
		http.Error(w, "no password configured; use setup first", http.StatusBadRequest)
		return
	}

	var req struct {
		Password    string `json:"password"`
		TrustDevice bool   `json:"trust_device"`
		DeviceName  string `json:"device_name"`
		UserID      string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ok, err := s.db.VerifyPassword(req.Password)
	if err != nil || !ok {
		http.Error(w, "incorrect password", http.StatusUnauthorized)
		return
	}

	ip := hub.ExtractClientIP(r)
	ua := r.UserAgent()
	if req.DeviceName == "" {
		req.DeviceName = "Trusted Device"
	}

	token, err := s.db.RegisterTrustedDevice(req.DeviceName, ip, ua, req.UserID)
	if err != nil {
		http.Error(w, "failed to register trusted device", http.StatusInternalServerError)
		return
	}

	maxAge := 365 * 24 * 3600
	if !req.TrustDevice {
		maxAge = 24 * 3600
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "devdrop_auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"token":   token,
	})
}

func (s *ServerHandler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	token := ExtractToken(r)
	if token != "" {
		_ = s.db.RevokeTrustedDevice(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "devdrop_auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *ServerHandler) handleAuthGetDevices(w http.ResponseWriter, r *http.Request) {
	token := ExtractToken(r)
	valid, dev, _ := s.db.ValidateTrustedDevice(token)
	if !valid {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	count, _ := s.db.GetTrustedDevicesCount()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"total_trusted_devices": count,
		"current_device":        dev,
	})
}

