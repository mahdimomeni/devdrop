// Client API and WebSocket management for DevDrop

// Fallback-safe UUID generator for non-secure contexts (e.g. HTTP over LAN IP)
export function generateUUID() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, (c) =>
      (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16)
    );
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// Fallback-safe Clipboard copy for non-secure contexts (e.g. HTTP over LAN IP)
export async function copyToClipboard(text) {
  if (navigator?.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fallback if permission denied or error
    }
  }

  try {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.top = '0';
    textArea.style.left = '0';
    textArea.style.width = '2em';
    textArea.style.height = '2em';
    textArea.style.padding = '0';
    textArea.style.border = 'none';
    textArea.style.outline = 'none';
    textArea.style.boxShadow = 'none';
    textArea.style.background = 'transparent';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    const successful = document.execCommand('copy');
    textArea.remove();
    return successful;
  } catch (err) {
    console.error('Clipboard copy failed:', err);
    return false;
  }
}

export function getUserId() {
  let id = localStorage.getItem('devdrop_user_id');
  if (!id) {
    id = generateUUID();
    localStorage.setItem('devdrop_user_id', id);
  }
  return id;
}

export function getAuthToken() {
  return localStorage.getItem('devdrop_auth_token') || '';
}

export function setAuthToken(token) {
  if (token) {
    localStorage.setItem('devdrop_auth_token', token);
  } else {
    localStorage.removeItem('devdrop_auth_token');
  }
}

export function clearAuthToken() {
  localStorage.removeItem('devdrop_auth_token');
}

let authRequiredCallback = null;
export function onAuthRequired(cb) {
  authRequiredCallback = cb;
}

export function authHeaders(headers = {}) {
  const token = getAuthToken();
  const h = { ...headers };
  if (token) {
    h['Authorization'] = `Bearer ${token}`;
    h['X-Device-Token'] = token;
  }
  return h;
}

export async function customFetch(url, options = {}) {
  const opts = {
    ...options,
    headers: authHeaders(options.headers || {}),
  };
  const res = await fetch(url, opts);
  if (res.status === 401) {
    if (authRequiredCallback) {
      authRequiredCallback();
    }
  }
  return res;
}

export function detectDeviceName() {
  if (typeof navigator === 'undefined') return 'Web Browser';
  const ua = navigator.userAgent || '';
  let os = 'Device';
  if (ua.includes('Win')) os = 'Windows PC';
  else if (ua.includes('Mac')) os = 'Mac';
  else if (ua.includes('Linux')) os = 'Linux';
  else if (ua.includes('Android')) os = 'Android';
  else if (ua.includes('iPhone') || ua.includes('iPad')) os = 'iOS Device';

  let browser = 'Browser';
  if (ua.includes('Firefox')) browser = 'Firefox';
  else if (ua.includes('Edg')) browser = 'Edge';
  else if (ua.includes('Chrome')) browser = 'Chrome';
  else if (ua.includes('Safari')) browser = 'Safari';

  return `${browser} on ${os}`;
}

export function getDownloadUrl(messageId) {
  const token = getAuthToken();
  if (token) {
    return `/api/transfers/${encodeURIComponent(messageId)}/download?token=${encodeURIComponent(token)}`;
  }
  return `/api/transfers/${encodeURIComponent(messageId)}/download`;
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

// Auth API calls
export async function checkAuthStatus() {
  const res = await customFetch('/api/auth/status');
  if (!res.ok) throw new Error('Failed to check auth status');
  return res.json();
}

export async function loginWithPassword(password, trustDevice = true, deviceName = '', userId = '') {
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      password,
      trust_device: trustDevice,
      device_name: deviceName || detectDeviceName(),
      user_id: userId || getUserId(),
    }),
  });
  if (!res.ok) {
    const err = await res.text().catch(() => 'Invalid password');
    throw new Error(err || 'Invalid password');
  }
  const data = await res.json();
  if (data.token) {
    setAuthToken(data.token);
  }
  return data;
}

export async function logoutDevice() {
  try {
    await customFetch('/api/auth/logout', { method: 'POST' });
  } finally {
    clearAuthToken();
  }
}

export async function getAuthDevices() {
  const res = await customFetch('/api/auth/devices');
  if (!res.ok) throw new Error('Failed to get devices');
  return res.json();
}

// REST API calls
export async function getProfile(userId) {
  const res = await customFetch(`/api/profile?user_id=${encodeURIComponent(userId)}`);
  if (!res.ok) throw new Error('Failed to load profile');
  return res.json();
}

export async function updateDisplayName(userId, displayName) {
  const res = await customFetch('/api/profile/name', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, display_name: displayName }),
  });
  if (!res.ok) throw new Error('Failed to update name');
  return res.json();
}

export async function getPeers() {
  const res = await customFetch('/api/peers');
  if (!res.ok) throw new Error('Failed to load peers');
  return res.json();
}

export async function getMessages(userId, peerId, limit = 40, before = null) {
  let url = `/api/messages?user_id=${encodeURIComponent(userId)}&peer_id=${encodeURIComponent(peerId)}&limit=${limit}`;
  if (before) {
    url += `&before=${encodeURIComponent(before)}`;
  }
  const res = await customFetch(url);
  if (!res.ok) throw new Error('Failed to load messages');
  const messages = await res.json();
  const rawHeader = res.headers.get('X-Has-More') || res.headers.get('x-has-more');
  const hasMore = rawHeader !== null ? rawHeader.toLowerCase() === 'true' : (messages && messages.length >= limit);
  return { messages: messages || [], hasMore };
}

export async function sendTextMessage(senderId, receiverId, body, replyToId = null) {
  const payload = {
    sender_id: senderId,
    receiver_id: receiverId,
    type: 'text',
    body,
  };
  if (replyToId) payload.reply_to_id = replyToId;

  const res = await customFetch('/api/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to send message');
  return res.json();
}

export async function sendCodeMessage(senderId, receiverId, body, language, codeContent, replyToId = null) {
  const payload = {
    sender_id: senderId,
    receiver_id: receiverId,
    type: 'code',
    body,
    language,
    code_content: codeContent,
  };
  if (replyToId) payload.reply_to_id = replyToId;

  const res = await customFetch('/api/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to send code');
  return res.json();
}

export async function toggleMessageReaction(messageId, userId, emoji) {
  const res = await customFetch(`/api/messages/${encodeURIComponent(messageId)}/reactions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, emoji }),
  });
  if (!res.ok) {
    const errText = await res.text().catch(() => 'Failed to toggle reaction');
    throw new Error(errText || 'Failed to toggle reaction');
  }
  return res.json();
}

// Client-side cache for link previews to prevent redundant requests
const linkPreviewCache = new Map();
const pendingPreviewRequests = new Map();

export async function fetchLinkPreview(url) {
  if (!url) return null;
  if (linkPreviewCache.has(url)) {
    return linkPreviewCache.get(url);
  }
  if (pendingPreviewRequests.has(url)) {
    return pendingPreviewRequests.get(url);
  }

  const reqPromise = (async () => {
    try {
      const res = await customFetch(`/api/preview?url=${encodeURIComponent(url)}`);
      if (!res.ok) {
        linkPreviewCache.set(url, null);
        return null;
      }
      const data = await res.json();
      linkPreviewCache.set(url, data);
      return data;
    } catch (err) {
      console.warn('Failed to load link preview for', url, err);
      linkPreviewCache.set(url, null);
      return null;
    } finally {
      pendingPreviewRequests.delete(url);
    }
  })();

  pendingPreviewRequests.set(url, reqPromise);
  return reqPromise;
}

export async function uploadTransfer(formData, onProgress) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/upload');

    const token = getAuthToken();
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      xhr.setRequestHeader('X-Device-Token', token);
    }

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
    const token = getAuthToken();
    let wsUrl = `${protocol}//${host}/ws?user_id=${encodeURIComponent(this.userId)}`;
    if (token) {
      wsUrl += `&token=${encodeURIComponent(token)}`;
    }

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
