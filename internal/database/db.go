package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
	mu sync.RWMutex
}

type TrustedDevice struct {
	Token      string    `json:"token,omitempty"`
	DeviceName string    `json:"device_name"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	UserID     string    `json:"user_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type User struct {
	ID          string    `json:"id"`
	IPAddress   string    `json:"ip_address"`
	DisplayName string    `json:"display_name"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	IsOnline    bool      `json:"is_online"`
}

type CodeSnippet struct {
	MessageID   string `json:"message_id"`
	Language    string `json:"language"`
	CodeContent string `json:"code_content"`
}

type Transfer struct {
	MessageID     string    `json:"message_id"`
	FileName      string    `json:"file_name"`
	FileSize      int64     `json:"file_size"`
	FilePath      string    `json:"file_path,omitempty"`
	IsFolderZip   bool      `json:"is_folder_zip"`
	BurnOnRead    bool      `json:"burn_on_read"`
	ExpiresAt     time.Time `json:"expires_at"`
	DownloadCount int       `json:"download_count"`
	IsExpired     bool      `json:"is_expired"`
}

type ReactionGroup struct {
	Emoji     string   `json:"emoji"`
	Count     int      `json:"count"`
	UserIDs   []string `json:"user_ids"`
	UserNames []string `json:"user_names"`
}

type Message struct {
	ID         string          `json:"id"`
	SenderID   string          `json:"sender_id"`
	ReceiverID string          `json:"receiver_id"`
	Type       string          `json:"type"` // 'text', 'code', 'file'
	Body       string          `json:"body"`
	ReplyToID  *string         `json:"reply_to_id,omitempty"`
	ReplyTo    *Message        `json:"reply_to,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	Snippet    *CodeSnippet    `json:"snippet,omitempty"`
	Transfer   *Transfer       `json:"transfer,omitempty"`
	Reactions  []ReactionGroup `json:"reactions"`
}

func InitDB(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "devdrop.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Configure connection pool for SQLite WAL
	db.SetMaxOpenConns(1) // Single writer for SQLite
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Execute WAL PRAGMAs
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA synchronous=NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("Warning: pragma execution %q failed: %v", pragma, err)
		}
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		ip_address TEXT NOT NULL,
		display_name TEXT NOT NULL,
		last_seen_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		sender_id TEXT NOT NULL,
		receiver_id TEXT NOT NULL,
		type TEXT NOT NULL, -- 'text', 'code', 'file'
		body TEXT,
		reply_to_id TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(reply_to_id) REFERENCES messages(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS code_snippets (
		message_id TEXT PRIMARY KEY,
		language TEXT NOT NULL,
		code_content TEXT NOT NULL,
		FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS transfers (
		message_id TEXT PRIMARY KEY,
		file_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		file_path TEXT NOT NULL,
		is_folder_zip BOOLEAN NOT NULL DEFAULT 0,
		burn_on_read BOOLEAN NOT NULL DEFAULT 0,
		expires_at DATETIME NOT NULL,
		download_count INTEGER DEFAULT 0,
		FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS message_reactions (
		message_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		emoji TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		PRIMARY KEY (message_id, user_id, emoji),
		FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_reactions_message ON message_reactions(message_id);

	CREATE INDEX IF NOT EXISTS idx_messages_conversation 
	ON messages(sender_id, receiver_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_messages_receiver 
	ON messages(receiver_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at 
	ON messages(created_at);

	CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS trusted_devices (
		token TEXT PRIMARY KEY,
		device_name TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		user_id TEXT,
		created_at DATETIME NOT NULL,
		last_used_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_trusted_devices_token ON trusted_devices(token);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Migrate messages table if upgrading from previous version without reply_to_id
	_, _ = db.Exec("ALTER TABLE messages ADD COLUMN reply_to_id TEXT;")

	// Migrate message_reactions table if upgrading
	reactionsTableSchema := `
	CREATE TABLE IF NOT EXISTS message_reactions (
		message_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		emoji TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		PRIMARY KEY (message_id, user_id, emoji),
		FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_reactions_message ON message_reactions(message_id);
	`
	_, _ = db.Exec(reactionsTableSchema)

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

// User Operations
func (d *DB) UpsertUser(id, ip, displayName string) (*User, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	query := `
	INSERT INTO users (id, ip_address, display_name, last_seen_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		ip_address = excluded.ip_address,
		last_seen_at = excluded.last_seen_at
	RETURNING id, ip_address, display_name, last_seen_at;
	`
	var u User
	err := d.db.QueryRow(query, id, ip, displayName, now).Scan(&u.ID, &u.IPAddress, &u.DisplayName, &u.LastSeenAt)
	if err != nil {
		// Fallback for drivers where RETURNING might have quirks
		updateQuery := `
		INSERT INTO users (id, ip_address, display_name, last_seen_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			ip_address = excluded.ip_address,
			last_seen_at = excluded.last_seen_at;
		`
		if _, err2 := d.db.Exec(updateQuery, id, ip, displayName, now); err2 != nil {
			return nil, err2
		}
		return d.getUserUnlocked(id)
	}
	return &u, nil
}

func (d *DB) UpdateUserDisplayName(id, newName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	query := `UPDATE users SET display_name = ? WHERE id = ?;`
	_, err := d.db.Exec(query, newName, id)
	return err
}

func (d *DB) UpdateUserLastSeen(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	query := `UPDATE users SET last_seen_at = ? WHERE id = ?;`
	_, err := d.db.Exec(query, time.Now().UTC(), id)
	return err
}

func (d *DB) GetUser(id string) (*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getUserUnlocked(id)
}

func (d *DB) getUserUnlocked(id string) (*User, error) {
	query := `SELECT id, ip_address, display_name, last_seen_at FROM users WHERE id = ?;`
	var u User
	err := d.db.QueryRow(query, id).Scan(&u.ID, &u.IPAddress, &u.DisplayName, &u.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *DB) GetAllUsers() ([]User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT id, ip_address, display_name, last_seen_at FROM users ORDER BY last_seen_at DESC;`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.IPAddress, &u.DisplayName, &u.LastSeenAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Message Operations
func (d *DB) GetMessageByID(id string) (*Message, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getMessageByIDUnlocked(id)
}

func (d *DB) getMessageByIDUnlocked(id string) (*Message, error) {
	query := `
	SELECT m.id, m.sender_id, m.receiver_id, m.type, m.body, m.reply_to_id, m.created_at,
	       c.language, c.code_content,
	       t.file_name, t.file_size, t.file_path, t.is_folder_zip, t.burn_on_read, t.expires_at, t.download_count
	FROM messages m
	LEFT JOIN code_snippets c ON m.id = c.message_id
	LEFT JOIN transfers t ON m.id = t.message_id
	WHERE m.id = ?;
	`
	var m Message
	var (
		replyToID           sql.NullString
		cLang, cContent     sql.NullString
		tName, tPath        sql.NullString
		tSize               sql.NullInt64
		tIsZip, tBurnOnRead sql.NullBool
		tExpiresAt          sql.NullTime
		tDownloadCount      sql.NullInt64
	)
	err := d.db.QueryRow(query, id).Scan(
		&m.ID, &m.SenderID, &m.ReceiverID, &m.Type, &m.Body, &replyToID, &m.CreatedAt,
		&cLang, &cContent,
		&tName, &tSize, &tPath, &tIsZip, &tBurnOnRead, &tExpiresAt, &tDownloadCount,
	)
	if err != nil {
		return nil, err
	}
	if replyToID.Valid && replyToID.String != "" {
		m.ReplyToID = &replyToID.String
	}
	if m.Type == "code" && cLang.Valid {
		m.Snippet = &CodeSnippet{
			MessageID:   m.ID,
			Language:    cLang.String,
			CodeContent: cContent.String,
		}
	}
	if m.Type == "file" && tName.Valid {
		now := time.Now().UTC()
		isExpired := tExpiresAt.Valid && now.After(tExpiresAt.Time)
		m.Transfer = &Transfer{
			MessageID:     m.ID,
			FileName:      tName.String,
			FileSize:      tSize.Int64,
			FilePath:      tPath.String,
			IsFolderZip:   tIsZip.Bool,
			BurnOnRead:    tBurnOnRead.Bool,
			ExpiresAt:     tExpiresAt.Time,
			DownloadCount: int(tDownloadCount.Int64),
			IsExpired:     isExpired,
		}
	}
	m.Reactions = make([]ReactionGroup, 0)
	_ = d.populateReactionsUnlocked([]*Message{&m})
	return &m, nil
}

func (d *DB) SaveTextMessage(id, senderID, receiverID, body string, replyToID *string, createdAt time.Time) (*Message, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `INSERT INTO messages (id, sender_id, receiver_id, type, body, reply_to_id, created_at) VALUES (?, ?, ?, 'text', ?, ?, ?);`
	var replyIDVal any
	if replyToID != nil && *replyToID != "" {
		replyIDVal = *replyToID
	}
	_, err := d.db.Exec(query, id, senderID, receiverID, body, replyIDVal, createdAt)
	if err != nil {
		return nil, err
	}

	msg := &Message{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       "text",
		Body:       body,
		ReplyToID:  replyToID,
		CreatedAt:  createdAt,
		Reactions:  make([]ReactionGroup, 0),
	}

	if replyToID != nil && *replyToID != "" {
		if rm, err := d.getMessageByIDUnlocked(*replyToID); err == nil {
			msg.ReplyTo = rm
		}
	}

	return msg, nil
}

func (d *DB) SaveCodeMessage(id, senderID, receiverID, body, language, codeContent string, replyToID *string, createdAt time.Time) (*Message, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msgQuery := `INSERT INTO messages (id, sender_id, receiver_id, type, body, reply_to_id, created_at) VALUES (?, ?, ?, 'code', ?, ?, ?);`
	var replyIDVal any
	if replyToID != nil && *replyToID != "" {
		replyIDVal = *replyToID
	}
	if _, err := tx.Exec(msgQuery, id, senderID, receiverID, body, replyIDVal, createdAt); err != nil {
		return nil, err
	}

	codeQuery := `INSERT INTO code_snippets (message_id, language, code_content) VALUES (?, ?, ?);`
	if _, err := tx.Exec(codeQuery, id, language, codeContent); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	msg := &Message{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       "code",
		Body:       body,
		ReplyToID:  replyToID,
		CreatedAt:  createdAt,
		Reactions:  make([]ReactionGroup, 0),
		Snippet: &CodeSnippet{
			MessageID:   id,
			Language:    language,
			CodeContent: codeContent,
		},
	}

	if replyToID != nil && *replyToID != "" {
		if rm, err := d.getMessageByIDUnlocked(*replyToID); err == nil {
			msg.ReplyTo = rm
		}
	}

	return msg, nil
}

func (d *DB) SaveTransferMessage(
	id, senderID, receiverID, body, fileName string,
	fileSize int64, filePath string,
	isFolderZip, burnOnRead bool,
	expiresAt, createdAt time.Time,
	replyToID *string,
) (*Message, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msgQuery := `INSERT INTO messages (id, sender_id, receiver_id, type, body, reply_to_id, created_at) VALUES (?, ?, ?, 'file', ?, ?, ?);`
	var replyIDVal any
	if replyToID != nil && *replyToID != "" {
		replyIDVal = *replyToID
	}
	if _, err := tx.Exec(msgQuery, id, senderID, receiverID, body, replyIDVal, createdAt); err != nil {
		return nil, err
	}

	trQuery := `
	INSERT INTO transfers (message_id, file_name, file_size, file_path, is_folder_zip, burn_on_read, expires_at, download_count)
	VALUES (?, ?, ?, ?, ?, ?, ?, 0);
	`
	if _, err := tx.Exec(trQuery, id, fileName, fileSize, filePath, isFolderZip, burnOnRead, expiresAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	msg := &Message{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       "file",
		Body:       body,
		ReplyToID:  replyToID,
		CreatedAt:  createdAt,
		Reactions:  make([]ReactionGroup, 0),
		Transfer: &Transfer{
			MessageID:     id,
			FileName:      fileName,
			FileSize:      fileSize,
			FilePath:      filePath,
			IsFolderZip:   isFolderZip,
			BurnOnRead:    burnOnRead,
			ExpiresAt:     expiresAt,
			DownloadCount: 0,
		},
	}

	if replyToID != nil && *replyToID != "" {
		if rm, err := d.getMessageByIDUnlocked(*replyToID); err == nil {
			msg.ReplyTo = rm
		}
	}

	return msg, nil
}

func (d *DB) GetMessages(userA, userB string, limit int, beforeID string) ([]Message, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 40
	}

	var beforeCreatedAt time.Time
	var beforeRowID int64
	if beforeID != "" {
		err := d.db.QueryRow("SELECT rowid, created_at FROM messages WHERE id = ?", beforeID).Scan(&beforeRowID, &beforeCreatedAt)
		if err != nil {
			return []Message{}, false, nil
		}
	}

	var query string
	var rows *sql.Rows
	var err error

	// Query limit + 1 to accurately detect if more messages exist
	queryLimit := limit + 1

	selectCols := `
		SELECT m.id, m.sender_id, m.receiver_id, m.type, m.body, m.reply_to_id, m.created_at,
		       c.language, c.code_content,
		       t.file_name, t.file_size, t.file_path, t.is_folder_zip, t.burn_on_read, t.expires_at, t.download_count,
		       rm.id, rm.sender_id, rm.receiver_id, rm.type, rm.body, rm.created_at,
		       rc.language, rc.code_content,
		       rt.file_name, rt.file_size, rt.file_path, rt.is_folder_zip, rt.burn_on_read, rt.expires_at, rt.download_count
		FROM messages m
		LEFT JOIN code_snippets c ON m.id = c.message_id
		LEFT JOIN transfers t ON m.id = t.message_id
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		LEFT JOIN code_snippets rc ON rm.id = rc.message_id
		LEFT JOIN transfers rt ON rm.id = rt.message_id
	`

	if userB == "broadcast" || userB == "all" || userB == "" {
		if beforeID != "" {
			query = selectCols + `
			WHERE (m.receiver_id = 'all' OR m.receiver_id = 'broadcast')
			  AND (m.created_at < ? OR (m.created_at = ? AND m.rowid < ?))
			ORDER BY m.created_at DESC, m.rowid DESC
			LIMIT ?;
			`
			rows, err = d.db.Query(query, beforeCreatedAt, beforeCreatedAt, beforeRowID, queryLimit)
		} else {
			query = selectCols + `
			WHERE (m.receiver_id = 'all' OR m.receiver_id = 'broadcast')
			ORDER BY m.created_at DESC, m.rowid DESC
			LIMIT ?;
			`
			rows, err = d.db.Query(query, queryLimit)
		}
	} else {
		if beforeID != "" {
			query = selectCols + `
			WHERE ((m.sender_id = ? AND m.receiver_id = ?) OR (m.sender_id = ? AND m.receiver_id = ?))
			  AND (m.created_at < ? OR (m.created_at = ? AND m.rowid < ?))
			ORDER BY m.created_at DESC, m.rowid DESC
			LIMIT ?;
			`
			rows, err = d.db.Query(query, userA, userB, userB, userA, beforeCreatedAt, beforeCreatedAt, beforeRowID, queryLimit)
		} else {
			query = selectCols + `
			WHERE ((m.sender_id = ? AND m.receiver_id = ?) OR (m.sender_id = ? AND m.receiver_id = ?))
			ORDER BY m.created_at DESC, m.rowid DESC
			LIMIT ?;
			`
			rows, err = d.db.Query(query, userA, userB, userB, userA, queryLimit)
		}
	}

	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	messages := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		var (
			mReplyToID          sql.NullString
			cLang, cContent     sql.NullString
			tName, tPath        sql.NullString
			tSize               sql.NullInt64
			tIsZip, tBurnOnRead sql.NullBool
			tExpiresAt          sql.NullTime
			tDownloadCount      sql.NullInt64

			rmID, rmSender, rmReceiver, rmType, rmBody sql.NullString
			rmCreatedAt                                sql.NullTime
			rcLang, rcContent                          sql.NullString
			rtName, rtPath                             sql.NullString
			rtSize                                     sql.NullInt64
			rtIsZip, rtBurnOnRead                      sql.NullBool
			rtExpiresAt                                sql.NullTime
			rtDownloadCount                            sql.NullInt64
		)

		if err := rows.Scan(
			&m.ID, &m.SenderID, &m.ReceiverID, &m.Type, &m.Body, &mReplyToID, &m.CreatedAt,
			&cLang, &cContent,
			&tName, &tSize, &tPath, &tIsZip, &tBurnOnRead, &tExpiresAt, &tDownloadCount,
			&rmID, &rmSender, &rmReceiver, &rmType, &rmBody, &rmCreatedAt,
			&rcLang, &rcContent,
			&rtName, &rtSize, &rtPath, &rtIsZip, &rtBurnOnRead, &rtExpiresAt, &rtDownloadCount,
		); err != nil {
			return nil, false, err
		}

		if mReplyToID.Valid && mReplyToID.String != "" {
			m.ReplyToID = &mReplyToID.String
		}

		if m.Type == "code" && cLang.Valid {
			m.Snippet = &CodeSnippet{
				MessageID:   m.ID,
				Language:    cLang.String,
				CodeContent: cContent.String,
			}
		}

		if m.Type == "file" && tName.Valid {
			isExpired := tExpiresAt.Valid && now.After(tExpiresAt.Time)
			m.Transfer = &Transfer{
				MessageID:     m.ID,
				FileName:      tName.String,
				FileSize:      tSize.Int64,
				FilePath:      tPath.String,
				IsFolderZip:   tIsZip.Bool,
				BurnOnRead:    tBurnOnRead.Bool,
				ExpiresAt:     tExpiresAt.Time,
				DownloadCount: int(tDownloadCount.Int64),
				IsExpired:     isExpired,
			}
		}

		if rmID.Valid && rmID.String != "" {
			replyMsg := &Message{
				ID:         rmID.String,
				SenderID:   rmSender.String,
				ReceiverID: rmReceiver.String,
				Type:       rmType.String,
				Body:       rmBody.String,
				CreatedAt:  rmCreatedAt.Time,
			}
			if replyMsg.Type == "code" && rcLang.Valid {
				replyMsg.Snippet = &CodeSnippet{
					MessageID:   replyMsg.ID,
					Language:    rcLang.String,
					CodeContent: rcContent.String,
				}
			}
			if replyMsg.Type == "file" && rtName.Valid {
				replyExpired := rtExpiresAt.Valid && now.After(rtExpiresAt.Time)
				replyMsg.Transfer = &Transfer{
					MessageID:     replyMsg.ID,
					FileName:      rtName.String,
					FileSize:      rtSize.Int64,
					FilePath:      rtPath.String,
					IsFolderZip:   rtIsZip.Bool,
					BurnOnRead:    rtBurnOnRead.Bool,
					ExpiresAt:     rtExpiresAt.Time,
					DownloadCount: int(rtDownloadCount.Int64),
					IsExpired:     replyExpired,
				}
			}
			m.ReplyTo = replyMsg
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(messages) > limit {
		hasMore = true
		messages = messages[:limit]
	}

	// Reverse messages so they are returned in ascending chronological order (oldest to newest)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	msgPtrs := make([]*Message, len(messages))
	for i := range messages {
		msgPtrs[i] = &messages[i]
	}
	_ = d.populateReactionsUnlocked(msgPtrs)

	return messages, hasMore, nil
}

func (d *DB) GetTransfer(messageID string) (*Transfer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT message_id, file_name, file_size, file_path, is_folder_zip, burn_on_read, expires_at, download_count
	FROM transfers
	WHERE message_id = ?;
	`
	var t Transfer
	err := d.db.QueryRow(query, messageID).Scan(
		&t.MessageID, &t.FileName, &t.FileSize, &t.FilePath,
		&t.IsFolderZip, &t.BurnOnRead, &t.ExpiresAt, &t.DownloadCount,
	)
	if err != nil {
		return nil, err
	}
	t.IsExpired = time.Now().UTC().After(t.ExpiresAt)
	return &t, nil
}

func (d *DB) IncrementTransferDownload(messageID string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE transfers SET download_count = download_count + 1 WHERE message_id = ?;`
	if _, err := d.db.Exec(query, messageID); err != nil {
		return 0, err
	}

	var count int
	if err := d.db.QueryRow(`SELECT download_count FROM transfers WHERE message_id = ?;`, messageID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (d *DB) MarkTransferExpired(messageID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Set expires_at to 1970 and clear file_path so it cannot be served
	query := `UPDATE transfers SET expires_at = ?, file_path = '' WHERE message_id = ?;`
	_, err := d.db.Exec(query, time.Unix(0, 0).UTC(), messageID)
	return err
}

func (d *DB) GetExpiredTransfers(now time.Time) ([]Transfer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT message_id, file_name, file_size, file_path, is_folder_zip, burn_on_read, expires_at, download_count
	FROM transfers
	WHERE expires_at <= ? AND file_path != '';
	`
	rows, err := d.db.Query(query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expired []Transfer
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(
			&t.MessageID, &t.FileName, &t.FileSize, &t.FilePath,
			&t.IsFolderZip, &t.BurnOnRead, &t.ExpiresAt, &t.DownloadCount,
		); err != nil {
			return nil, err
		}
		t.IsExpired = true
		expired = append(expired, t)
	}
	return expired, rows.Err()
}

func (d *DB) populateReactionsUnlocked(messages []*Message) error {
	if len(messages) == 0 {
		return nil
	}

	msgMap := make(map[string][]*Message)
	for _, m := range messages {
		if m == nil {
			continue
		}
		if m.Reactions == nil {
			m.Reactions = make([]ReactionGroup, 0)
		}
		msgMap[m.ID] = append(msgMap[m.ID], m)
		if m.ReplyTo != nil {
			if m.ReplyTo.Reactions == nil {
				m.ReplyTo.Reactions = make([]ReactionGroup, 0)
			}
			msgMap[m.ReplyTo.ID] = append(msgMap[m.ReplyTo.ID], m.ReplyTo)
		}
	}

	if len(msgMap) == 0 {
		return nil
	}

	ids := make([]string, 0, len(msgMap))
	for id := range msgMap {
		ids = append(ids, id)
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT r.message_id, r.emoji, r.user_id, COALESCE(u.display_name, '')
		FROM message_reactions r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.message_id IN (%s)
		ORDER BY r.created_at ASC;
	`, strings.Join(placeholders, ","))

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	type rawGroupMap struct {
		emojiOrder []string
		groups     map[string]*ReactionGroup
	}
	msgReactions := make(map[string]*rawGroupMap)

	for rows.Next() {
		var mID, emoji, uID, uName string
		if err := rows.Scan(&mID, &emoji, &uID, &uName); err != nil {
			return err
		}
		if uName == "" {
			uName = uID
		}

		rgm, ok := msgReactions[mID]
		if !ok {
			rgm = &rawGroupMap{
				emojiOrder: make([]string, 0),
				groups:     make(map[string]*ReactionGroup),
			}
			msgReactions[mID] = rgm
		}

		grp, exists := rgm.groups[emoji]
		if !exists {
			rgm.emojiOrder = append(rgm.emojiOrder, emoji)
			grp = &ReactionGroup{
				Emoji:     emoji,
				Count:     0,
				UserIDs:   make([]string, 0),
				UserNames: make([]string, 0),
			}
			rgm.groups[emoji] = grp
		}
		grp.Count++
		grp.UserIDs = append(grp.UserIDs, uID)
		grp.UserNames = append(grp.UserNames, uName)
	}

	for mID, rgm := range msgReactions {
		reactionList := make([]ReactionGroup, 0, len(rgm.emojiOrder))
		for _, e := range rgm.emojiOrder {
			reactionList = append(reactionList, *rgm.groups[e])
		}
		for _, targetMsg := range msgMap[mID] {
			targetMsg.Reactions = reactionList
		}
	}

	return nil
}

func (d *DB) ToggleReaction(messageID, userID, emoji string) ([]ReactionGroup, bool, string, string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var senderID, receiverID string
	err := d.db.QueryRow("SELECT sender_id, receiver_id FROM messages WHERE id = ?", messageID).Scan(&senderID, &receiverID)
	if err != nil {
		return nil, false, "", "", err
	}

	var count int
	err = d.db.QueryRow("SELECT COUNT(*) FROM message_reactions WHERE message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).Scan(&count)
	if err != nil {
		return nil, false, "", "", err
	}

	added := false
	if count > 0 {
		_, err = d.db.Exec("DELETE FROM message_reactions WHERE message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji)
		if err != nil {
			return nil, false, "", "", err
		}
	} else {
		now := time.Now().UTC()
		_, err = d.db.Exec("INSERT INTO message_reactions (message_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?)", messageID, userID, emoji, now)
		if err != nil {
			return nil, false, "", "", err
		}
		added = true
	}

	query := `
		SELECT r.emoji, r.user_id, COALESCE(u.display_name, '')
		FROM message_reactions r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.message_id = ?
		ORDER BY r.created_at ASC;
	`
	rows, err := d.db.Query(query, messageID)
	if err != nil {
		return nil, added, senderID, receiverID, err
	}
	defer rows.Close()

	var emojiOrder []string
	groupsMap := make(map[string]*ReactionGroup)

	for rows.Next() {
		var em, uID, uName string
		if err := rows.Scan(&em, &uID, &uName); err != nil {
			return nil, added, senderID, receiverID, err
		}
		if uName == "" {
			uName = uID
		}
		grp, exists := groupsMap[em]
		if !exists {
			emojiOrder = append(emojiOrder, em)
			grp = &ReactionGroup{
				Emoji:     em,
				Count:     0,
				UserIDs:   make([]string, 0),
				UserNames: make([]string, 0),
			}
			groupsMap[em] = grp
		}
		grp.Count++
		grp.UserIDs = append(grp.UserIDs, uID)
		grp.UserNames = append(grp.UserNames, uName)
	}

	result := make([]ReactionGroup, 0, len(emojiOrder))
	for _, em := range emojiOrder {
		result = append(result, *groupsMap[em])
	}

	return result, added, senderID, receiverID, nil
}

// Authentication & Trusted Device Operations

// HasPassword checks if an access password has been configured
func (d *DB) HasPassword() (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var val string
	err := d.db.QueryRow("SELECT value FROM app_settings WHERE key = 'password_hash'").Scan(&val)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(val) != "", nil
}

// SyncPassword synchronizes the master password from CLI or environment variables.
// If plainPassword is empty, password protection is disabled.
// If plainPassword has changed, all existing trusted devices are revoked.
// If plainPassword is unchanged, existing trusted devices remain valid across restarts.
func (d *DB) SyncPassword(plainPassword string) error {
	plainPassword = strings.TrimSpace(plainPassword)
	if plainPassword == "" {
		d.mu.Lock()
		defer d.mu.Unlock()
		_, _ = d.db.Exec("DELETE FROM app_settings WHERE key = 'password_hash'")
		_, _ = d.db.Exec("DELETE FROM trusted_devices")
		return nil
	}

	// Check if matches existing configured password
	matches, err := d.VerifyPassword(plainPassword)
	if err == nil && matches {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	hashed, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	query := `
	INSERT INTO app_settings (key, value, updated_at)
	VALUES ('password_hash', ?, ?)
	ON CONFLICT(key) DO UPDATE SET
		value = excluded.value,
		updated_at = excluded.updated_at;
	`
	if _, err := d.db.Exec(query, string(hashed), now); err != nil {
		return err
	}

	// Invalidate previous trusted devices since password was changed
	_, _ = d.db.Exec("DELETE FROM trusted_devices")
	return nil
}

// SetPassword updates or creates the master password hash
func (d *DB) SetPassword(plainPassword string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	plainPassword = strings.TrimSpace(plainPassword)
	if plainPassword == "" {
		return fmt.Errorf("password cannot be empty")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	query := `
	INSERT INTO app_settings (key, value, updated_at)
	VALUES ('password_hash', ?, ?)
	ON CONFLICT(key) DO UPDATE SET
		value = excluded.value,
		updated_at = excluded.updated_at;
	`
	_, err = d.db.Exec(query, string(hashed), now)
	return err
}

// VerifyPassword compares the provided plain text password with the stored hash
func (d *DB) VerifyPassword(plainPassword string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var hash string
	err := d.db.QueryRow("SELECT value FROM app_settings WHERE key = 'password_hash'").Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(plainPassword))
	return err == nil, nil
}

// RegisterTrustedDevice generates a secure random token and registers the device
func (d *DB) RegisterTrustedDevice(deviceName, ipAddress, userAgent, userID string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random device token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	if strings.TrimSpace(deviceName) == "" {
		deviceName = "Trusted Device"
	}
	now := time.Now().UTC()

	query := `
	INSERT INTO trusted_devices (token, device_name, ip_address, user_agent, user_id, created_at, last_used_at)
	VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	_, err := d.db.Exec(query, token, deviceName, ipAddress, userAgent, userID, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to register trusted device: %w", err)
	}
	return token, nil
}

// ValidateTrustedDevice verifies if the token exists and updates last_used_at
func (d *DB) ValidateTrustedDevice(token string) (bool, *TrustedDevice, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
	SELECT token, device_name, ip_address, user_agent, COALESCE(user_id, ''), created_at, last_used_at
	FROM trusted_devices
	WHERE token = ?;
	`
	var dev TrustedDevice
	err := d.db.QueryRow(query, token).Scan(
		&dev.Token,
		&dev.DeviceName,
		&dev.IPAddress,
		&dev.UserAgent,
		&dev.UserID,
		&dev.CreatedAt,
		&dev.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	now := time.Now().UTC()
	_, _ = d.db.Exec("UPDATE trusted_devices SET last_used_at = ? WHERE token = ?", now, token)
	dev.LastUsedAt = now

	return true, &dev, nil
}

// RevokeTrustedDevice removes a single device token
func (d *DB) RevokeTrustedDevice(token string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("DELETE FROM trusted_devices WHERE token = ?", token)
	return err
}

// RevokeAllTrustedDevices removes all trusted devices, optionally keeping exceptToken
func (d *DB) RevokeAllTrustedDevices(exceptToken string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if strings.TrimSpace(exceptToken) == "" {
		_, err := d.db.Exec("DELETE FROM trusted_devices")
		return err
	}
	_, err := d.db.Exec("DELETE FROM trusted_devices WHERE token != ?", exceptToken)
	return err
}

// GetTrustedDevicesCount returns total active trusted devices
func (d *DB) GetTrustedDevicesCount() (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM trusted_devices").Scan(&count)
	return count, err
}

