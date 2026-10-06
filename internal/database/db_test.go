package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplyFunctionality(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devdrop_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := InitDB(tempDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	// 1. Save original text message
	msg1, err := db.SaveTextMessage("msg-1", "user-alice", "broadcast", "Hello LAN!", nil, now)
	if err != nil {
		t.Fatalf("failed to save msg1: %v", err)
	}
	if msg1.ReplyToID != nil {
		t.Errorf("expected msg1.ReplyToID to be nil, got %v", msg1.ReplyToID)
	}

	// 2. Save reply text message
	replyToID := "msg-1"
	msg2, err := db.SaveTextMessage("msg-2", "user-bob", "broadcast", "Replying to Hello!", &replyToID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("failed to save msg2: %v", err)
	}
	if msg2.ReplyToID == nil || *msg2.ReplyToID != "msg-1" {
		t.Errorf("expected msg2.ReplyToID to be msg-1, got %v", msg2.ReplyToID)
	}
	if msg2.ReplyTo == nil {
		t.Fatalf("expected msg2.ReplyTo to be populated, got nil")
	}
	if msg2.ReplyTo.Body != "Hello LAN!" {
		t.Errorf("expected msg2.ReplyTo.Body to be 'Hello LAN!', got %q", msg2.ReplyTo.Body)
	}

	// 3. Save reply code message replying to msg-2
	replyToID2 := "msg-2"
	msg3, err := db.SaveCodeMessage("msg-3", "user-charlie", "broadcast", "Check this snippet", "go", "fmt.Println(123)", &replyToID2, now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("failed to save msg3: %v", err)
	}
	if msg3.ReplyTo == nil || msg3.ReplyTo.Body != "Replying to Hello!" {
		t.Errorf("expected msg3.ReplyTo to reference msg-2, got %+v", msg3.ReplyTo)
	}

	// 4. Save reply transfer message replying to msg-3
	replyToID3 := "msg-3"
	msg4, err := db.SaveTransferMessage("msg-4", "user-alice", "broadcast", "Here is the archive", "bundle.zip", 1024, filepath.Join(tempDir, "bundle.zip"), true, false, now.Add(time.Hour), now.Add(3*time.Second), &replyToID3)
	if err != nil {
		t.Fatalf("failed to save msg4: %v", err)
	}
	if msg4.ReplyTo == nil || msg4.ReplyTo.Snippet == nil || msg4.ReplyTo.Snippet.Language != "go" {
		t.Errorf("expected msg4.ReplyTo to have code snippet with lang 'go', got %+v", msg4.ReplyTo)
	}

	// 5. Test GetMessages retrieves replied messages with full joined details
	msgs, err := db.GetMessages("user-alice", "broadcast", 50)
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// Verify msg-2 in list
	var foundMsg2 *Message
	var foundMsg4 *Message
	for i := range msgs {
		if msgs[i].ID == "msg-2" {
			foundMsg2 = &msgs[i]
		}
		if msgs[i].ID == "msg-4" {
			foundMsg4 = &msgs[i]
		}
	}

	if foundMsg2 == nil || foundMsg2.ReplyTo == nil || foundMsg2.ReplyTo.Body != "Hello LAN!" {
		t.Errorf("expected retrieved msg-2 to have ReplyTo populated with 'Hello LAN!', got %+v", foundMsg2)
	}
	if foundMsg4 == nil || foundMsg4.ReplyTo == nil || foundMsg4.ReplyTo.Snippet == nil || foundMsg4.ReplyTo.Snippet.Language != "go" {
		t.Errorf("expected retrieved msg-4 to have ReplyTo populated with snippet lang go, got %+v", foundMsg4)
	}
}
