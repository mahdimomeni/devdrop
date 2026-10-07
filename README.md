# DevDrop — Internal LAN Developer Collaboration Service

> **Zero-Config, Self-Hosted Local Network Collaboration Tool**  
> Built with **Go** (embedded SQLite WAL + WebSockets) & **Svelte** (CodeMirror 6 + Tailwind CSS). Distributed as a **single, self-contained executable**.

---

## 🌟 Highlights

DevDrop runs on a central LAN workstation or server with **zero configuration** required. Team members on the same Wi-Fi or office network simply navigate to the server's IP address to chat, drop folders, share code snippets with diffing, and transfer files ephemerally.

- **Zero Config & Instant Identity:** No passwords or user signups. Monikers are automatically assigned (e.g. `Dev-192.168.1.55` or `HyperGopher`) and users can click to customize their display handle anytime.
- **Real-Time LAN Presence:** Persistent WebSockets track online/offline peers in real-time, displaying active status dots, IP addresses, and unread message badges.
- **Syntax-Aware Code Snippet Drawer:** Powered by CodeMirror 6 with dark mode (`One Dark`), line numbers, multi-language support (Go, C#, Java, TypeScript/JavaScript, Python, Markdown with chat preview, JSON, YAML, SQL, Shell, Plaintext), linting & diagnostic markers, a **Format / Beautify** button, 1-click **Copy Raw**, and **Side-by-Side Diff View** for comparing revisions.
- **Streaming Folder Zip & Ephemeral Storage:**
  - **Dev-Ignore Filter:** Toggle enabled by default to strip out common bloatware directories: `.git`, `node_modules`, `target`, `vendor`, `.idea`, `.vscode`, `dist`, `build`.
  - **Streaming Zip to Disk:** Incoming folders stream directly into an `archive/zip` writer backed by an `os.File` in `./data/uploads/` using **fixed-size $32\text{ KB}$ buffers**. RAM consumption stays constant regardless of whether you drop a 10MB or 2GB repository.
  - **Ephemeral Lifecycle & Burn-On-Read:** Transfers support customizable TTLs: **Burn on Read** (instantly destroyed from disk after first download), 1 hour, 6 hours, or 24 hours (default). A background worker scans every 5 minutes to purge expired files.
- **Smart Paste (`Ctrl+V` / `Cmd+V`):**
  - Pasting an image directly uploads and attaches it with an inline preview.
  - Pasting multi-line code triggers a toast: *"Multi-line code detected. Open in Code Editor?"*.
- **Single Binary Distribution:** The compiled Svelte SPA is embedded into the Go binary using `embed.FS`, creating a single executable with zero external runtime dependencies.

---

## 🏗️ Architecture & Technical Stack

```
                          DevDrop Central LAN Node
                 ┌────────────────────────────────────────┐
                 │        Go 1.22+ HTTP Server            │
                 │   (Embed FS: Svelte SPA + Assets)      │
                 └──────┬───────────────┬─────────────────┘
                        │               │
            WebSocket   │               │   REST API
         (gorilla/ws)   │               │   (/api/upload, /api/messages)
                        ▼               ▼
                 ┌──────────────┐ ┌───────────────────────┐
                 │ Presence Hub │ │  Transfer Manager     │
                 │ (Broadcast / │ │  - Fixed 32KB Buffers │
                 │ 1-to-1 Route)│ │  - Streaming Zip Disk │
                 └──────┬───────┘ │  - Ephemeral Cleaner  │
                        │         └───────────┬───────────┘
                        ▼                     ▼
                 ┌────────────────────────────────────────┐
                 │       Embedded SQLite (WAL Mode)       │
                 │   modernc.org/sqlite (Pure Go, no CGO) │
                 └────────────────────────────────────────┘
```

### 1. Database Schema
Uses `modernc.org/sqlite` with WAL mode enabled (`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`):

```sql
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
```

---

## 📂 Project Directory Structure

```
.
├── cmd/
│   └── server/
│       └── main.go          # CLI entrypoint, LAN IP banner, SPA router
├── internal/
│   ├── database/            # SQLite migrations, queries, and connection pool
│   ├── hub/                 # WebSocket hub, peer presence, moniker generator
│   ├── transfer/            # 32KB streaming zip creator, cleaner goroutine
│   └── handlers/            # REST API endpoints & WebSocket upgrader
├── web/                     # Svelte 5 frontend source
│   ├── embed.go             # Go embed.FS embedding web/dist into binary
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api.js       # Client API, WebSocket & countdown formatting
│   │   │   ├── ChatView.svelte      # Main chat, drag-drop HUD, smart paste
│   │   │   ├── CodeEditor.svelte    # CodeMirror 6 drawer + beautifier
│   │   │   ├── DiffModal.svelte     # Side-by-side snippet diff view
│   │   │   ├── FileCard.svelte      # Ephemeral file bubble, countdown chip
│   │   │   └── PeerList.svelte      # Sidebar, status dots, profile edit
│   │   ├── App.svelte       # Top-level view coordinator
│   │   └── main.js
│   ├── package.json
│   └── vite.config.js
├── Makefile                 # Targets: build-web, build-server, run, clean
└── README.md
```

---

## 🚀 Quickstart & Usage

### 1. Build and Run via Makefile
```bash
# Build Svelte frontend and compile single self-contained Go executable
make all

# Start DevDrop server
./devdrop
```

### 2. Run Directly with Go
```bash
# Start server with default port 8080 and storage in ./data
go run ./cmd/server
```

When started, DevDrop detects your local LAN network adapters and prints accessible URLs:
```
  ┌──────────────────────────────────────────────────────────┐
  │                      DevDrop LAN                         │
  │    Zero-Config Developer Collaboration Service           │
  └──────────────────────────────────────────────────────────┘
   > Local:    http://localhost:8080
   > Network:  http://192.168.1.55:8080
```

### 3. Command Line Flags & Environment Variables
| Flag | Env Variable | Default | Description |
|------|-------------|---------|-------------|
| `-port` | `PORT` | `8080` | Port to listen on |
| `-data` | `DATA_DIR` | `./data` | Directory for SQLite DB and upload files |
| `-password` | `DEVDROP_PASSWORD` / `PASSWORD` | *Empty* | Access password for the LAN node (managed exclusively via CLI or environment variable) |
| `-tls` | `TLS` | `false` | Enable auto-generated self-signed HTTPS/TLS for secure PWA installation |
| `-cert` | `TLS_CERT` | *Empty* | Path to custom TLS certificate file (e.g. from mkcert) |
| `-key` | `TLS_KEY` | *Empty* | Path to custom TLS private key file |

---

## 🔒 Access Password & Trusted Devices

DevDrop supports access protection with seamless trusted device management:
1. **Host-Controlled Password**: The password cannot be created or changed from the browser UI by any visitor. It is configured and changed exclusively on the server host via `-password <secret>` or `DEVDROP_PASSWORD=<secret>`. If no password is provided, the node runs with open access.
2. **First-Time Visit Authentication**: When a user on the LAN accesses the node for the first time, they are prompted to enter the password. With **Trust this device** checked (default), a cryptographically secure device token is issued and saved to their browser.
3. **Subsequent Visits**: On returning visits, the browser automatically authenticates with the trusted device token without prompting for the password again.
4. **Security & Device Management**: Click the **Security (Shield)** icon in the header to view trust status, device name, lock the current device (to re-require the password), and see the total number of trusted devices on the node.
5. **Rotating the Password**: When the host changes the password via CLI or environment variable, existing device trusts are automatically revoked so all devices must enter the new password on their next visit.

---

## 💡 Developer Workflows

### 📁 Sending Project Folders with Dev-Ignore
1. Click **Folder** in the chat footer or drag and drop an entire folder onto the chat window.
2. Dev-Ignore is enabled by default, stripping `.git/`, `node_modules/`, `target/`, etc.
3. The server streams files directly into a `.zip` archive on disk using a 32KB buffer.

### 💻 Code Snippet Sharing & Diffing
1. Click **Add Code** (or paste multi-line code to get auto-prompted).
2. Choose language (Go, Python, TypeScript, Markdown, etc.) and click **Format / Beautify**.
3. Once sent, peers can click **Copy Raw**, click **Compare Diff** on consecutive snippets to see side-by-side line additions and deletions, or click **Preview** on Markdown snippets to switch seamlessly between syntax-highlighted code and rendered Markdown.

### 🔥 Ephemeral Transfers & Burn-On-Read
- Select **Burn on Read (1x)** from the TTL selector.
- The recipient downloads the file once; upon download completion, the server immediately purges the file from disk and notifies all connected clients that the file has self-destructed.

---

## 📱 Progressive Web App (PWA) — Install on Phone & PC

DevDrop is a full **Progressive Web App (PWA)**, allowing developers to install it as a native desktop or mobile application directly from the browser without any app store.

### 🤖 Installing on Android (Why Chrome says "This app cannot be installed")
Mobile Chrome **strictly requires a Secure Context (HTTPS or localhost)** to install PWAs. Over plain HTTP on a local LAN IP (e.g. `http://192.10.105.53:8080`), Chrome flags the connection with a `⚠️` warning and disables the install button by default.

**Option A: 30-Second Chrome Flag (No tools or certs needed):**
1. On your phone, open Chrome and navigate to:
   ```
   chrome://flags/#unsafely-treat-insecure-origin-as-secure
   ```
2. Set the flag to **Enabled**.
3. In the input box, paste your DevDrop LAN address (e.g. `http://192.10.105.53:8080`).
4. Tap **Relaunch** at the bottom of Chrome.
5. Refresh DevDrop. The `⚠️` icon disappears, and tapping **Install** will install DevDrop immediately to your home screen!

**Option B: Run DevDrop with Built-in HTTPS:**
Start DevDrop with `-tls` to serve over HTTPS:
```bash
./devdrop -tls
```
Or provide trusted certificates (e.g. via [mkcert](https://github.com/FiloSottile/mkcert)):
```bash
./devdrop -cert /path/to/cert.pem -key /path/to/key.pem
```

### 💻 Installing on PC & Mac (Chrome, Edge, Brave)
1. Open DevDrop in your browser (e.g. `http://localhost:8080` or `http://<lan-ip>:8080`).
2. Click the **Install App** button in the header or in the left sidebar footer (or click the install icon **⊕** in the browser address bar).
3. Confirm **Install**.
4. DevDrop launches in a dedicated, frameless window with:
   - Dedicated taskbar/dock icon and window frame (no browser URL bar or tabs).
   - Fast startup with pre-cached offline application shell.
   - Native OS desktop notifications for mentions and file transfers.

### 🍏 Installing on iPhone & iPad (iOS Safari)
1. Open DevDrop in **Safari** on your iOS device.
2. Tap the **Share** button (box with upward arrow $\Box \uparrow$) in Safari's bottom navigation bar.
3. Scroll down and select **Add to Home Screen** ($\boxplus$).
4. Tap **Add** in the top-right corner.
5. DevDrop appears on your home screen with its custom icon and runs in full-screen standalone mode.

### 🔄 Automatic Updates
DevDrop's Service Worker checks for updates in the background. When a new version is deployed to the LAN node, an update banner appears automatically: *"A new version of DevDrop is ready to install!"* with a 1-click **Update Now** button.

