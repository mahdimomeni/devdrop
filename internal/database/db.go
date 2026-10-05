package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
	mu sync.RWMutex
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

type Message struct {
	ID         string       `json:"id"`
	SenderID   string       `json:"sender_id"`
	ReceiverID string       `json:"receiver_id"`
	Type       string       `json:"type"` // 'text', 'code', 'file'
	Body       string       `json:"body"`
	CreatedAt  time.Time    `json:"created_at"`
	Snippet    *CodeSnippet `json:"snippet,omitempty"`
	Transfer   *Transfer    `json:"transfer,omitempty"`
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
		created_at DATETIME NOT NULL
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

	CREATE INDEX IF NOT EXISTS idx_messages_conversation 
	ON messages(sender_id, receiver_id, created_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

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
func (d *DB) SaveTextMessage(id, senderID, receiverID, body string, createdAt time.Time) (*Message, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `INSERT INTO messages (id, sender_id, receiver_id, type, body, created_at) VALUES (?, ?, ?, 'text', ?, ?);`
	_, err := d.db.Exec(query, id, senderID, receiverID, body, createdAt)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       "text",
		Body:       body,
		CreatedAt:  createdAt,
	}, nil
}

func (d *DB) SaveCodeMessage(id, senderID, receiverID, body, language, codeContent string, createdAt time.Time) (*Message, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msgQuery := `INSERT INTO messages (id, sender_id, receiver_id, type, body, created_at) VALUES (?, ?, ?, 'code', ?, ?);`
	if _, err := tx.Exec(msgQuery, id, senderID, receiverID, body, createdAt); err != nil {
		return nil, err
	}

	codeQuery := `INSERT INTO code_snippets (message_id, language, code_content) VALUES (?, ?, ?);`
	if _, err := tx.Exec(codeQuery, id, language, codeContent); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Message{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       "code",
		Body:       body,
		CreatedAt:  createdAt,
		Snippet: &CodeSnippet{
			MessageID:   id,
			Language:    language,
			CodeContent: codeContent,
		},
	}, nil
}

func (d *DB) SaveTransferMessage(
	id, senderID, receiverID, body, fileName string,
	fileSize int64, filePath string,
	isFolderZip, burnOnRead bool,
	expiresAt, createdAt time.Time,
) (*Message, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msgQuery := `INSERT INTO messages (id, sender_id, receiver_id, type, body, created_at) VALUES (?, ?, ?, 'file', ?, ?);`
	if _, err := tx.Exec(msgQuery, id, senderID, receiverID, body, createdAt); err != nil {
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

	return &Message{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       "file",
		Body:       body,
		CreatedAt:  createdAt,
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
	}, nil
}

func (d *DB) GetMessages(userA, userB string, limit int) ([]Message, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	// Support both direct 1-to-1 conversation and broadcast (receiver_id = 'all')
	var query string
	var rows *sql.Rows
	var err error

	if userB == "broadcast" || userB == "all" || userB == "" {
		query = `
		SELECT m.id, m.sender_id, m.receiver_id, m.type, m.body, m.created_at,
		       c.language, c.code_content,
		       t.file_name, t.file_size, t.file_path, t.is_folder_zip, t.burn_on_read, t.expires_at, t.download_count
		FROM messages m
		LEFT JOIN code_snippets c ON m.id = c.message_id
		LEFT JOIN transfers t ON m.id = t.message_id
		WHERE m.receiver_id = 'all' OR m.receiver_id = 'broadcast'
		ORDER BY m.created_at ASC
		LIMIT ?;
		`
		rows, err = d.db.Query(query, limit)
	} else {
		query = `
		SELECT m.id, m.sender_id, m.receiver_id, m.type, m.body, m.created_at,
		       c.language, c.code_content,
		       t.file_name, t.file_size, t.file_path, t.is_folder_zip, t.burn_on_read, t.expires_at, t.download_count
		FROM messages m
		LEFT JOIN code_snippets c ON m.id = c.message_id
		LEFT JOIN transfers t ON m.id = t.message_id
		WHERE (m.sender_id = ? AND m.receiver_id = ?)
		   OR (m.sender_id = ? AND m.receiver_id = ?)
		ORDER BY m.created_at ASC
		LIMIT ?;
		`
		rows, err = d.db.Query(query, userA, userB, userB, userA, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	var messages []Message
	for rows.Next() {
		var m Message
		var (
			cLang, cContent                         sql.NullString
			tName, tPath                            sql.NullString
			tSize                                   sql.NullInt64
			tIsZip, tBurnOnRead                     sql.NullBool
			tExpiresAt                              sql.NullTime
			tDownloadCount                          sql.NullInt64
		)

		if err := rows.Scan(
			&m.ID, &m.SenderID, &m.ReceiverID, &m.Type, &m.Body, &m.CreatedAt,
			&cLang, &cContent,
			&tName, &tSize, &tPath, &tIsZip, &tBurnOnRead, &tExpiresAt, &tDownloadCount,
		); err != nil {
			return nil, err
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

		messages = append(messages, m)
	}

	return messages, rows.Err()
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
