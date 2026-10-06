// Notification service for DevDrop
// Handles Web Audio chimes, browser desktop notifications, in-app notification toasts, and tab title unread badges.

import { formatBytes } from './api.js';
import { isUserMentioned } from './mentionUtils.js';

const SETTINGS_KEY = 'devdrop_notification_settings';

/**
 * @typedef {Object} NotificationSettings
 * @property {boolean} enabled - Master toggle for all notifications
 * @property {boolean} soundEnabled - Whether audio chime plays
 * @property {boolean} desktopEnabled - Whether desktop notifications should display
 * @property {'dms_mentions'|'all'} scope - Which messages trigger notifications
 */

export function getDefaultSettings() {
  return {
    enabled: true,
    soundEnabled: true,
    desktopEnabled: true,
    scope: 'dms_mentions',
  };
}

export function loadNotificationSettings() {
  if (typeof localStorage === 'undefined') return getDefaultSettings();
  try {
    const raw = localStorage.getItem(SETTINGS_KEY);
    if (!raw) return getDefaultSettings();
    return { ...getDefaultSettings(), ...JSON.parse(raw) };
  } catch (err) {
    console.warn('Failed to parse notification settings:', err);
    return getDefaultSettings();
  }
}

export function saveNotificationSettings(settings) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
  } catch (err) {
    console.error('Failed to save notification settings:', err);
  }
}

export function isNotificationSupported() {
  return typeof window !== 'undefined' && 'Notification' in window;
}

export function getNotificationPermission() {
  if (!isNotificationSupported()) return 'unsupported';
  return Notification.permission; // 'granted' | 'denied' | 'default'
}

export async function requestNotificationPermission() {
  if (!isNotificationSupported()) return 'unsupported';
  try {
    const perm = await Notification.requestPermission();
    return perm;
  } catch (err) {
    console.warn('Failed to request notification permission:', err);
    return getNotificationPermission();
  }
}

// Web Audio API synthesized chimes
let audioCtx = null;

function getAudioContext() {
  if (typeof window === 'undefined') return null;
  const AudioContextClass = window.AudioContext || window.webkitAudioContext;
  if (!AudioContextClass) return null;

  if (!audioCtx) {
    try {
      audioCtx = new AudioContextClass();
    } catch {
      return null;
    }
  }

  if (audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {});
  }
  return audioCtx;
}

// Pre-warm audio context on user gesture
if (typeof window !== 'undefined') {
  const resumeAudio = () => {
    if (audioCtx && audioCtx.state === 'suspended') {
      audioCtx.resume().catch(() => {});
    }
  };
  window.addEventListener('pointerdown', resumeAudio, { once: false, passive: true });
  window.addEventListener('keydown', resumeAudio, { once: false, passive: true });
}

let lastChimeTime = 0;

/**
 * Plays a synthesized notification chime using Web Audio API.
 * @param {'default'|'mention'|'test'} type
 */
export function playNotificationChime(type = 'default') {
  const now = Date.now();
  // Debounce rapid bursts within 80ms
  if (now - lastChimeTime < 80) return;
  lastChimeTime = now;

  try {
    const ctx = getAudioContext();
    if (!ctx) return;
    if (ctx.state === 'suspended') {
      ctx.resume().catch(() => {});
    }

    const t = ctx.currentTime;

    if (type === 'mention') {
      // 3-tone bright harmonic chime: E5 (659.25Hz), G#5 (830.61Hz), B5 (987.77Hz)
      const freqs = [659.25, 830.61, 987.77];
      freqs.forEach((freq, idx) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'sine';
        osc.frequency.setValueAtTime(freq, t + idx * 0.07);

        gain.gain.setValueAtTime(0, t + idx * 0.07);
        gain.gain.linearRampToValueAtTime(0.18, t + idx * 0.07 + 0.015);
        gain.gain.exponentialRampToValueAtTime(0.001, t + idx * 0.07 + 0.2);

        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.start(t + idx * 0.07);
        osc.stop(t + idx * 0.07 + 0.22);
      });
    } else {
      // Dual-tone gentle modern chime: D5 (587.33Hz), A5 (880Hz)
      const freqs = [587.33, 880.0];
      freqs.forEach((freq, idx) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'sine';
        osc.frequency.setValueAtTime(freq, t + idx * 0.08);

        gain.gain.setValueAtTime(0, t + idx * 0.08);
        gain.gain.linearRampToValueAtTime(0.15, t + idx * 0.08 + 0.015);
        gain.gain.exponentialRampToValueAtTime(0.001, t + idx * 0.08 + 0.24);

        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.start(t + idx * 0.08);
        osc.stop(t + idx * 0.08 + 0.26);
      });
    }
  } catch (err) {
    console.warn('Audio chime playback error:', err);
  }
}

/**
 * Triggers a desktop OS notification if permitted.
 * @param {Object} options
 * @param {string} options.title
 * @param {string} options.body
 * @param {string} [options.icon]
 * @param {string} [options.tag]
 * @param {Function} [options.onClick]
 * @returns {Notification|null}
 */
export function showDesktopNotification({ title, body, icon = '/vite.svg', tag, onClick }) {
  if (!isNotificationSupported()) return null;
  if (Notification.permission !== 'granted') return null;

  try {
    const notification = new Notification(title, {
      body,
      icon,
      tag: tag || `devdrop-${Date.now()}`,
      silent: true, // Audio handled by playNotificationChime for consistent LAN tone
    });

    if (onClick) {
      notification.onclick = (e) => {
        try {
          e.preventDefault();
          window.focus();
          onClick();
          notification.close();
        } catch (err) {
          console.warn('Notification click error:', err);
        }
      };
    }

    setTimeout(() => {
      try {
        notification.close();
      } catch {}
    }, 7000);

    return notification;
  } catch (err) {
    console.warn('Desktop notification dispatch failed:', err);
    return null;
  }
}

/**
 * Formats a message's notification title, preview, and metadata.
 * @param {Object} message
 * @param {Object|null} currentUser
 * @param {Array} peers
 * @returns {Object}
 */
export function formatNotificationDetails(message, currentUser, peers = []) {
  const sender = peers.find((p) => p.id === message.sender_id);
  const senderName = sender?.display_name || 'A peer';
  const isDM = message.receiver_id === currentUser?.id;
  const isMention = isUserMentioned(message.body, currentUser, peers);

  let title = `New message from ${senderName}`;
  let subtitle = isDM ? 'Direct Message' : 'LAN Broadcast';

  if (isMention) {
    title = `@${senderName} mentioned you`;
    subtitle = isDM ? 'Direct Mention' : 'Broadcast Mention';
  } else if (!isDM) {
    title = `${senderName} in #LAN Broadcast`;
  }

  let body = '';
  if (message.type === 'text') {
    body = message.body || 'New message';
  } else if (message.type === 'code') {
    const lang = message.snippet?.language || 'code';
    const comment = message.body ? `${message.body}: ` : '';
    const codeSnippet = message.snippet?.code_content
      ? message.snippet.code_content.slice(0, 60).replace(/\s+/g, ' ')
      : '';
    body = `${comment}[${lang}] ${codeSnippet}`.trim();
  } else if (message.type === 'file') {
    const fileName = message.transfer?.file_name || 'file';
    const size = message.transfer?.file_size ? formatBytes(message.transfer.file_size) : '';
    const isFolder = message.transfer?.is_folder_zip;
    body = isFolder ? `Folder: ${fileName}` : `File: ${fileName}${size ? ` (${size})` : ''}`;
  }

  if (body.length > 100) {
    body = body.slice(0, 97) + '...';
  }

  return {
    title,
    subtitle,
    body,
    senderName,
    senderId: message.sender_id,
    isDM,
    isMention,
    isCode: message.type === 'code',
    isFile: message.type === 'file',
  };
}

/**
 * Determines whether a message warrants notification and builds notification parameters.
 * @param {Object} message
 * @param {Object} ctx
 * @param {string} ctx.userId
 * @param {Object|null} ctx.currentUser
 * @param {Array} ctx.peers
 * @param {string} ctx.selectedPeerId
 * @param {boolean} ctx.isAppFocused
 * @param {NotificationSettings} ctx.settings
 * @returns {Object}
 */
export function shouldNotify(message, {
  userId,
  currentUser,
  peers = [],
  selectedPeerId = 'broadcast',
  isAppFocused = true,
  settings = getDefaultSettings(),
}) {
  if (!settings || !settings.enabled) {
    return { shouldNotify: false };
  }

  // Never notify for messages sent by this client
  if (!message || message.sender_id === userId) {
    return { shouldNotify: false };
  }

  const isMention = isUserMentioned(message.body, currentUser, peers);
  const isDM = message.receiver_id === userId;
  const isBroadcast =
    message.receiver_id === 'broadcast' ||
    message.receiver_id === 'all' ||
    !message.receiver_id;

  // Filter based on scope
  if (settings.scope === 'dms_mentions') {
    if (!isDM && !isMention) {
      return { shouldNotify: false };
    }
  }

  // Check if user is currently looking at this channel
  const isCurrentView =
    (selectedPeerId === 'broadcast' && isBroadcast) ||
    (selectedPeerId !== 'broadcast' && selectedPeerId === message.sender_id);

  // If user has the window focused and is in the active chat, suppress popups
  const isActivelyViewing = isAppFocused && isCurrentView;

  const details = formatNotificationDetails(message, currentUser, peers);
  const targetPeerId = isBroadcast ? 'broadcast' : message.sender_id;

  return {
    shouldNotify: true,
    isActivelyViewing,
    shouldShowDesktop: !isActivelyViewing && settings.desktopEnabled,
    shouldPlaySound: settings.soundEnabled,
    shouldShowInAppToast: !isCurrentView,
    details,
    targetPeerId,
  };
}

let originalTitle = typeof document !== 'undefined' ? document.title : 'DevDrop';

/**
 * Updates document tab title with unread badge counter.
 * @param {number} totalUnreads
 * @param {string} [latestSender]
 */
export function updateDocumentTitle(totalUnreads, latestSender = null) {
  if (typeof document === 'undefined') return;
  if (!originalTitle || originalTitle.startsWith('(')) {
    originalTitle = 'DevDrop — LAN Developer Collaboration';
  }

  if (totalUnreads > 0) {
    if (latestSender) {
      document.title = `(${totalUnreads}) ${latestSender} • DevDrop`;
    } else {
      document.title = `(${totalUnreads}) DevDrop`;
    }
  } else {
    document.title = originalTitle;
  }
}
