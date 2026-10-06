package transfer

import (
	"archive/zip"
	"context"
	"devdrop/internal/database"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Dev-Ignore bloatware folder patterns
var DefaultDevIgnoreDirs = []string{
	".git",
	"node_modules",
	"target",
	"vendor",
	".idea",
	".vscode",
	"dist",
	"build",
	"__pycache__",
	".pytest_cache",
	".next",
	".svelte-kit",
}

type Broadcaster interface {
	BroadcastMessage(msg *database.Message)
	BroadcastFileExpired(messageID string)
}

type Manager struct {
	uploadDir string
	db        *database.DB
	hub       Broadcaster
}

func NewManager(uploadDir string, db *database.DB, hub Broadcaster) (*Manager, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory %s: %w", uploadDir, err)
	}

	return &Manager{
		uploadDir: uploadDir,
		db:        db,
		hub:       hub,
	}, nil
}

// StartCleanupWorker starts the background goroutine scanning for expired files
func (m *Manager) StartCleanupWorker(ctx context.Context) {
	// Run cleanup once immediately on startup
	m.cleanupExpired()

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.cleanupExpired()
			}
		}
	}()
}

func (m *Manager) cleanupExpired() {
	now := time.Now().UTC()
	expired, err := m.db.GetExpiredTransfers(now)
	if err != nil {
		log.Printf("[CleanupWorker] Error querying expired transfers: %v", err)
		return
	}

	for _, t := range expired {
		if t.FilePath != "" {
			if err := os.Remove(t.FilePath); err != nil && !os.IsNotExist(err) {
				log.Printf("[CleanupWorker] Failed to delete expired file %s: %v", t.FilePath, err)
			}
		}
		_ = m.db.MarkTransferExpired(t.MessageID)
		if m.hub != nil {
			m.hub.BroadcastFileExpired(t.MessageID)
		}
		log.Printf("[CleanupWorker] Cleaned up expired transfer message: %s (%s)", t.MessageID, t.FileName)
	}
}

// ShouldDevIgnore checks if any path segment matches the ignore list
func ShouldDevIgnore(relPath string) bool {
	// Normalize path separators to forward slash
	clean := filepath.ToSlash(relPath)
	parts := strings.Split(clean, "/")
	for _, part := range parts {
		for _, ignored := range DefaultDevIgnoreDirs {
			if strings.EqualFold(part, ignored) {
				return true
			}
		}
	}
	return false
}

// UploadResult contains information about the processed upload
type UploadResult struct {
	MessageID   string
	FileName    string
	FileSize    int64
	FilePath    string
	IsFolderZip bool
	BurnOnRead  bool
	ExpiresAt   time.Time
}

// HandleUpload streams multipart files to disk using a fixed 32KB buffer
func (m *Manager) HandleUpload(r *http.Request) (*UploadResult, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("invalid multipart form: %w", err)
	}

	var (
		senderID    string
		receiverID  string
		body        string
		replyToID   *string
		expiration  = "24h"
		burnOnRead  = false
		devIgnore   = true
		isFolder    = false
		folderName  = "archive"
		singleFile  *os.File
		zipFile     *os.File
		zipWriter   *zip.Writer
		outFilePath string
		outFileName string
		totalBytes  int64
		msgID       = uuid.New().String()
	)

	buffer := make([]byte, 32*1024) // Fixed 32KB buffer as required by specification

	defer func() {
		if singleFile != nil {
			_ = singleFile.Close()
		}
		if zipWriter != nil {
			_ = zipWriter.Close()
		}
		if zipFile != nil {
			_ = zipFile.Close()
		}
	}()

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading multipart part failed: %w", err)
		}

		formName := part.FormName()
		fileName := part.FileName()

		// Read form fields
		if fileName == "" {
			val, _ := io.ReadAll(part)
			strVal := strings.TrimSpace(string(val))
			switch formName {
			case "sender_id":
				senderID = strVal
			case "receiver_id":
				receiverID = strVal
			case "body":
				body = strVal
			case "reply_to_id":
				if strVal != "" {
					v := strVal
					replyToID = &v
				}
			case "expiration":
				expiration = strVal
			case "burn_on_read":
				burnOnRead = strVal == "true" || strVal == "1"
			case "dev_ignore":
				devIgnore = strVal == "true" || strVal == "1"
			case "is_folder":
				isFolder = strVal == "true" || strVal == "1"
			case "folder_name":
				if strVal != "" {
					folderName = strVal
				}
			}
			continue
		}

		// It is a file part
		relPath := fileName
		disp := part.Header.Get("Content-Disposition")
		if _, params, err := mime.ParseMediaType(disp); err == nil {
			if fn, ok := params["filename"]; ok && fn != "" {
				relPath = fn
			}
		}

		// If part has relative path encoded or transmitted in filename
		if isFolder {
			// Initialize zip file on disk if not already open
			if zipFile == nil {
				if !strings.HasSuffix(strings.ToLower(folderName), ".zip") {
					folderName += ".zip"
				}
				outFileName = folderName
				diskName := fmt.Sprintf("%s_%s", msgID, sanitizeFilename(folderName))
				outFilePath = filepath.Join(m.uploadDir, diskName)

				zf, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return nil, fmt.Errorf("failed to create zip file: %w", err)
				}
				zipFile = zf
				zipWriter = zip.NewWriter(zipFile)
			}

			// Check dev-ignore filter
			if devIgnore && ShouldDevIgnore(relPath) {
				// Discard stream without writing to disk
				_, _ = io.CopyBuffer(io.Discard, part, buffer)
				continue
			}

			// Prepare zip file header
			cleanRel := filepath.ToSlash(filepath.Clean(relPath))
			header := &zip.FileHeader{
				Name:     cleanRel,
				Method:   zip.Deflate,
				Modified: time.Now().UTC(),
			}

			entryWriter, err := zipWriter.CreateHeader(header)
			if err != nil {
				return nil, fmt.Errorf("failed to create zip entry for %s: %w", cleanRel, err)
			}

			// Stream entry with 32KB buffer
			n, err := io.CopyBuffer(entryWriter, part, buffer)
			if err != nil {
				return nil, fmt.Errorf("streaming file to zip entry failed: %w", err)
			}
			totalBytes += n

		} else {
			// Single file upload
			if singleFile == nil {
				outFileName = fileName
				diskName := fmt.Sprintf("%s_%s", msgID, sanitizeFilename(fileName))
				outFilePath = filepath.Join(m.uploadDir, diskName)

				sf, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return nil, fmt.Errorf("failed to create target file: %w", err)
				}
				singleFile = sf

				n, err := io.CopyBuffer(singleFile, part, buffer)
				if err != nil {
					return nil, fmt.Errorf("streaming single file to disk failed: %w", err)
				}
				totalBytes = n
			} else {
				// Discard any additional files if not in folder mode
				_, _ = io.CopyBuffer(io.Discard, part, buffer)
			}
		}
	}

	// Close zip writers to finalize zip structure
	if zipWriter != nil {
		if err := zipWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to finalize zip: %w", err)
		}
		zipWriter = nil
	}
	if zipFile != nil {
		_ = zipFile.Close()
		zipFile = nil
	}
	if singleFile != nil {
		_ = singleFile.Close()
		singleFile = nil
	}

	if outFilePath == "" {
		return nil, errors.New("no file content uploaded")
	}

	// Calculate actual file size on disk
	fi, err := os.Stat(outFilePath)
	if err == nil {
		totalBytes = fi.Size()
	}

	// Calculate expiration timestamp
	expiresAt := calculateExpiration(expiration)

	// Save to DB
	createdAt := time.Now().UTC()
	msg, err := m.db.SaveTransferMessage(
		msgID, senderID, receiverID, body, outFileName,
		totalBytes, outFilePath, isFolder, burnOnRead,
		expiresAt, createdAt, replyToID,
	)
	if err != nil {
		_ = os.Remove(outFilePath)
		return nil, fmt.Errorf("failed to save transfer record: %w", err)
	}

	// Broadcast to clients via WebSocket
	if m.hub != nil {
		m.hub.BroadcastMessage(msg)
	}

	return &UploadResult{
		MessageID:   msgID,
		FileName:    outFileName,
		FileSize:    totalBytes,
		FilePath:    outFilePath,
		IsFolderZip: isFolder,
		BurnOnRead:  burnOnRead,
		ExpiresAt:   expiresAt,
	}, nil
}

// ServeDownload streams file to client, handles Burn-On-Read
func (m *Manager) ServeDownload(w http.ResponseWriter, r *http.Request, messageID string) {
	transfer, err := m.db.GetTransfer(messageID)
	if err != nil {
		http.Error(w, "File transfer not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()
	if now.After(transfer.ExpiresAt) || transfer.FilePath == "" {
		http.Error(w, "This file transfer has expired or is no longer available.", http.StatusGone)
		return
	}

	file, err := os.Open(transfer.FilePath)
	if err != nil {
		http.Error(w, "File not accessible on server", http.StatusNotFound)
		return
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		http.Error(w, "Cannot stat file", http.StatusInternalServerError)
		return
	}

	// Set headers
	encodedName := url.PathEscape(transfer.FileName)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))

	// Increment download count in DB
	_, _ = m.db.IncrementTransferDownload(messageID)

	// Stream with 32KB buffer
	buffer := make([]byte, 32*1024)
	_, _ = io.CopyBuffer(w, file, buffer)

	// If Burn on Read: delete file immediately after transfer completes!
	if transfer.BurnOnRead {
		_ = file.Close()
		_ = os.Remove(transfer.FilePath)
		_ = m.db.MarkTransferExpired(messageID)
		if m.hub != nil {
			m.hub.BroadcastFileExpired(messageID)
		}
		log.Printf("[BurnOnRead] Transfer %s (%s) burned after download.", messageID, transfer.FileName)
	}
}

func sanitizeFilename(name string) string {
	clean := filepath.Base(name)
	clean = strings.ReplaceAll(clean, "..", "")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	clean = strings.ReplaceAll(clean, " ", "_")
	if clean == "" || clean == "." {
		return "file"
	}
	return clean
}

func calculateExpiration(exp string) time.Time {
	now := time.Now().UTC()
	switch exp {
	case "burn":
		// Burn on read: default 24h safety limit if never downloaded
		return now.Add(24 * time.Hour)
	case "1h":
		return now.Add(1 * time.Hour)
	case "6h":
		return now.Add(6 * time.Hour)
	case "24h":
		return now.Add(24 * time.Hour)
	default:
		// Attempt duration parse
		if d, err := time.ParseDuration(exp); err == nil && d > 0 {
			return now.Add(d)
		}
		return now.Add(24 * time.Hour)
	}
}
