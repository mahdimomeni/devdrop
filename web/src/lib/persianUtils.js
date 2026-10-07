/**
 * Persian / Arabic text detection and typography utilities
 */

// Regex covering Persian/Arabic Unicode blocks:
// - Arabic (U+0600 - U+06FF)
// - Arabic Supplement (U+0750 - U+077F)
// - Arabic Extended-A (U+08A0 - U+08FF)
// - Arabic Presentation Forms-A (U+FB50 - U+FDFF)
// - Arabic Presentation Forms-B (U+FE70 - U+FEFC)
export const PERSIAN_ARABIC_REGEX = /[\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF\uFB50-\uFDFF\uFE70-\uFEFC]/;

// Persian specific characters: گ، چ، پ، ژ، ی، ک، etc.
export const PERSIAN_CHAR_REGEX = /[\u067E\u0686\u0698\u06AF\u06CC\u06A9\u06F0-\u06F9]/;

/**
 * Checks if the string contains any Persian or Arabic characters
 * @param {string} text
 * @returns {boolean}
 */
export function hasPersianText(text) {
  if (!text || typeof text !== 'string') return false;
  return PERSIAN_ARABIC_REGEX.test(text);
}

/**
 * Determines whether the text starts with a strong RTL (Persian/Arabic) character
 * (equivalent to the HTML5 dir="auto" first-strong-character algorithm)
 * @param {string} text
 * @returns {boolean}
 */
export function isRtlText(text) {
  if (!text || typeof text !== 'string') return false;
  // Match the first strong character (Latin or RTL)
  const match = text.match(/[A-Za-z\u00C0-\u024F\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF\uFB50-\uFDFF\uFE70-\uFEFC]/);
  if (!match) return false;
  return PERSIAN_ARABIC_REGEX.test(match[0]);
}

/**
 * Returns 'rtl' or 'ltr' text direction
 * @param {string} text
 * @returns {'rtl' | 'ltr'}
 */
export function getTextDirection(text) {
  return isRtlText(text) ? 'rtl' : 'ltr';
}
