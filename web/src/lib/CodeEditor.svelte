<script>
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState, Compartment } from '@codemirror/state';
  import { oneDark } from '@codemirror/theme-one-dark';
  import { Code2, Wand2, X, Send, Sparkles } from 'lucide-svelte';
  import { supportedLanguages, getLangExtension } from './syntaxHighlight.js';

  let { initialCode = '', initialLanguage = 'go', onSend, onClose } = $props();

  let editorContainer = $state(null);
  let editorView = $state(null);
  let selectedLang = $state('go');
  let comment = $state('');
  let code = $state('');
  let formatMessage = $state('');

  $effect.pre(() => {
    if (initialLanguage) selectedLang = initialLanguage;
    if (initialCode) code = initialCode;
  });

  const languageCompartment = new Compartment();

  function formatBeautify() {
    if (!editorView) return;
    const currentText = editorView.state.doc.toString();
    let formatted = currentText;

    try {
      if (selectedLang === 'json') {
        const parsed = JSON.parse(currentText);
        formatted = JSON.stringify(parsed, null, 2);
        showFormatNotification('JSON formatted successfully');
      } else if (selectedLang === 'typescript' || selectedLang === 'javascript') {
        // Basic clean indentation and spacing
        formatted = formatSimpleIndentedCode(currentText);
        showFormatNotification('Code indented & trimmed');
      } else {
        formatted = formatSimpleIndentedCode(currentText);
        showFormatNotification('Code cleaned & formatted');
      }

      // Update editor text
      editorView.dispatch({
        changes: { from: 0, to: editorView.state.doc.length, insert: formatted },
      });
      code = formatted;
    } catch (err) {
      showFormatNotification('Format error: ' + err.message, true);
    }
  }

  function formatSimpleIndentedCode(text) {
    const lines = text.split('\n');
    let indentLevel = 0;
    const result = [];

    for (let rawLine of lines) {
      const line = rawLine.trim();
      if (!line) {
        result.push('');
        continue
      }

      // If line starts with closing brace, decrease indent before printing
      if (line.startsWith('}') || line.startsWith(']') || line.startsWith(')')) {
        indentLevel = Math.max(0, indentLevel - 1);
      }

      result.push('  '.repeat(indentLevel) + line);

      // If line ends with opening brace, increase indent for next lines
      if (line.endsWith('{') || line.endsWith('[') || line.endsWith('(') || line.endsWith(':')) {
        indentLevel++;
      }
    }
    return result.join('\n');
  }

  function showFormatNotification(msg, isError = false) {
    formatMessage = msg;
    setTimeout(() => {
      if (formatMessage === msg) formatMessage = '';
    }, 2500);
  }

  function updateLanguage(newLang) {
    selectedLang = newLang;
    if (editorView) {
      editorView.dispatch({
        effects: languageCompartment.reconfigure(getLangExtension(newLang)),
      });
    }
  }

  function handleSend() {
    const finalCode = editorView ? editorView.state.doc.toString().trim() : code.trim();
    if (!finalCode) return;
    onSend({
      code: finalCode,
      language: selectedLang,
      comment: comment.trim(),
    });
  }

  onMount(() => {
    if (!editorContainer) return;

    const startState = EditorState.create({
      doc: initialCode || '',
      extensions: [
        basicSetup,
        oneDark,
        languageCompartment.of(getLangExtension(selectedLang)),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            code = update.state.doc.toString();
          }
        }),
        EditorView.theme({
          '&': { height: '100%', fontSize: '13px' },
          '.cm-scroller': { overflow: 'auto', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' },
          '.cm-content': { padding: '12px 0' },
          '.cm-gutters': { backgroundColor: '#090d16', borderRight: '1px solid #1e293b', color: '#64748b' }
        })
      ],
    });

    editorView = new EditorView({
      state: startState,
      parent: editorContainer,
    });

    // Auto focus editor
    editorView.focus();
  });

  onDestroy(() => {
    if (editorView) {
      editorView.destroy();
    }
  });
</script>

<div class="border-t border-slate-700/80 bg-slate-900/95 backdrop-blur-md flex flex-col shadow-2xl transition-all duration-300">
  <!-- Toolbar Header -->
  <div class="flex items-center justify-between px-3 sm:px-4 py-2 sm:py-2.5 bg-slate-950/80 border-b border-slate-800 gap-2">
    <div class="flex items-center gap-2 sm:gap-3 flex-wrap min-w-0">
      <div class="flex items-center gap-1.5 text-cyan-400 font-semibold text-xs tracking-wide flex-shrink-0">
        <Code2 class="w-4 h-4" />
        <span class="hidden sm:inline">CODE SNIPPET DRAWER</span>
        <span class="sm:hidden font-mono font-bold">SNIPPET</span>
      </div>

      <!-- Language Selector -->
      <select
        value={selectedLang}
        onchange={(e) => updateLanguage(e.target.value)}
        class="bg-slate-800 hover:bg-slate-750 text-slate-200 text-xs rounded-lg px-2 sm:px-2.5 py-1 border border-slate-700 focus:outline-none focus:border-cyan-500 font-mono transition-colors cursor-pointer"
      >
        {#each supportedLanguages as lang}
          <option value={lang.id}>{lang.name}</option>
        {/each}
      </select>

      <!-- Beautify Button -->
      <button
        onclick={formatBeautify}
        class="flex items-center gap-1 sm:gap-1.5 px-2 sm:px-2.5 py-1 text-xs font-medium text-slate-300 hover:text-cyan-300 bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-lg transition-colors cursor-pointer"
        title="Format / Beautify code"
      >
        <Wand2 class="w-3.5 h-3.5 text-cyan-400" />
        <span class="hidden xs:inline">Format</span>
      </button>

      {#if formatMessage}
        <span class="text-[10px] sm:text-xs text-cyan-400 font-mono animate-fade-in truncate">
          {formatMessage}
        </span>
      {/if}
    </div>

    <!-- Close Drawer -->
    <button
      onclick={onClose}
      class="p-1.5 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors flex-shrink-0"
      title="Close Editor"
    >
      <X class="w-4 h-4" />
    </button>
  </div>

  <!-- Optional Description input -->
  <div class="px-3 sm:px-4 py-1.5 sm:py-2 bg-slate-900/50 border-b border-slate-800/80">
    <input
      type="text"
      bind:value={comment}
      placeholder="Snippet note or comment (optional)..."
      class="w-full bg-slate-950/70 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500 font-sans"
    />
  </div>

  <!-- CodeMirror Container -->
  <div
    bind:this={editorContainer}
    class="h-48 xs:h-56 sm:h-72 w-full overflow-hidden bg-slate-950 text-slate-200"
  ></div>

  <!-- Action Bar -->
  <div class="flex items-center justify-between px-3 sm:px-4 py-2 sm:py-2.5 bg-slate-950/90 border-t border-slate-800">
    <div class="text-[11px] text-slate-500 font-mono hidden xs:flex items-center gap-2">
      <span>Language: <strong class="text-slate-300">{selectedLang}</strong></span>
      <span>•</span>
      <span>2 spaces</span>
    </div>

    <div class="flex items-center gap-2 w-full xs:w-auto justify-end">
      <button
        onclick={onClose}
        class="px-3 py-1.5 text-xs font-medium text-slate-400 hover:text-white bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors"
      >
        Cancel
      </button>

      <button
        onclick={handleSend}
        class="flex items-center gap-1.5 px-3.5 sm:px-4 py-1.5 text-xs font-semibold text-white bg-cyan-600 hover:bg-cyan-500 rounded-lg shadow-sm shadow-cyan-600/30 transition-all active:scale-95 cursor-pointer"
      >
        <Send class="w-3.5 h-3.5" />
        <span class="hidden xs:inline">Drop Snippet to LAN</span>
        <span class="xs:hidden">Drop Snippet</span>
      </button>
    </div>
  </div>
</div>
