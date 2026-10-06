import { javascript } from '@codemirror/lang-javascript';
import { python } from '@codemirror/lang-python';
import { json } from '@codemirror/lang-json';
import { sql } from '@codemirror/lang-sql';
import { java } from '@codemirror/lang-java';
import { yaml } from '@codemirror/lang-yaml';
import { markdown } from '@codemirror/lang-markdown';
import { linter } from '@codemirror/lint';
import { StreamLanguage } from '@codemirror/language';
import { go } from '@codemirror/legacy-modes/mode/go';
import { csharp } from '@codemirror/legacy-modes/mode/clike';
import { shell } from '@codemirror/legacy-modes/mode/shell';
import { highlightCode, tagHighlighter, tags } from '@lezer/highlight';
import { marked } from 'marked';
import DOMPurify from 'dompurify';

export const supportedLanguages = [
  { id: 'go', name: 'Go' },
  { id: 'typescript', name: 'TypeScript / JS' },
  { id: 'python', name: 'Python' },
  { id: 'markdown', name: 'Markdown' },
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
  if (l === 'md' || l === 'markdown') return 'markdown';
  if (l === 'json') return 'json';
  if (l === 'yml' || l === 'yaml') return 'yaml';
  if (l === 'sql') return 'sql';
  if (l === 'c#' || l === 'cs' || l === 'csharp') return 'csharp';
  if (l === 'java') return 'java';
  if (l === 'sh' || l === 'bash' || l === 'shell' || l === 'zsh') return 'shell';
  return l;
}

export const markdownLinter = linter((view) => {
  const diagnostics = [];
  const text = view.state.doc.toString();
  if (!text) return diagnostics;

  const lines = text.split('\n');
  let inCodeBlock = false;
  let codeBlockStartPos = 0;
  let codeBlockFenceChar = '';
  let codeBlockFenceLen = 0;

  let offset = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineLen = line.length;

    const fenceMatch = line.match(/^(\s*)(`{3,}|~{3,})/);
    if (fenceMatch) {
      const fenceStr = fenceMatch[2];
      const fChar = fenceStr[0];
      const fLen = fenceStr.length;

      if (!inCodeBlock) {
        inCodeBlock = true;
        codeBlockStartPos = offset;
        codeBlockFenceChar = fChar;
        codeBlockFenceLen = fLen;
      } else if (fChar === codeBlockFenceChar && fLen >= codeBlockFenceLen) {
        inCodeBlock = false;
      }
    } else if (!inCodeBlock) {
      // 1. Heading missing space after '#' (CommonMark requirement)
      const headingMatch = line.match(/^(\s*)(#{1,6})([^\s#].*)$/);
      if (headingMatch) {
        const start = offset + headingMatch[1].length;
        const end = start + headingMatch[2].length;
        diagnostics.push({
          from: start,
          to: end,
          severity: 'warning',
          message: 'Headings must have a space after "#" (e.g., "# Heading")',
        });
      }

      // 2. Empty link destination URL: [text]()
      const emptyLinkRegex = /\[([^\]]+)\]\(\s*\)/g;
      let emptyMatch;
      while ((emptyMatch = emptyLinkRegex.exec(line)) !== null) {
        const start = offset + emptyMatch.index;
        diagnostics.push({
          from: start,
          to: start + emptyMatch[0].length,
          severity: 'warning',
          message: 'Link destination URL is empty',
        });
      }

      // 3. Unmatched inline code backticks on a single line
      let backtickCount = 0;
      for (let c = 0; c < line.length; c++) {
        if (line[c] === '`' && (c === 0 || line[c - 1] !== '\\')) {
          backtickCount++;
        }
      }
      if (backtickCount % 2 !== 0) {
        const lastIdx = line.lastIndexOf('`');
        diagnostics.push({
          from: offset + (lastIdx >= 0 ? lastIdx : 0),
          to: offset + lineLen,
          severity: 'info',
          message: 'Unclosed inline code backtick (`)',
        });
      }
    }

    offset += lineLen + 1; // +1 for \n
  }

  if (inCodeBlock) {
    diagnostics.push({
      from: codeBlockStartPos,
      to: Math.min(view.state.doc.length, codeBlockStartPos + 3),
      severity: 'error',
      message: 'Unclosed code block (missing closing fence ```)',
    });
  }

  return diagnostics;
});

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
    case 'markdown':
      ext = [markdown(), markdownLinter];
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
  if (!ext) return null;

  if (Array.isArray(ext)) {
    for (const item of ext) {
      if (item && item.language && item.language.parser) return item.language.parser;
      if (item && item.parser) return item.parser;
    }
    return null;
  }

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
  { tag: [tags.heading, tags.heading1, tags.heading2, tags.heading3, tags.heading4, tags.heading5, tags.heading6], class: 'cm-hl-heading' },
  { tag: tags.strong, class: 'cm-hl-strong' },
  { tag: tags.emphasis, class: 'cm-hl-emphasis' },
  { tag: tags.monospace, class: 'cm-hl-mono' },
  { tag: tags.quote, class: 'cm-hl-quote' },
  { tag: tags.list, class: 'cm-hl-list' },
  { tag: [tags.function(tags.variableName), tags.labelName], class: 'cm-hl-func' },
  { tag: [tags.propertyName], class: 'cm-hl-prop' },
  { tag: [tags.typeName, tags.className, tags.namespace], class: 'cm-hl-type' },
  { tag: [tags.number], class: 'cm-hl-num' },
  { tag: [tags.bool, tags.atom], class: 'cm-hl-bool' },
  { tag: [tags.string, tags.special(tags.string), tags.character], class: 'cm-hl-str' },
  { tag: [tags.comment, tags.meta, tags.lineComment, tags.blockComment, tags.docComment], class: 'cm-hl-comment' },
  { tag: [tags.operator, tags.operatorKeyword, tags.url, tags.escape, tags.regexp], class: 'cm-hl-op' },
  { tag: [tags.link, tags.url], class: 'cm-hl-link' },
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

// Markdown preview HTML generation with sanitization & caching
const markdownRenderCache = new Map();
const MAX_RENDER_CACHE = 150;

export function renderMarkdown(content) {
  if (!content) return '';
  if (markdownRenderCache.has(content)) {
    return markdownRenderCache.get(content);
  }

  try {
    const rawHtml = marked.parse(content, { gfm: true, breaks: true });
    const cleanHtml = typeof window !== 'undefined' && DOMPurify ? DOMPurify.sanitize(rawHtml) : rawHtml;
    if (markdownRenderCache.size >= MAX_RENDER_CACHE) {
      const oldestKey = markdownRenderCache.keys().next().value;
      markdownRenderCache.delete(oldestKey);
    }
    markdownRenderCache.set(content, cleanHtml);
    return cleanHtml;
  } catch (err) {
    console.warn('Markdown render failed, falling back to escaped text:', err);
    return escapeHtml(content);
  }
}
