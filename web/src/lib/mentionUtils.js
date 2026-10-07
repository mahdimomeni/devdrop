// Utility functions for mentions detection, autocomplete, and message segmentation
import { cleanTrailingPunctuation, normalizeHref, URL_REGEX } from './linkUtils.js';

function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\\/]/g, '\\$&');
}

/**
 * Strips trailing sentence punctuation from a mention name unless the full name matches a known peer.
 * Handles trailing commas, exclamation points, colons, dots, and unbalanced closing brackets/parentheses.
 * @param {string} rawName
 * @param {string[]} knownNames
 * @returns {{ name: string, trailing: string }}
 */
export function cleanTrailingMentionPunctuation(rawName, knownNames = []) {
  let name = rawName;
  let trailing = '';

  // If full rawName matches a known name exactly (case-insensitive), keep it
  const isKnown = knownNames.some((k) => k && k.toLowerCase() === rawName.toLowerCase());
  if (isKnown) {
    return { name, trailing };
  }

  // Strip trailing punctuation
  const matchPunct = name.match(/[.,!?;:]+$/);
  if (matchPunct) {
    trailing = matchPunct[0];
    name = name.slice(0, -trailing.length);
  }

  // Unbalanced closing brackets/parentheses
  while (name.endsWith(')') || name.endsWith(']')) {
    const lastChar = name.slice(-1);
    trailing = lastChar + trailing;
    name = name.slice(0, -1);
  }

  return { name, trailing };
}

/**
 * Parses message text into segments of text, clickable links, and styled mention pills.
 *
 * @param {string} text
 * @param {Array} peers
 * @param {Object|null} currentUser
 * @returns {Array<{type: 'text'|'link'|'mention', text: string, href?: string, name?: string, isAll?: boolean, isMe?: boolean, peer?: Object|null}>}
 */
export function parseMessageSegments(text, peers = [], currentUser = null) {
  if (!text) return [];

  const spans = [];

  // 1. Detect all URL spans first so mentions inside URLs (e.g. twitter.com/@user) are not broken
  const urlRegex = new RegExp(URL_REGEX.source, 'gi');
  let match;
  while ((match = urlRegex.exec(text)) !== null) {
    const rawMatch = match[0];
    const startIndex = match.index;
    const { url } = cleanTrailingPunctuation(rawMatch);
    if (url && (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('www.'))) {
      spans.push({
        type: 'link',
        start: startIndex,
        end: startIndex + url.length,
        text: url,
        href: normalizeHref(url),
      });
    }
  }

  // 2. Build list of mention candidates for precise matching (longest display names first)
  const candidateNames = new Set(['all', 'everyone']);
  if (currentUser?.display_name) candidateNames.add(currentUser.display_name.trim());
  if (Array.isArray(peers)) {
    for (const p of peers) {
      if (p?.display_name) candidateNames.add(p.display_name.trim());
    }
  }

  const knownList = Array.from(candidateNames).filter(Boolean);
  const sortedNames = [...knownList].sort((a, b) => b.length - a.length);

  const escapedNames = sortedNames.map(escapeRegex).join('|');
  const mentionPattern = escapedNames.length > 0
    ? '(?:' + escapedNames + '|[a-zA-Z0-9_\\-\\.]+)'
    : '[a-zA-Z0-9_\\-\\.]+';

  // Look for mention token preceded by start-of-string or non-alphanumeric character (not [a-zA-Z0-9_])
  const mentionRegex = new RegExp('(?:^|[^a-zA-Z0-9_])(@(' + mentionPattern + '))', 'gi');

  while ((match = mentionRegex.exec(text)) !== null) {
    const fullMatch = match[0];
    const atWithPrefix = match[1];
    const rawName = match[2];
    const atIndex = match.index + (fullMatch.length - atWithPrefix.length);

    // Clean trailing sentence punctuation if any
    const { name } = cleanTrailingMentionPunctuation(rawName, knownList);
    const endIndex = atIndex + 1 + name.length;

    // Check overlap with any URL span
    const overlapsUrl = spans.some((s) => s.type === 'link' && atIndex >= s.start && atIndex < s.end);
    if (!overlapsUrl && name.length > 0) {
      const lowerName = name.toLowerCase();
      const isAll = lowerName === 'all' || lowerName === 'everyone';
      const matchedPeer = Array.isArray(peers)
        ? peers.find((p) => p.display_name?.toLowerCase() === lowerName) || null
        : null;

      const isMe =
        isAll ||
        (currentUser &&
          (lowerName === currentUser.display_name?.toLowerCase() ||
            (matchedPeer && matchedPeer.id === currentUser.id)));

      spans.push({
        type: 'mention',
        start: atIndex,
        end: endIndex,
        text: '@' + name,
        name: name,
        isAll,
        isMe: Boolean(isMe),
        peer: matchedPeer,
      });
    }

    // Step ahead safely without skipping adjacent matches
    mentionRegex.lastIndex = atIndex + 1;
  }

  // Sort all spans in reading order
  spans.sort((a, b) => a.start - b.start);

  // Filter overlapping spans
  const finalSpans = [];
  let lastEnd = 0;
  for (const span of spans) {
    if (span.start >= lastEnd) {
      finalSpans.push(span);
      lastEnd = span.end;
    }
  }

  // Construct final token segments
  const segments = [];
  let currentIndex = 0;
  for (const span of finalSpans) {
    if (span.start > currentIndex) {
      segments.push({
        type: 'text',
        text: text.slice(currentIndex, span.start),
      });
    }
    segments.push(span);
    currentIndex = span.end;
  }
  if (currentIndex < text.length) {
    segments.push({
      type: 'text',
      text: text.slice(currentIndex),
    });
  }

  return segments;
}

/**
 * Checks whether text mentions the current user or everyone (@all, @everyone).
 * @param {string} text
 * @param {Object|null} currentUser
 * @param {Array} [peers=[]]
 * @returns {boolean}
 */
export function isUserMentioned(text, currentUser, peers = []) {
  if (!text || !currentUser) return false;
  const segments = parseMessageSegments(text, peers, currentUser);
  return segments.some((s) => s.type === 'mention' && s.isMe);
}

/**
 * Inspects text before the cursor to see if a mention autocomplete trigger is active.
 * @param {string} text
 * @param {number} cursorPos
 * @returns {{ isOpen: boolean, query: string, atIndex: number }}
 */
export function extractMentionContext(text, cursorPos) {
  if (cursorPos < 1 || !text) return { isOpen: false, query: '', atIndex: -1 };
  const textBefore = text.slice(0, cursorPos);
  const lastAtIndex = textBefore.lastIndexOf('@');
  if (lastAtIndex === -1) return { isOpen: false, query: '', atIndex: -1 };

  // Validate prefix preceding '@'
  const charBefore = lastAtIndex > 0 ? textBefore[lastAtIndex - 1] : ' ';
  const isValidPrefix = /[\s\(\[\{\"'~*<>]/.test(charBefore);
  if (!isValidPrefix) return { isOpen: false, query: '', atIndex: -1 };

  const query = textBefore.slice(lastAtIndex + 1);
  if (/\s/.test(query) || query.length > 30) {
    return { isOpen: false, query: '', atIndex: -1 };
  }

  return { isOpen: true, query, atIndex: lastAtIndex };
}

/**
 * Filters and ranks mention candidates (broadcast @all and peers) based on the user's query.
 * @param {Array} peers
 * @param {Object|null} currentUser
 * @param {string} query
 * @returns {Array<{ id: string, name: string, displayName: string, subtitle: string, isAll: boolean, isOnline: boolean, isMe: boolean, ip?: string, peer?: Object }>}
 */
export function filterMentionCandidates(peers = [], currentUser = null, query = '') {
  const q = (query || '').toLowerCase().trim();
  const list = [];

  // 1. @all broadcast entry
  const matchesAll = !q || 'all'.startsWith(q) || 'everyone'.startsWith(q);
  if (matchesAll) {
    list.push({
      id: 'broadcast-all',
      name: 'all',
      displayName: 'all',
      subtitle: 'Notify all peers on LAN',
      isAll: true,
      isOnline: true,
      ip: 'broadcast',
    });
  }

  // 2. Peer entries
  const peerList = Array.isArray(peers) ? peers : [];
  const peerItems = peerList
    .filter((p) => p && p.display_name)
    .map((peer) => {
      const isMe = peer.id === currentUser?.id;
      return {
        id: peer.id,
        name: peer.display_name,
        displayName: peer.display_name,
        subtitle: isMe ? `${peer.ip_address} • You` : peer.ip_address,
        isAll: false,
        isOnline: Boolean(peer.is_online),
        isMe,
        ip: peer.ip_address,
        peer,
      };
    })
    .filter((item) => {
      if (!q) return true;
      return (
        item.name.toLowerCase().includes(q) ||
        (item.ip && item.ip.toLowerCase().includes(q))
      );
    })
    .sort((a, b) => {
      // Put non-self online first, then non-self offline, then me
      if (a.isMe !== b.isMe) return a.isMe ? 1 : -1;
      if (a.isOnline !== b.isOnline) return a.isOnline ? -1 : 1;
      return a.name.localeCompare(b.name);
    });

  return [...list, ...peerItems];
}
