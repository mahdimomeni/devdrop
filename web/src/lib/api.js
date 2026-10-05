// Client API and WebSocket management for DevDrop

export function getUserId() {
  let id = localStorage.getItem('devdrop_user_id');
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem('devdrop_user_id', id);
  }
  return id;
}

export function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function formatRelativeTime(dateStr) {
  const d = new Date(dateStr);
  const now = new Date();
  const diffMs = now - d;
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 10) return 'just now';
  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  return d.toLocaleDateString();
}

export function formatExpirationCountdown(expiresAtStr, burnOnRead, downloadCount) {
  if (burnOnRead) {
    if (downloadCount > 0) return { label: 'Burned (Downloaded)', expired: true, isBurn: true };
    return { label: 'Burn on read (1x)', expired: false, isBurn: true };
  }

  const exp = new Date(expiresAtStr);
  const now = new Date();
  const diffMs = exp - now;

  if (diffMs <= 0) {
    return { label: 'Expired', expired: true, isBurn: false };
  }

  const diffSec = Math.floor(diffMs / 1000);
  const hours = Math.floor(diffSec / 3600);
  const mins = Math.floor((diffSec % 3600) / 60);

  if (hours > 0) {
    return { label: `Expires in ${hours}h ${mins}m`, expired: false, isBurn: false };
  }
  if (mins > 0) {
    return { label: `Expires in ${mins}m`, expired: false, isBurn: false };
  }
  return { label: `Expires in ${diffSec}s`, expired: false, isBurn: false };
}

// REST API calls
export async function getProfile(userId) {
  const res = await fetch(`/api/profile?user_id=${encodeURIComponent(userId)}`);
  if (!res.ok) throw new Error('Failed to load profile');
  return res.json();
}

export async function updateDisplayName(userId, displayName) {
  const res = await fetch('/api/profile/name', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, display_name: displayName }),
  });
  if (!res.ok) throw new Error('Failed to update name');
  return res.json();
}

export async function getPeers() {
  const res = await fetch('/api/peers');
  if (!res.ok) throw new Error('Failed to load peers');
  return res.json();
}

export async function getMessages(userId, peerId) {
  const res = await fetch(`/api/messages?user_id=${encodeURIComponent(userId)}&peer_id=${encodeURIComponent(peerId)}`);
  if (!res.ok) throw new Error('Failed to load messages');
  return res.json();
}

export async function sendTextMessage(senderId, receiverId, body) {
  const res = await fetch('/api/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      sender_id: senderId,
      receiver_id: receiverId,
      type: 'text',
      body,
    }),
  });
  if (!res.ok) throw new Error('Failed to send message');
  return res.json();
}

export async function sendCodeMessage(senderId, receiverId, body, language, codeContent) {
  const res = await fetch('/api/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      sender_id: senderId,
      receiver_id: receiverId,
      type: 'code',
      body,
      language,
      code_content: codeContent,
    }),
  });
  if (!res.ok) throw new Error('Failed to send code');
  return res.json();
}

export async function uploadTransfer(formData, onProgress) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/upload');

    if (xhr.upload && onProgress) {
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          const percent = Math.round((e.loaded / e.total) * 100);
          onProgress(percent);
        }
      });
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText));
        } catch {
          resolve({ success: true });
        }
      } else {
        reject(new Error(xhr.responseText || 'Upload failed'));
      }
    };

    xhr.onerror = () => reject(new Error('Network error during upload'));
    xhr.send(formData);
  });
}

// WebSocket Connection Wrapper with auto-reconnect
export class DevDropSocket {
  constructor(userId, onEvent) {
    this.userId = userId;
    this.onEvent = onEvent;
    this.ws = null;
    this.reconnectTimeout = null;
    this.isConnected = false;
    this.connect();
  }

  connect() {
    if (this.ws) {
      try {
        this.ws.close();
      } catch {}
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/ws?user_id=${encodeURIComponent(this.userId)}`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      this.isConnected = true;
      if (this.onEvent) this.onEvent({ type: 'ws_open' });
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (this.onEvent) this.onEvent(data);
      } catch (err) {
        console.error('WS parse error:', err);
      }
    };

    this.ws.onclose = () => {
      this.isConnected = false;
      if (this.onEvent) this.onEvent({ type: 'ws_close' });
      // Schedule reconnect
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = setTimeout(() => this.connect(), 2000);
    };

    this.ws.onerror = () => {
      try {
        this.ws.close();
      } catch {}
    };
  }

  destroy() {
    clearTimeout(this.reconnectTimeout);
    if (this.ws) {
      try {
        this.ws.close();
      } catch {}
    }
  }
}
