<script>
  import { onMount, tick } from 'svelte';
  import {
    Send,
    Code,
    Paperclip,
    FolderUp,
    UploadCloud,
    Flame,
    Clock,
    Copy,
    Check,
    GitCompare,
    Shield,
    Sparkles,
    Radio,
    Terminal,
    AlertCircle,
    Info,
    FileText
  } from 'lucide-svelte';
  import { formatRelativeTime } from './api.js';
  import FileCard from './FileCard.svelte';
  import CodeEditor from './CodeEditor.svelte';
  import DiffModal from './DiffModal.svelte';

  let {
    currentUser,
    selectedPeer,
    messages = [],
    onSendMessage,
    onSendCode,
    onUploadFiles,
  } = $props();

  let textInput = $state('');
  let messagesContainer = $state(null);
  let fileInput = $state(null);
  let folderInput = $state(null);

  // Transfer options
  let expiration = $state('24h');
  let devIgnore = $state(true);

  // Drawer / Modal states
  let isCodeDrawerOpen = $state(false);
  let codeDrawerInitialCode = $state('');
  let codeDrawerInitialLang = $state('go');

  // Smart Paste Prompt
  let smartPastePrompt = $state(null); // { text, linesCount }

  // Diff Modal states
  let diffModalData = $state(null); // { oldCode, newCode, oldLabel, newLabel }

  // Drag and drop overlay
  let isDraggingOver = $state(false);
  let dragCounter = $state(0);

  // Copied message IDs
  let copiedId = $state(null);

  // Auto scroll to bottom
  async function scrollToBottom(smooth = true) {
    await tick();
    if (messagesContainer) {
      messagesContainer.scrollTo({
        top: messagesContainer.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto',
      });
    }
  }

  $effect(() => {
    // Scroll when messages change
    if (messages.length) {
      scrollToBottom();
    }
  });

  function handleSendText() {
    const trimmed = textInput.trim();
    if (!trimmed) return;
    onSendMessage(trimmed);
    textInput = '';
    scrollToBottom();
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendText();
    }
  }

  // Smart Paste Listener
  function handlePaste(e) {
    // 1. Check for clipboard image
    const items = e.clipboardData?.items;
    if (items) {
      for (let i = 0; i < items.length; i++) {
        if (items[i].type.startsWith('image/')) {
          const file = items[i].getAsFile();
          if (file) {
            e.preventDefault();
            const ext = file.type.split('/')[1] || 'png';
            const imgName = `pasted_image_${Date.now()}.${ext}`;
            const renamedFile = new File([file], imgName, { type: file.type });
            uploadFileList([renamedFile], false, imgName);
            return;
          }
        }
      }
    }

    // 2. Check for multi-line code
    const pastedText = e.clipboardData?.getData('text') || '';
    const lines = pastedText.split('\n');
    const hasCodeIndicators =
      lines.length > 3 &&
      (pastedText.includes('{') ||
        pastedText.includes('}') ||
        pastedText.includes('func ') ||
        pastedText.includes('def ') ||
        pastedText.includes('const ') ||
        pastedText.includes('import ') ||
        pastedText.includes('package ') ||
        pastedText.includes('class ') ||
        pastedText.includes('SELECT ') ||
        pastedText.includes('WHERE ') ||
        pastedText.includes('CREATE ') ||
        pastedText.includes('return ') ||
        pastedText.includes('public static'));

    if (hasCodeIndicators || lines.length > 6) {
      smartPastePrompt = {
        text: pastedText,
        linesCount: lines.length,
      };
    }
  }

  function openSmartPasteInEditor() {
    if (!smartPastePrompt) return;
    const text = smartPastePrompt.text;
    smartPastePrompt = null;
    textInput = '';
    openCodeDrawerWith(text);
  }

  function dismissSmartPaste() {
    smartPastePrompt = null;
  }

  function openCodeDrawerWith(initialCode = '', lang = 'go') {
    codeDrawerInitialCode = initialCode;
    codeDrawerInitialLang = detectLanguage(initialCode) || lang;
    isCodeDrawerOpen = true;
  }

  function detectLanguage(code) {
    if (!code) return 'go';
    if (code.includes('package ') && (code.includes('func ') || code.includes('import ('))) return 'go';
    if (code.includes('def ') && code.includes(':')) return 'python';
    if (code.startsWith('{') && code.endsWith('}')) {
      try {
        JSON.parse(code);
        return 'json';
      } catch {}
    }
    if (code.includes('const ') || code.includes('export ') || code.includes('function ')) return 'typescript';
    if (code.includes('SELECT ') || code.includes('INSERT INTO ')) return 'sql';
    if (code.includes('public class ') || code.includes('System.out')) return 'java';
    return 'plaintext';
  }

  function handleSendCodeFromDrawer(payload) {
    isCodeDrawerOpen = false;
    onSendCode(payload);
  }

  // File & Folder Uploads
  function handleSelectFile(e) {
    const files = Array.from(e.target.files || []);
    if (files.length > 0) {
      uploadFileList(files, false, files[0].name);
    }
    e.target.value = '';
  }

  function handleSelectFolder(e) {
    const files = Array.from(e.target.files || []);
    if (files.length > 0) {
      // Determine folder name from webkitRelativePath
      const firstPath = files[0].webkitRelativePath || '';
      const folderName = firstPath.split('/')[0] || 'folder_archive';
      uploadFileList(files, true, folderName);
    }
    e.target.value = '';
  }

  function uploadFileList(files, isFolder, archiveName) {
    onUploadFiles({
      files,
      isFolder,
      folderName: archiveName,
      expiration,
      devIgnore,
    });
  }

  // Drag and Drop Handling
  function handleDragEnter(e) {
    e.preventDefault();
    dragCounter++;
    isDraggingOver = true;
  }

  function handleDragLeave(e) {
    e.preventDefault();
    dragCounter--;
    if (dragCounter <= 0) {
      isDraggingOver = false;
      dragCounter = 0;
    }
  }

  function handleDragOver(e) {
    e.preventDefault();
  }

  async function handleDrop(e) {
    e.preventDefault();
    isDraggingOver = false;
    dragCounter = 0;

    const dt = e.dataTransfer;
    if (!dt) return;

    // Check if files or folders dropped
    if (dt.items) {
      const entries = [];
      for (let i = 0; i < dt.items.length; i++) {
        const item = dt.items[i];
        if (item.kind === 'file') {
          const entry = item.webkitGetAsEntry ? item.webkitGetAsEntry() : null;
          if (entry) entries.push(entry);
        }
      }

      if (entries.length > 0) {
        // If an entry is directory, traverse it
        const hasDirectory = entries.some((entry) => entry.isDirectory);
        if (hasDirectory) {
          const allFiles = [];
          for (const entry of entries) {
            await traverseEntry(entry, '', allFiles);
          }
          const folderName = entries.find((e) => e.isDirectory)?.name || 'dropped_folder';
          uploadFileList(allFiles, true, folderName);
          return;
        }
      }
    }

    // Fallback to regular files drop
    const files = Array.from(dt.files || []);
    if (files.length > 0) {
      uploadFileList(files, false, files[0].name);
    }
  }

  // Recursively traverse dropped directory
  async function traverseEntry(entry, path, resultFiles) {
    if (entry.isFile) {
      return new Promise((resolve) => {
        entry.file((file) => {
          // Attach relative path for folder preservation
          const relPath = path ? `${path}/${file.name}` : file.name;
          const renamed = new File([file], relPath, { type: file.type });
          resultFiles.push(renamed);
          resolve();
        });
      });
    } else if (entry.isDirectory) {
      const dirReader = entry.createReader();
      const readEntries = () =>
        new Promise((resolve) => {
          dirReader.readEntries(async (entries) => {
            if (entries.length === 0) {
              resolve();
            } else {
              const newPath = path ? `${path}/${entry.name}` : entry.name;
              for (const child of entries) {
                await traverseEntry(child, newPath, resultFiles);
              }
              await readEntries(); // Continue reading batch
              resolve();
            }
          });
        });
      await readEntries();
    }
  }

  function copyCodeRaw(msgId, codeContent) {
    navigator.clipboard.writeText(codeContent);
    copiedId = msgId;
    setTimeout(() => {
      if (copiedId === msgId) copiedId = null;
    }, 2000);
  }

  // Find previous code snippet in message history to compare against
  function findPreviousSnippet(currentIndex) {
    for (let i = currentIndex - 1; i >= 0; i--) {
      if (messages[i].type === 'code' && messages[i].snippet?.code_content) {
        return messages[i];
      }
    }
    return null;
  }

  function openDiff(snippetA, snippetB) {
    diffModalData = {
      oldCode: snippetA.snippet.code_content,
      newCode: snippetB.snippet.code_content,
      oldLabel: `${snippetA.snippet.language.toUpperCase()} (${formatRelativeTime(snippetA.created_at)})`,
      newLabel: `${snippetB.snippet.language.toUpperCase()} (${formatRelativeTime(snippetB.created_at)})`,
    };
  }

  let isBroadcast = $derived(selectedPeer === 'broadcast' || !selectedPeer);
</script>

<div
  role="region"
  aria-label="Chat messages and drop area"
  class="relative flex-1 h-full flex flex-col bg-slate-900 overflow-hidden"
  ondragenter={handleDragEnter}
  ondragleave={handleDragLeave}
  ondragover={handleDragOver}
  ondrop={handleDrop}
  onpaste={handlePaste}
>
  <!-- Full Screen Drag and Drop HUD Overlay -->
  {#if isDraggingOver}
    <div class="absolute inset-0 z-50 bg-cyan-950/85 backdrop-blur-md border-4 border-dashed border-cyan-400 flex flex-col items-center justify-center p-8 text-center animate-in fade-in duration-150 pointer-events-none">
      <div class="p-6 rounded-2xl bg-cyan-500/20 text-cyan-300 border border-cyan-400/40 mb-4 shadow-2xl animate-bounce">
        <UploadCloud class="w-16 h-16" />
      </div>
      <h2 class="text-2xl font-bold text-white font-mono tracking-tight">
        Drop Files or Folders Here
      </h2>
      <p class="text-sm text-cyan-200 mt-2 max-w-md font-mono">
        {#if devIgnore}
          <span class="inline-flex items-center gap-1 text-emerald-400 font-bold">
            <Shield class="w-4 h-4" /> Dev-Ignore Active:
          </span>
          Excludes .git, node_modules, target, vendor, dist.
        {:else}
          Streaming directly to LAN disk in chunks.
        {/if}
      </p>
    </div>
  {/if}

  <!-- Header -->
  <header class="h-16 px-6 bg-slate-950/90 border-b border-slate-800/80 flex items-center justify-between flex-shrink-0 backdrop-blur-sm z-10">
    <div class="flex items-center gap-3.5">
      {#if isBroadcast}
        <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-cyan-500/20 to-amber-500/20 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
          <Radio class="w-5 h-5 animate-pulse" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-sm font-semibold text-slate-100 font-mono tracking-tight">
              LAN Broadcast Room
            </h1>
            <span class="text-[10px] px-2 py-0.5 rounded-full bg-cyan-950 text-cyan-400 font-bold border border-cyan-800/50 uppercase tracking-wider">
              Public Feed
            </span>
          </div>
          <p class="text-xs text-slate-400 font-mono mt-0.5">
            Messages and files are visible to all connected LAN developers
          </p>
        </div>
      {:else}
        <div class="relative w-10 h-10 rounded-xl bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-200 text-sm">
          {(selectedPeer?.display_name || 'P')[0]?.toUpperCase()}
          {#if selectedPeer?.is_online}
            <div class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full bg-emerald-400 border-2 border-slate-900 animate-pulse shadow-sm shadow-emerald-400"></div>
          {:else}
            <div class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full bg-slate-600 border-2 border-slate-900"></div>
          {/if}
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-sm font-semibold text-slate-100 font-mono">
              {selectedPeer?.display_name}
            </h1>
            {#if selectedPeer?.is_online}
              <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 font-bold border border-emerald-800/40 font-mono">
                Online
              </span>
            {:else}
              <span class="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono">
                Offline
              </span>
            {/if}
          </div>
          <p class="text-xs text-slate-400 font-mono mt-0.5 flex items-center gap-2">
            <span>IP: <strong class="text-slate-300">{selectedPeer?.ip_address}</strong></span>
            <span>•</span>
            <span>Direct 1-to-1</span>
          </p>
        </div>
      {/if}
    </div>

    <!-- Right header stats -->
    <div class="flex items-center gap-3 text-xs text-slate-400 font-mono">
      <div class="hidden sm:flex items-center gap-1.5 px-3 py-1.5 bg-slate-900 rounded-lg border border-slate-800">
        <Terminal class="w-3.5 h-3.5 text-cyan-400" />
        <span>DevDrop Channel</span>
      </div>
    </div>
  </header>

  <!-- Smart Paste Toast Prompt -->
  {#if smartPastePrompt}
    <div class="bg-cyan-950/90 border-b border-cyan-800 px-6 py-2.5 flex items-center justify-between text-xs text-cyan-200 animate-in slide-in-from-top duration-200 z-10 shadow-lg">
      <div class="flex items-center gap-2.5">
        <Sparkles class="w-4 h-4 text-cyan-400 flex-shrink-0 animate-spin" />
        <span class="font-medium">
          Multi-line code detected ({smartPastePrompt.linesCount} lines). Open in Code Snippet Editor?
        </span>
      </div>
      <div class="flex items-center gap-2">
        <button
          onclick={openSmartPasteInEditor}
          class="px-3 py-1 bg-cyan-600 hover:bg-cyan-500 text-white font-semibold rounded-md shadow transition-colors cursor-pointer"
        >
          Open in Editor
        </button>
        <button
          onclick={dismissSmartPaste}
          class="px-2.5 py-1 text-slate-400 hover:text-white transition-colors"
        >
          Dismiss
        </button>
      </div>
    </div>
  {/if}

  <!-- Message History Scroll Area -->
  <div
    bind:this={messagesContainer}
    class="flex-1 overflow-y-auto p-6 space-y-5 select-text"
  >
    {#if messages.length === 0}
      <div class="h-full flex flex-col items-center justify-center text-center p-8 text-slate-500">
        <div class="p-4 rounded-2xl bg-slate-950/70 border border-slate-800 mb-3 text-slate-400">
          <Terminal class="w-10 h-10" />
        </div>
        <h3 class="text-sm font-semibold text-slate-300 font-mono">No messages yet</h3>
        <p class="text-xs text-slate-500 max-w-sm mt-1 font-mono">
          {#if isBroadcast}
            Drop a message, code snippet, or folder here. Everyone on this LAN can see it immediately.
          {:else}
            Start chatting or drop files with {selectedPeer?.display_name}.
          {/if}
        </p>
      </div>
    {:else}
      {#each messages as msg, index (msg.id)}
        {@const isMe = msg.sender_id === currentUser?.id}
        {@const prevSnippet = msg.type === 'code' ? findPreviousSnippet(index) : null}

        <div class="flex flex-col {isMe ? 'items-end' : 'items-start'} group">
          <!-- Sender Info & Timestamp -->
          <div class="flex items-center gap-2 mb-1 px-1 text-[11px] text-slate-400 font-mono">
            <span class="font-semibold {isMe ? 'text-cyan-400' : 'text-slate-300'}">
              {isMe ? 'You' : msg.sender_id === selectedPeer?.id ? selectedPeer?.display_name : 'Peer'}
            </span>
            <span>•</span>
            <span>{formatRelativeTime(msg.created_at)}</span>
          </div>

          <!-- Message Body by Type -->
          {#if msg.type === 'text'}
            <div
              class="max-w-2xl px-4 py-2.5 rounded-2xl text-sm leading-relaxed whitespace-pre-wrap break-words font-sans shadow-md {isMe
                ? 'bg-cyan-600 text-white rounded-br-xs'
                : 'bg-slate-800 border border-slate-700 text-slate-100 rounded-bl-xs'}"
            >
              {msg.body}
            </div>

          {:else if msg.type === 'code' && msg.snippet}
            <!-- Code Snippet Card -->
            <div class="w-full max-w-3xl rounded-xl overflow-hidden border border-slate-750 bg-slate-950 shadow-xl">
              <!-- Code Card Header -->
              <div class="flex items-center justify-between px-4 py-2 bg-slate-900/90 border-b border-slate-800">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-bold font-mono px-2 py-0.5 rounded bg-cyan-950 text-cyan-400 border border-cyan-800/60 uppercase">
                    {msg.snippet.language}
                  </span>
                  {#if msg.body}
                    <span class="text-xs text-slate-300 font-mono truncate max-w-sm">
                      {msg.body}
                    </span>
                  {/if}
                </div>

                <div class="flex items-center gap-2">
                  <!-- Diff Button if previous snippet exists -->
                  {#if prevSnippet}
                    <button
                      onclick={() => openDiff(prevSnippet, msg)}
                      class="flex items-center gap-1 px-2.5 py-1 text-xs text-indigo-300 hover:text-indigo-200 bg-indigo-950/60 hover:bg-indigo-900/70 border border-indigo-700/50 rounded-lg transition-colors cursor-pointer"
                      title="Compare side-by-side with previous snippet"
                    >
                      <GitCompare class="w-3.5 h-3.5" />
                      <span>Compare Diff</span>
                    </button>
                  {/if}

                  <!-- Copy Raw Button -->
                  <button
                    onclick={() => copyCodeRaw(msg.id, msg.snippet.code_content)}
                    class="flex items-center gap-1 px-2.5 py-1 text-xs text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-lg transition-colors cursor-pointer"
                    title="Copy Raw Code"
                  >
                    {#if copiedId === msg.id}
                      <Check class="w-3.5 h-3.5 text-emerald-400" />
                      <span class="text-emerald-400">Copied!</span>
                    {:else}
                      <Copy class="w-3.5 h-3.5" />
                      <span>Copy Raw</span>
                    {/if}
                  </button>
                </div>
              </div>

              <!-- Code Lines with numbering -->
              <div class="p-3 overflow-x-auto text-xs font-mono leading-relaxed bg-slate-950 text-slate-200">
                <table class="border-collapse w-full">
                  <tbody>
                    {#each msg.snippet.code_content.split('\n') as line, lIdx}
                      <tr class="hover:bg-slate-900/70 transition-colors">
                        <td class="pr-4 py-0.5 text-right text-slate-600 select-none w-10 font-mono text-[11px] align-top">
                          {lIdx + 1}
                        </td>
                        <td class="py-0.5 whitespace-pre font-mono text-slate-200 break-normal">
                          {line || ' '}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>

          {:else if msg.type === 'file' && msg.transfer}
            <!-- File or Folder Transfer Bubble -->
            <FileCard transfer={msg.transfer} isOutgoing={isMe} />
          {/if}
        </div>
      {/each}
    {/if}
  </div>

  <!-- Code Drawer Component (Collapsible sliding up from input area) -->
  {#if isCodeDrawerOpen}
    <CodeEditor
      initialCode={codeDrawerInitialCode}
      initialLanguage={codeDrawerInitialLang}
      onSend={handleSendCodeFromDrawer}
      onClose={() => (isCodeDrawerOpen = false)}
    />
  {/if}

  <!-- Footer Input Area -->
  <footer class="p-4 bg-slate-950/95 border-t border-slate-800 flex-shrink-0">
    <div class="relative flex flex-col bg-slate-900 border border-slate-700/80 rounded-2xl shadow-xl focus-within:border-cyan-500/80 focus-within:ring-1 focus-within:ring-cyan-500/40 transition-all">
      
      <!-- Textarea -->
      <textarea
        bind:value={textInput}
        onkeydown={handleKeyDown}
        placeholder={isBroadcast ? "Type message to LAN broadcast (Shift+Enter for new line, Ctrl+V to paste images/code)..." : `Type direct message to ${selectedPeer?.display_name}...`}
        rows="2"
        class="w-full bg-transparent text-slate-100 placeholder-slate-500 text-sm p-3.5 focus:outline-none resize-none font-sans"
      ></textarea>

      <!-- Input Toolbar -->
      <div class="flex items-center justify-between px-3 pb-2.5 pt-1 border-t border-slate-800/60">
        <!-- Left tools: Code drawer, File, Folder, Dev-Ignore toggle, Expiration -->
        <div class="flex flex-wrap items-center gap-1.5 sm:gap-2">
          <!-- Open Code Drawer -->
          <button
            onclick={() => openCodeDrawerWith('')}
            class="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-cyan-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer"
            title="Open Code Snippet Editor"
          >
            <Code class="w-3.5 h-3.5 text-cyan-400" />
            <span class="hidden sm:inline">Add Code</span>
          </button>

          <!-- Hidden File Inputs -->
          <input
            type="file"
            bind:this={fileInput}
            onchange={handleSelectFile}
            class="hidden"
          />
          <input
            type="file"
            bind:this={folderInput}
            onchange={handleSelectFolder}
            webkitdirectory
            directory
            multiple
            class="hidden"
          />

          <!-- Attach File -->
          <button
            onclick={() => fileInput.click()}
            class="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-cyan-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer"
            title="Attach a file"
          >
            <Paperclip class="w-3.5 h-3.5 text-cyan-400" />
            <span class="hidden sm:inline">File</span>
          </button>

          <!-- Attach Folder -->
          <button
            onclick={() => folderInput.click()}
            class="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-amber-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer"
            title="Attach entire folder (compresses on stream)"
          >
            <FolderUp class="w-3.5 h-3.5 text-amber-400" />
            <span class="hidden sm:inline">Folder</span>
          </button>

          <!-- Dev-Ignore Filter Toggle -->
          <button
            onclick={() => (devIgnore = !devIgnore)}
            class="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all cursor-pointer {devIgnore
              ? 'bg-emerald-950/80 text-emerald-300 border border-emerald-800/80 shadow-xs'
              : 'bg-slate-800 text-slate-400 border border-slate-700 line-through opacity-70'}"
            title="Toggle Dev-Ignore filter (strips .git, node_modules, target, vendor, dist)"
          >
            <Shield class="w-3 h-3 text-emerald-400" />
            <span class="hidden md:inline">Dev-Ignore {devIgnore ? 'ON' : 'OFF'}</span>
          </button>

          <!-- TTL Selector -->
          <div class="flex items-center gap-1 text-xs text-slate-400 font-mono">
            <Clock class="w-3.5 h-3.5 text-slate-400 ml-1" />
            <select
              bind:value={expiration}
              class="bg-slate-800 text-slate-300 text-xs rounded-lg px-2 py-1 border border-slate-700 focus:outline-none focus:border-cyan-500 font-mono cursor-pointer"
              title="Transfer Expiration Lifecycle"
            >
              <option value="burn">🔥 Burn on Read (1x)</option>
              <option value="1h">1 Hour</option>
              <option value="6h">6 Hours</option>
              <option value="24h">24 Hours (Default)</option>
            </select>
          </div>
        </div>

        <!-- Right: Send Button -->
        <button
          onclick={handleSendText}
          disabled={!textInput.trim()}
          class="flex items-center gap-1.5 px-4 py-1.5 rounded-xl text-xs font-semibold text-white bg-cyan-600 hover:bg-cyan-500 disabled:opacity-40 disabled:cursor-not-allowed shadow-md shadow-cyan-600/30 transition-all active:scale-95 cursor-pointer"
        >
          <Send class="w-3.5 h-3.5" />
          <span>Send</span>
        </button>
      </div>
    </div>
  </footer>

  <!-- Diff Comparison Modal -->
  {#if diffModalData}
    <DiffModal
      oldCode={diffModalData.oldCode}
      newCode={diffModalData.newCode}
      oldLabel={diffModalData.oldLabel}
      newLabel={diffModalData.newLabel}
      onClose={() => (diffModalData = null)}
    />
  {/if}
</div>
