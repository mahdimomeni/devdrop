package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"devdrop/internal/database"
	"devdrop/internal/hub"
	"devdrop/internal/transfer"

	"github.com/go-chi/chi/v5"
)

func TestSendMessageWithReply(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devdrop_handler_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := database.InitDB(tempDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	wsHub := hub.NewHub(db)
	uploadDir := filepath.Join(tempDir, "uploads")
	tm, err := transfer.NewManager(uploadDir, db, wsHub)
	if err != nil {
		t.Fatalf("failed to init tm: %v", err)
	}

	h := NewServerHandler(db, wsHub, tm)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// 1. Create original message
	origMsg, err := db.SaveTextMessage("msg-orig-1", "user-1", "broadcast", "Initial Announcement", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to save original message: %v", err)
	}

	// 2. Post reply message via HTTP handler
	reqBody := map[string]interface{}{
		"sender_id":   "user-2",
		"receiver_id": "broadcast",
		"type":        "text",
		"body":        "Acknowledged!",
		"reply_to_id": origMsg.ID,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdMsg database.Message
	if err := json.NewDecoder(rec.Body).Decode(&createdMsg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdMsg.ReplyToID == nil || *createdMsg.ReplyToID != origMsg.ID {
		t.Errorf("expected reply_to_id %s, got %v", origMsg.ID, createdMsg.ReplyToID)
	}
	if createdMsg.ReplyTo == nil || createdMsg.ReplyTo.Body != "Initial Announcement" {
		t.Errorf("expected ReplyTo to be populated with 'Initial Announcement', got %+v", createdMsg.ReplyTo)
	}

	// 3. Fetch messages via GET /api/messages
	getReq := httptest.NewRequest(http.MethodGet, "/api/messages?user_id=user-1&peer_id=broadcast", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET /api/messages, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var list []database.Message
	if err := json.NewDecoder(getRec.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode message list: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(list))
	}
	if list[1].ReplyTo == nil || list[1].ReplyTo.Body != "Initial Announcement" {
		t.Errorf("expected list[1].ReplyTo to be populated, got %+v", list[1].ReplyTo)
	}
}
