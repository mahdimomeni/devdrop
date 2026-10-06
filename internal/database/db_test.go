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
	msgs, hasMore, err := db.GetMessages("user-alice", "broadcast", 50, "")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if hasMore {
		t.Errorf("expected hasMore to be false for all messages fetched, got true")
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

func TestGetMessagesPagination(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devdrop_page_test_*")
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
	// Insert 5 messages in chronological order
	for i := 1; i <= 5; i++ {
		msgID := "msg-" + string(rune('0'+i))
		body := "Message " + string(rune('0'+i))
		_, err := db.SaveTextMessage(msgID, "user-alice", "broadcast", body, nil, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("failed to save msg %d: %v", i, err)
		}
	}

	// 1. Initial fetch with limit=2 (should return the 2 most recent: msg-4, msg-5, in chronological order)
	page1, hasMore1, err := db.GetMessages("user-alice", "broadcast", 2, "")
	if err != nil {
		t.Fatalf("failed to get page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected 2 messages in page 1, got %d", len(page1))
	}
	if !hasMore1 {
		t.Errorf("expected hasMore to be true for page 1")
	}
	if page1[0].ID != "msg-4" || page1[1].ID != "msg-5" {
		t.Errorf("expected page1 to be [msg-4, msg-5], got [%s, %s]", page1[0].ID, page1[1].ID)
	}

	// 2. Fetch older messages before msg-4 with limit=2 (should return msg-2, msg-3)
	page2, hasMore2, err := db.GetMessages("user-alice", "broadcast", 2, page1[0].ID)
	if err != nil {
		t.Fatalf("failed to get page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("expected 2 messages in page 2, got %d", len(page2))
	}
	if !hasMore2 {
		t.Errorf("expected hasMore to be true for page 2")
	}
	if page2[0].ID != "msg-2" || page2[1].ID != "msg-3" {
		t.Errorf("expected page2 to be [msg-2, msg-3], got [%s, %s]", page2[0].ID, page2[1].ID)
	}

	// 3. Fetch older messages before msg-2 with limit=2 (should return msg-1, and hasMore=false)
	page3, hasMore3, err := db.GetMessages("user-alice", "broadcast", 2, page2[0].ID)
	if err != nil {
		t.Fatalf("failed to get page 3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("expected 1 message in page 3, got %d", len(page3))
	}
	if hasMore3 {
		t.Errorf("expected hasMore to be false for page 3")
	}
	if page3[0].ID != "msg-1" {
		t.Errorf("expected page3 to be [msg-1], got [%s]", page3[0].ID)
	}
}

func TestReactionFunctionality(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devdrop_reaction_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := InitDB(tempDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	// Create users
	_, err = db.UpsertUser("user-alice", "127.0.0.1", "Alice")
	if err != nil {
		t.Fatalf("failed to upsert user alice: %v", err)
	}
	_, err = db.UpsertUser("user-bob", "127.0.0.1", "Bob")
	if err != nil {
		t.Fatalf("failed to upsert user bob: %v", err)
	}

	// Create message
	now := time.Now().UTC()
	msg, err := db.SaveTextMessage("msg-react-1", "user-alice", "broadcast", "Welcome to LAN!", nil, now)
	if err != nil {
		t.Fatalf("failed to save message: %v", err)
	}
	if len(msg.Reactions) != 0 {
		t.Errorf("expected 0 initial reactions, got %d", len(msg.Reactions))
	}

	// Alice adds reaction 👍
	groups, added, senderID, receiverID, err := db.ToggleReaction("msg-react-1", "user-alice", "👍")
	if err != nil {
		t.Fatalf("failed to toggle reaction: %v", err)
	}
	if !added {
		t.Errorf("expected added=true, got false")
	}
	if senderID != "user-alice" || receiverID != "broadcast" {
		t.Errorf("expected sender=user-alice, receiver=broadcast, got %s, %s", senderID, receiverID)
	}
	if len(groups) != 1 || groups[0].Emoji != "👍" || groups[0].Count != 1 {
		t.Fatalf("expected 1 reaction group with count 1, got %+v", groups)
	}
	if len(groups[0].UserNames) != 1 || groups[0].UserNames[0] != "Alice" {
		t.Errorf("expected user name Alice, got %+v", groups[0].UserNames)
	}

	// Bob also reacts with 👍
	groups, added, _, _, err = db.ToggleReaction("msg-react-1", "user-bob", "👍")
	if err != nil {
		t.Fatalf("failed to toggle bob reaction: %v", err)
	}
	if !added {
		t.Errorf("expected added=true for Bob")
	}
	if len(groups) != 1 || groups[0].Count != 2 {
		t.Fatalf("expected count=2 for 👍, got %+v", groups)
	}

	// Bob reacts with 🔥
	groups, added, _, _, err = db.ToggleReaction("msg-react-1", "user-bob", "🔥")
	if err != nil {
		t.Fatalf("failed to toggle bob fire reaction: %v", err)
	}
	if !added {
		t.Errorf("expected added=true for Bob 🔥")
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 reaction groups, got %d", len(groups))
	}

	// Verify GetMessages loads these reactions
	msgs, _, err := db.GetMessages("user-alice", "broadcast", 10, "")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].Reactions) != 2 {
		t.Fatalf("expected 2 reactions in retrieved message, got %d", len(msgs[0].Reactions))
	}

	// Alice toggles 👍 off
	groups, added, _, _, err = db.ToggleReaction("msg-react-1", "user-alice", "👍")
	if err != nil {
		t.Fatalf("failed to toggle alice reaction off: %v", err)
	}
	if added {
		t.Errorf("expected added=false on toggle off, got true")
	}
	// 👍 should now have count 1 (Bob only)
	var thumbsGroup *ReactionGroup
	for i := range groups {
		if groups[i].Emoji == "👍" {
			thumbsGroup = &groups[i]
		}
	}
	if thumbsGroup == nil || thumbsGroup.Count != 1 {
		t.Fatalf("expected 👍 count=1, got %+v", thumbsGroup)
	}

	// Bob toggles 👍 off -> group should disappear
	groups, added, _, _, err = db.ToggleReaction("msg-react-1", "user-bob", "👍")
	if err != nil {
		t.Fatalf("failed to toggle bob reaction off: %v", err)
	}
	if added {
		t.Errorf("expected added=false")
	}
	if len(groups) != 1 || groups[0].Emoji != "🔥" {
		t.Fatalf("expected only 🔥 reaction left, got %+v", groups)
	}
}

func TestAuthAndTrustedDevices(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devdrop_auth_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := InitDB(tempDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	// 1. Initial state: has no password
	hasPass, err := db.HasPassword()
	if err != nil {
		t.Fatalf("unexpected error checking password: %v", err)
	}
	if hasPass {
		t.Errorf("expected HasPassword=false on new DB")
	}

	// 2. Set password
	err = db.SetPassword("supersecret123")
	if err != nil {
		t.Fatalf("failed to set password: %v", err)
	}

	hasPass, err = db.HasPassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasPass {
		t.Errorf("expected HasPassword=true after SetPassword")
	}

	// 3. Verify password
	ok, err := db.VerifyPassword("wrongpass")
	if err != nil {
		t.Fatalf("unexpected error on VerifyPassword: %v", err)
	}
	if ok {
		t.Errorf("expected VerifyPassword=false for wrong password")
	}

	ok, err = db.VerifyPassword("supersecret123")
	if err != nil {
		t.Fatalf("unexpected error on VerifyPassword: %v", err)
	}
	if !ok {
		t.Errorf("expected VerifyPassword=true for correct password")
	}

	// 4. Register trusted devices
	token1, err := db.RegisterTrustedDevice("Laptop Chrome", "192.168.1.10", "Mozilla/5.0", "user-1")
	if err != nil {
		t.Fatalf("failed to register device 1: %v", err)
	}
	if token1 == "" {
		t.Errorf("expected non-empty token")
	}

	token2, err := db.RegisterTrustedDevice("Phone Safari", "192.168.1.20", "Mobile Safari", "user-2")
	if err != nil {
		t.Fatalf("failed to register device 2: %v", err)
	}
	if token2 == "" {
		t.Errorf("expected non-empty token2")
	}

	count, err := db.GetTrustedDevicesCount()
	if err != nil || count != 2 {
		t.Fatalf("expected 2 trusted devices, got %d (err: %v)", count, err)
	}

	// 5. Validate device
	valid, dev, err := db.ValidateTrustedDevice(token1)
	if err != nil {
		t.Fatalf("failed to validate device: %v", err)
	}
	if !valid || dev == nil {
		t.Fatalf("expected device 1 to be valid")
	}
	if dev.DeviceName != "Laptop Chrome" || dev.IPAddress != "192.168.1.10" {
		t.Errorf("unexpected device data: %+v", dev)
	}

	// Invalid token check
	valid, _, err = db.ValidateTrustedDevice("invalid-fake-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Errorf("expected invalid token to return false")
	}

	// 6. Revoke single device
	err = db.RevokeTrustedDevice(token1)
	if err != nil {
		t.Fatalf("failed to revoke device: %v", err)
	}

	valid, _, _ = db.ValidateTrustedDevice(token1)
	if valid {
		t.Errorf("expected revoked device 1 to be invalid")
	}

	count, _ = db.GetTrustedDevicesCount()
	if count != 1 {
		t.Errorf("expected 1 trusted device remaining, got %d", count)
	}

	// 7. Revoke all devices
	err = db.RevokeAllTrustedDevices("")
	if err != nil {
		t.Fatalf("failed to revoke all: %v", err)
	}

	count, _ = db.GetTrustedDevicesCount()
	if count != 0 {
		t.Errorf("expected 0 trusted devices after revoke all, got %d", count)
	}

	// 8. Test SyncPassword from CLI / env
	err = db.SyncPassword("newcli123")
	if err != nil {
		t.Fatalf("failed to sync password: %v", err)
	}
	hasPass, _ = db.HasPassword()
	if !hasPass {
		t.Errorf("expected has_password=true after sync")
	}

	t1, _ := db.RegisterTrustedDevice("Device1", "127.0.0.1", "Agent", "user-1")

	// Sync with same password -> device remains valid
	err = db.SyncPassword("newcli123")
	if err != nil {
		t.Fatalf("sync error: %v", err)
	}
	valid, _, _ = db.ValidateTrustedDevice(t1)
	if !valid {
		t.Errorf("expected device to remain valid when password is unchanged")
	}

	// Sync with changed password -> old device revoked
	err = db.SyncPassword("changedpassword456")
	if err != nil {
		t.Fatalf("sync error: %v", err)
	}
	valid, _, _ = db.ValidateTrustedDevice(t1)
	if valid {
		t.Errorf("expected device to be revoked when password is changed")
	}

	// Sync with empty password -> password protection disabled
	err = db.SyncPassword("")
	if err != nil {
		t.Fatalf("sync error: %v", err)
	}
	hasPass, _ = db.HasPassword()
	if hasPass {
		t.Errorf("expected has_password=false when empty password synced")
	}
}


