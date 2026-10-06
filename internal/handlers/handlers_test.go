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

func TestGetMessagesPaginationHandler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devdrop_handler_page_test_*")
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

	now := time.Now().UTC()
	for i := 1; i <= 3; i++ {
		id := "msg-h-" + string(rune('0'+i))
		_, err := db.SaveTextMessage(id, "user-1", "broadcast", "Content "+string(rune('0'+i)), nil, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("failed to save msg %d: %v", i, err)
		}
	}

	// 1. Initial request with limit=2 (should return msg-h-2, msg-h-3)
	req1 := httptest.NewRequest(http.MethodGet, "/api/messages?user_id=user-1&peer_id=broadcast&limit=2", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-Has-More") != "true" {
		t.Errorf("expected X-Has-More to be 'true', got %q", rec1.Header().Get("X-Has-More"))
	}

	var list1 []database.Message
	if err := json.NewDecoder(rec1.Body).Decode(&list1); err != nil {
		t.Fatalf("failed to decode list1: %v", err)
	}
	if len(list1) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(list1))
	}
	if list1[0].ID != "msg-h-2" || list1[1].ID != "msg-h-3" {
		t.Errorf("expected [msg-h-2, msg-h-3], got [%s, %s]", list1[0].ID, list1[1].ID)
	}

	// 2. Fetch older with before=msg-h-2
	req2 := httptest.NewRequest(http.MethodGet, "/api/messages?user_id=user-1&peer_id=broadcast&limit=2&before="+list1[0].ID, nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("X-Has-More") != "false" {
		t.Errorf("expected X-Has-More to be 'false', got %q", rec2.Header().Get("X-Has-More"))
	}

	var list2 []database.Message
	if err := json.NewDecoder(rec2.Body).Decode(&list2); err != nil {
		t.Fatalf("failed to decode list2: %v", err)
	}
	if len(list2) != 1 {
		t.Fatalf("expected 1 message, got %d", len(list2))
	}
	if list2[0].ID != "msg-h-1" {
		t.Errorf("expected msg-h-1, got %s", list2[0].ID)
	}
}

func TestLinkPreviewEndpoint(t *testing.T) {
	// Spin up a mock web server with OG tags
	mockTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta property="og:site_name" content="DevDrop Docs" />
			<meta property="og:title" content="Instant LAN Sharing" />
			<meta property="og:description" content="Super fast local file sharing" />
			<meta property="og:image" content="/preview.png" />
			<title>Fallback</title>
		</head>
		<body></body>
		</html>
		`
		w.Write([]byte(html))
	}))
	defer mockTarget.Close()

	tempDir, err := os.MkdirTemp("", "devdrop_preview_test_*")
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

	// 1. Request with missing url
	reqMissing := httptest.NewRequest(http.MethodGet, "/api/preview", nil)
	recMissing := httptest.NewRecorder()
	r.ServeHTTP(recMissing, reqMissing)
	if recMissing.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing url, got %d", recMissing.Code)
	}

	// 2. Request with valid mock URL
	reqValid := httptest.NewRequest(http.MethodGet, "/api/preview?url="+mockTarget.URL, nil)
	recValid := httptest.NewRecorder()
	r.ServeHTTP(recValid, reqValid)

	if recValid.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recValid.Code, recValid.Body.String())
	}

	var data map[string]interface{}
	if err := json.NewDecoder(recValid.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode preview json: %v", err)
	}

	if data["title"] != "Instant LAN Sharing" {
		t.Errorf("expected title 'Instant LAN Sharing', got %v", data["title"])
	}
	if data["site_name"] != "DevDrop Docs" {
		t.Errorf("expected site_name 'DevDrop Docs', got %v", data["site_name"])
	}
	if data["description"] != "Super fast local file sharing" {
		t.Errorf("expected description 'Super fast local file sharing', got %v", data["description"])
	}
	if data["image"] != mockTarget.URL+"/preview.png" {
		t.Errorf("expected image %s, got %v", mockTarget.URL+"/preview.png", data["image"])
	}
}

