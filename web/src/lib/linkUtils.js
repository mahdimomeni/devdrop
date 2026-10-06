// Utility functions for link extraction, URL detection, and tokenizing message text

// Regex to detect HTTP/HTTPS and www URLs
export const URL_REGEX = /(?:https?:\/\/|www\.)[^\s<>'"`]+/gi;

/**
 * Strips trailing sentence punctuation from URLs while preserving balanced parentheses.
 * e.g. "https://example.com/foo," -> { url: "https://example.com/foo", trailing: "," }
 * e.g. "(https://en.wikipedia.org/wiki/Go_(programming_language))" -> { url: "https://en.wikipedia.org/wiki/Go_(programming_language)", trailing: "" }
 */
export function cleanTrailingPunctuation(rawUrl) {
  let url = rawUrl;
  let trailing = '';

  const trailingPunctuationRegex = /[.,!?;:]+$/;
  const matchPunct = url.match(trailingPunctuationRegex);
  if (matchPunct) {
    trailing = matchPunct[0] + trailing;
    url = url.slice(0, url.length - matchPunct[0].length);
  }

  // Handle closing parentheses/brackets if unbalanced
  while (url.endsWith(')')) {
    const openCount = (url.match(/\(/g) || []).length;
    const closeCount = (url.match(/\)/g) || []).length;
    if (closeCount > openCount) {
      trailing = ')' + trailing;
      url = url.slice(0, -1);
    } else {
      break;
    }
  }

  while (url.endsWith(']')) {
    const openCount = (url.match(/\[/g) || []).length;
    const closeCount = (url.match(/\]/g) || []).length;
    if (closeCount > openCount) {
      trailing = ']' + trailing;
      url = url.slice(0, -1);
    } else {
      break;
    }
  }

  return { url, trailing };
}

/**
 * Normalizes URL for href attribute.
 * If URL starts with "www.", prefixes "https://".
 */
export function normalizeHref(url) {
  if (!url) return '';
  if (/^www\./i.test(url)) {
    return 'https://' + url;
  }
  return url;
}

/**
 * Extracts all valid URLs from a string.
 * @param {string} text
 * @returns {string[]} List of full href URLs
 */
export function extractUrls(text) {
  if (!text) return [];
  const matches = text.match(URL_REGEX);
  if (!matches) return [];

  const urls = [];
  for (const raw of matches) {
    const { url } = cleanTrailingPunctuation(raw);
    if (url && (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('www.'))) {
      urls.push(normalizeHref(url));
    }
  }
  return urls;
}

/**
 * Extracts the first valid URL from a string.
 * @param {string} text
 * @returns {string|null}
 */
export function extractFirstUrl(text) {
  const urls = extractUrls(text);
  return urls.length > 0 ? urls[0] : null;
}

// Re-export message segment parser and mention check for convenience
export { parseMessageSegments, isUserMentioned } from './mentionUtils.js';

