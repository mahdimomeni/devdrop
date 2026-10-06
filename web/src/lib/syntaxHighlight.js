import { javascript } from '@codemirror/lang-javascript';
import { python } from '@codemirror/lang-python';
import { json } from '@codemirror/lang-json';
import { sql } from '@codemirror/lang-sql';
import { java } from '@codemirror/lang-java';
import { yaml } from '@codemirror/lang-yaml';
import { StreamLanguage } from '@codemirror/language';
import { go } from '@codemirror/legacy-modes/mode/go';
import { csharp } from '@codemirror/legacy-modes/mode/clike';
import { shell } from '@codemirror/legacy-modes/mode/shell';
import { highlightCode, tagHighlighter, tags } from '@lezer/highlight';

export const supportedLanguages = [
  { id: 'go', name: 'Go' },
  { id: 'typescript', name: 'TypeScript / JS' },
  { id: 'python', name: 'Python' },
  { id: 'json', name: 'JSON' },
  { id: 'yaml', name: 'YAML' },
  { id: 'sql', name: 'SQL' },
  { id: 'csharp', name: 'C#' },
  { id: 'java', name: 'Java' },
  { id: 'shell', name: 'Shell / Bash' },
  { id: 'plaintext', name: 'Plaintext' },
];

export function normalizeLang(lang) {
  if (!lang) return 'plaintext';
  const l = String(lang).toLowerCase().trim();
  if (l === 'js' || l === 'javascript' || l === 'ts' || l === 'typescript') return 'typescript';
  if (l === 'py' || l === 'python') return 'python';
  if (l === 'golang' || l === 'go') return 'go';
  if (l === 'json') return 'json';
  if (l === 'yml' || l === 'yaml') return 'yaml';
  if (l === 'sql') return 'sql';
  if (l === 'c#' || l === 'cs' || l === 'csharp') return 'csharp';
  if (l === 'java') return 'java';
  if (l === 'sh' || l === 'bash' || l === 'shell' || l === 'zsh') return 'shell';
  return l;
}

const extensionCache = new Map();

export function getLangExtension(lang) {
  const normalized = normalizeLang(lang);
  if (extensionCache.has(normalized)) {
    return extensionCache.get(normalized);
  }

  let ext;
  switch (normalized) {
    case 'go':
      ext = StreamLanguage.define(go);
      break;
    case 'typescript':
      ext = javascript({ typescript: true });
      break;
    case 'python':
      ext = python();
      break;
    case 'json':
      ext = json();
      break;
    case 'yaml':
      ext = yaml();
      break;
    case 'sql':
      ext = sql();
      break;
    case 'csharp':
      ext = StreamLanguage.define(csharp);
      break;
    case 'java':
      ext = java();
      break;
    case 'shell':
      ext = StreamLanguage.define(shell);
      break;
    default:
      ext = [];
      break;
  }

  extensionCache.set(normalized, ext);
  return ext;
}

export function getParser(lang) {
  const ext = getLangExtension(lang);
  if (!ext || Array.isArray(ext)) return null;
  if (ext.language && ext.language.parser) {
    return ext.language.parser;
  }
  if (ext.parser) {
    return ext.parser;
  }
  return null;
}

export const devDropHighlighter = tagHighlighter([
  { tag: tags.keyword, class: 'cm-hl-keyword' },
  { tag: [tags.function(tags.variableName), tags.labelName], class: 'cm-hl-func' },
  { tag: [tags.propertyName], class: 'cm-hl-prop' },
  { tag: [tags.typeName, tags.className, tags.namespace], class: 'cm-hl-type' },
  { tag: [tags.number], class: 'cm-hl-num' },
  { tag: [tags.bool, tags.atom], class: 'cm-hl-bool' },
  { tag: [tags.string, tags.special(tags.string), tags.character], class: 'cm-hl-str' },
  { tag: [tags.comment, tags.meta, tags.lineComment, tags.blockComment, tags.docComment], class: 'cm-hl-comment' },
  { tag: [tags.operator, tags.operatorKeyword, tags.url, tags.escape, tags.regexp], class: 'cm-hl-op' },
  { tag: [tags.punctuation, tags.separator, tags.bracket], class: 'cm-hl-punct' },
  { tag: [tags.variableName, tags.name], class: 'cm-hl-var' },
  { tag: tags.invalid, class: 'cm-hl-invalid' },
]);

export function escapeHtml(str) {
  if (!str) return '';
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

const highlightCache = new Map();
const MAX_CACHE_SIZE = 250;

function setCache(key, val) {
  if (highlightCache.size >= MAX_CACHE_SIZE) {
    const oldestKey = highlightCache.keys().next().value;
    highlightCache.delete(oldestKey);
  }
  highlightCache.set(key, val);
}

export function highlightCodeLines(code, lang = '') {
  if (!code && code !== '') return [];
  const normalized = normalizeLang(lang);
  const cacheKey = `${normalized}:${code}`;

  if (highlightCache.has(cacheKey)) {
    return highlightCache.get(cacheKey);
  }

  const parser = getParser(normalized);
  if (!parser) {
    const rawLines = code.split('\n').map((l) => escapeHtml(l));
    setCache(cacheKey, rawLines);
    return rawLines;
  }

  try {
    const tree = parser.parse(code);
    const lines = [];
    let currentLine = '';

    highlightCode(
      code,
      tree,
      devDropHighlighter,
      (text, style) => {
        const escaped = escapeHtml(text);
        if (style) {
          currentLine += `<span class="${style}">${escaped}</span>`;
        } else {
          currentLine += escaped;
        }
      },
      () => {
        lines.push(currentLine);
        currentLine = '';
      }
    );
    lines.push(currentLine);

    setCache(cacheKey, lines);
    return lines;
  } catch (err) {
    console.warn('Syntax highlight failed, falling back to plain text:', err);
    const fallbackLines = code.split('\n').map((l) => escapeHtml(l));
    setCache(cacheKey, fallbackLines);
    return fallbackLines;
  }
}
