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
    ShieldCheck,
    Sparkles,
    Radio,
    Terminal,
    AlertCircle,
    Info,
    FileText,
    Reply,
    X,
    Loader2,
    ChevronDown,
    ChevronUp,
    ChevronLeft,
    Eye,
    AtSign,
    Bell,
    BellRing,
    BellOff,
    Smile,
    SmilePlus
  } from 'lucide-svelte';
  import { formatRelativeTime, copyToClipboard } from './api.js';
  import FileCard from './FileCard.svelte';
  import CodeEditor from './CodeEditor.svelte';
  import DiffModal from './DiffModal.svelte';
  import LinkPreviewCard from './LinkPreviewCard.svelte';
  import EmojiPicker from './EmojiPicker.svelte';
  import { highlightCodeLines, normalizeLang, renderMarkdown } from './syntaxHighlight.js';
  import { extractFirstUrl } from './linkUtils.js';
  import {
    QUICK_REACTIONS,
    getEmojiOnlyInfo
  } from './emojiData.js';
  import {
    parseMessageSegments,
    isUserMentioned,
    extractMentionContext,
    filterMentionCandidates
  } from './mentionUtils.js';

  let {
    currentUser,
    selectedPeer,
    peers = [],
    messages = [],
    hasMore = false,
    isLoadingMessages = false,
    isLoadingOlder = false,
    notificationSettings = null,
    onOpenNotificationSettings = null,
    onOpenSecuritySettings = null,
    onLoadOlder,
    onSendMessage,
    onSendCode,
    onUploadFiles,
    onToggleReaction = null,
    onBackToPeers = null,
    totalUnreadCount = 0,
    onSelectPeer = null,
  } = $props();

  let textInput = $state('');
  let messagesContainer = $state(null);
  let textareaElement = $state(null);
  let fileInput = $state(null);
  let folderInput = $state(null);

  // Mention Autocomplete state
  let mentionMenuOpen = $state(false);
  let mentionQuery = $state('');
  let mentionAtIndex = $state(-1);
  let selectedMentionIndex = $state(0);

  let mentionCandidates = $derived.by(() => {
    if (!mentionMenuOpen) return [];
    return filterMentionCandidates(peers, currentUser, mentionQuery);
  });

  // Reply state
  let replyingTo = $state(null);
  let highlightedMsgId = $state(null);

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

  // Expanded messages tracking: all long messages and code are collapsed by default!
  let expandedMessageIds = $state(new Set());

  // Emoji Picker & Reactions state
  let isInputEmojiPickerOpen = $state(false);
  let activeReactionPickerMsgId = $state(null);
  let recentlyReactedAnim = $state({});

  function toggleReactionPicker(msgId) {
    if (activeReactionPickerMsgId === msgId) {
      activeReactionPickerMsgId = null;
    } else {
      activeReactionPickerMsgId = msgId;
    }
  }

  function handleToggleReaction(msgId, emoji) {
    const key = `${msgId}_${emoji}`;
    recentlyReactedAnim = { ...recentlyReactedAnim, [key]: true };
    setTimeout(() => {
      const next = { ...recentlyReactedAnim };
      delete next[key];
      recentlyReactedAnim = next;
    }, 400);

    onToggleReaction?.(msgId, emoji);
    activeReactionPickerMsgId = null;
  }

  function handleInsertEmoji(emoji) {
    if (!textareaElement) {
      textInput += emoji;
      return;
    }
    const start = textareaElement.selectionStart ?? textInput.length;
    const end = textareaElement.selectionEnd ?? textInput.length;
    const before = textInput.slice(0, start);
    const after = textInput.slice(end);
    textInput = before + emoji + after;
    tick().then(() => {
      if (textareaElement) {
        const nextPos = start + emoji.length;
        textareaElement.focus();
        textareaElement.setSelectionRange(nextPos, nextPos);
      }
    });
  }

  function toggleExpand(msgId) {
    const next = new Set(expandedMessageIds);
    if (next.has(msgId)) {
      next.delete(msgId);
    } else {
      next.add(msgId);
    }
    expandedMessageIds = next;
  }

  function isExpanded(msgId) {
    return expandedMessageIds.has(msgId);
  }

  // Markdown preview mode tracking: Set of snippet message IDs displayed in rendered preview mode
  let previewMessageIds = $state(new Set());

  function toggleMarkdownPreview(msgId) {
    const next = new Set(previewMessageIds);
    if (next.has(msgId)) {
      next.delete(msgId);
    } else {
      next.add(msgId);
    }
    previewMessageIds = next;
  }

  function isMarkdownPreview(msgId) {
    return previewMessageIds.has(msgId);
  }

  // Thresholds for collapsing long text and code
  const MAX_TEXT_CHARS = 400;
  const MAX_TEXT_LINES = 7;
  const MAX_CODE_LINES = 14;

  function getTextPreview(text, isMsgExpanded) {
    if (!text) return { displayText: '', isLong: false, remainingLines: 0, totalLength: 0 };
    const lines = text.split('\n');
    const isLong = text.length > MAX_TEXT_CHARS || lines.length > MAX_TEXT_LINES;
    if (!isLong || isMsgExpanded) {
      return { displayText: text, isLong, remainingLines: 0, totalLength: text.length };
    }

    let preview = lines.slice(0, MAX_TEXT_LINES).join('\n');
    if (preview.length > MAX_TEXT_CHARS) {
      preview = preview.slice(0, MAX_TEXT_CHARS).trimEnd() + '...';
    } else if (lines.length > MAX_TEXT_LINES) {
      preview += '\n...';
    }

    return {
      displayText: preview,
      isLong: true,
      remainingLines: Math.max(0, lines.length - MAX_TEXT_LINES),
      totalLength: text.length,
    };
  }

  function getCodePreview(codeContent, language, isSnippetExpanded) {
    if (!codeContent) return { displayLines: [], isLong: false, totalLines: 0, hiddenLines: 0 };
    const highlightedLines = highlightCodeLines(codeContent, language);
    const totalLines = highlightedLines.length;
    const isLong = totalLines > MAX_CODE_LINES;
    const displayLines = !isLong || isSnippetExpanded ? highlightedLines : highlightedLines.slice(0, MAX_CODE_LINES);

    return {
      displayLines,
      isLong,
      totalLines,
      hiddenLines: Math.max(0, totalLines - MAX_CODE_LINES),
    };
  }

  // Scroll and pagination tracking
  let isNearBottom = $state(true);
  let hasNewUnreadBelow = $state(false);
  let lastPeerId = $state(null);
  let lastMessageId = $state(null);
  let isInitialScrollDone = $state(false);
  let isPrepending = false;

  // Auto scroll to bottom
  async function scrollToBottom(smooth = true) {
    await tick();
    if (messagesContainer) {
      messagesContainer.scrollTo({
        top: messagesContainer.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto',
      });
      isNearBottom = true;
      hasNewUnreadBelow = false;
    }
  }

  // Handle scroll events on message container
  function handleContainerScroll() {
    if (!messagesContainer) return;
    const { scrollTop, scrollHeight, clientHeight } = messagesContainer;

    // Check if user is scrolled near bottom
    const nearBottom = scrollHeight - scrollTop - clientHeight < 150;
    isNearBottom = nearBottom;
    if (nearBottom) {
      hasNewUnreadBelow = false;
    }

    // Check if user scrolled near top to lazy-load older messages
    if (scrollTop < 120 && hasMore && !isLoadingOlder && !isPrepending && onLoadOlder) {
      triggerLoadOlder();
    }
  }

  // Lazy load older messages while seamlessly retaining scroll position using anchor element
  async function triggerLoadOlder() {
    if (!messagesContainer || !onLoadOlder || isLoadingOlder || isPrepending || !hasMore || messages.length === 0) return;

    isPrepending = true;

    // Record top offset of current oldest message element before prepending
    const anchorId = messages[0]?.id;
    const anchorEl = anchorId ? document.getElementById(`msg-${anchorId}`) : null;
    const anchorOffsetBefore = anchorEl ? anchorEl.getBoundingClientRect().top : null;
    const prevScrollHeight = messagesContainer.scrollHeight;
    const prevScrollTop = messagesContainer.scrollTop;

    try {
      const loaded = await onLoadOlder();
      if (!loaded) return;
      await tick();

      if (messagesContainer) {
        const anchorElAfter = anchorId ? document.getElementById(`msg-${anchorId}`) : null;
        if (anchorElAfter && anchorOffsetBefore !== null) {
          // Precise anchor delta restoration
          const anchorOffsetAfter = anchorElAfter.getBoundingClientRect().top;
          const delta = anchorOffsetAfter - anchorOffsetBefore;
          messagesContainer.scrollTop += delta;
        } else {
          // Fallback to scrollHeight diff
          const newScrollHeight = messagesContainer.scrollHeight;
          const heightDiff = newScrollHeight - prevScrollHeight;
          messagesContainer.scrollTop = prevScrollTop + heightDiff;
        }
      }
    } finally {
      setTimeout(() => {
        isPrepending = false;
      }, 150);
    }
  }

  // When selected peer changes, reset scroll tracking
  $effect(() => {
    const currentPeerId = selectedPeer?.id || 'broadcast';
    if (currentPeerId !== lastPeerId) {
      lastPeerId = currentPeerId;
      lastMessageId = null;
      isInitialScrollDone = false;
      isNearBottom = true;
      hasNewUnreadBelow = false;
    }
  });

  // When messages update:
  $effect(() => {
    if (messages.length > 0) {
      const currentLast = messages[messages.length - 1];

      if (!isInitialScrollDone) {
        // Initial load for this peer: jump directly to bottom
        isInitialScrollDone = true;
        lastMessageId = currentLast ? currentLast.id : null;
        scrollToBottom(false);
      } else if (currentLast && currentLast.id !== lastMessageId) {
        // A new message was appended to the bottom!
        const isFromMe = currentLast.sender_id === currentUser?.id;
        lastMessageId = currentLast.id;

        if (isFromMe || isNearBottom) {
          scrollToBottom(true);
        } else {
          hasNewUnreadBelow = true;
        }
      }
    }
  });

  // If initial batch doesn't fill the container and hasMore is true, auto-fetch until scrollable
  $effect(() => {
    if (messages.length > 0 && hasMore && !isLoadingOlder && !isPrepending && isInitialScrollDone) {
      tick().then(() => {
        if (messagesContainer && messagesContainer.scrollHeight <= messagesContainer.clientHeight + 60) {
          triggerLoadOlder();
        }
      });
    }
  });

  function getSenderDisplayName(senderId) {
    if (!senderId) return 'Peer';
    if (senderId === currentUser?.id) return 'You';
    if (senderId === selectedPeer?.id) return selectedPeer?.display_name || 'Peer';
    const peer = peers.find((p) => p.id === senderId);
    return peer?.display_name || 'Peer';
  }

  function startReply(msg) {
    replyingTo = msg;
    tick().then(() => {
      if (textareaElement) {
        textareaElement.focus();
      }
    });
  }

  function cancelReply() {
    replyingTo = null;
  }

  function scrollToMessage(targetId) {
    if (!targetId) return;
    const el = document.getElementById(`msg-${targetId}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      highlightedMsgId = targetId;
      setTimeout(() => {
        if (highlightedMsgId === targetId) {
          highlightedMsgId = null;
        }
      }, 2000);
    }
  }

  function getReplySnippet(replyMsg) {
    if (!replyMsg) return 'Original message';
    if (replyMsg.type === 'text') return replyMsg.body;
    if (replyMsg.type === 'code') {
      const firstLine = replyMsg.snippet?.code_content?.split('\n')[0] || '';
      return replyMsg.body ? `${replyMsg.body} — ${firstLine}` : firstLine || 'Code snippet';
    }
    if (replyMsg.type === 'file') {
      return replyMsg.transfer?.file_name || 'File transfer';
    }
    return replyMsg.body || 'Original message';
  }

  function handleSendText() {
    const trimmed = textInput.trim();
    if (!trimmed) return;
    const replyId = replyingTo?.id || null;
    onSendMessage(trimmed, replyId);
    textInput = '';
    replyingTo = null;
    mentionMenuOpen = false;
    scrollToBottom();
  }

  function updateMentionContext() {
    if (!textareaElement) {
      mentionMenuOpen = false;
      return;
    }
    const cursorPos = textareaElement.selectionStart;
    const ctx = extractMentionContext(textInput, cursorPos);
    if (ctx.isOpen) {
      mentionMenuOpen = true;
      mentionQuery = ctx.query;
      mentionAtIndex = ctx.atIndex;
      selectedMentionIndex = 0;
    } else {
      mentionMenuOpen = false;
      mentionAtIndex = -1;
    }
  }

  function applyMention(item) {
    if (!item || !textareaElement) return;
    const nameToInsert = item.name;
    const before = textInput.slice(0, mentionAtIndex);
    const after = textInput.slice(textareaElement.selectionStart);
    const inserted = `@${nameToInsert} `;
    textInput = before + inserted + after;
    mentionMenuOpen = false;

    tick().then(() => {
      if (textareaElement) {
        const newPos = before.length + inserted.length;
        textareaElement.focus();
        textareaElement.setSelectionRange(newPos, newPos);
      }
    });
  }

  function triggerMentionButton() {
    if (!textareaElement) return;
    textareaElement.focus();
    const cursorPos = textareaElement.selectionStart;
    const before = textInput.slice(0, cursorPos);
    const after = textInput.slice(cursorPos);
    const needsLeadingSpace = cursorPos > 0 && !/\s/.test(before.slice(-1));
    const insertText = needsLeadingSpace ? ' @' : '@';

    textInput = before + insertText + after;
    mentionAtIndex = before.length + (needsLeadingSpace ? 1 : 0);
    mentionQuery = '';
    mentionMenuOpen = true;
    selectedMentionIndex = 0;

    tick().then(() => {
      if (textareaElement) {
        const newPos = before.length + insertText.length;
        textareaElement.setSelectionRange(newPos, newPos);
      }
    });
  }

  function insertMention(peerName) {
    if (!peerName) return;
    const cleanName = peerName === 'You' ? (currentUser?.display_name || 'me') : peerName;
    const prefix = textInput && !textInput.endsWith(' ') ? ' ' : '';
    textInput = textInput + `${prefix}@${cleanName} `;
    mentionMenuOpen = false;
    tick().then(() => {
      if (textareaElement) {
        textareaElement.focus();
        textareaElement.setSelectionRange(textInput.length, textInput.length);
      }
    });
  }

  function handleMentionClick(segment) {
    if (segment.peer && onSelectPeer) {
      onSelectPeer(segment.peer.id);
    } else {
      insertMention(segment.name);
    }
  }

  function handleWindowPointerDown(e) {
    if (mentionMenuOpen && !e.target?.closest?.('[role="listbox"]') && e.target !== textareaElement) {
      mentionMenuOpen = false;
    }
    if (isInputEmojiPickerOpen && !e.target?.closest?.('[aria-label="Send Emoji"]') && !e.target?.closest?.('#input-emoji-btn')) {
      isInputEmojiPickerOpen = false;
    }
    if (activeReactionPickerMsgId && !e.target?.closest?.('[aria-label="Reaction Picker"]') && !e.target?.closest?.('.reaction-trigger-btn')) {
      activeReactionPickerMsgId = null;
    }
  }

  function handleKeyDown(e) {
    // Autocomplete keyboard navigation
    if (mentionMenuOpen && mentionCandidates.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        selectedMentionIndex = (selectedMentionIndex + 1) % mentionCandidates.length;
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        selectedMentionIndex = (selectedMentionIndex - 1 + mentionCandidates.length) % mentionCandidates.length;
        return;
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        applyMention(mentionCandidates[selectedMentionIndex]);
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        mentionMenuOpen = false;
        return;
      }
    }

    if (e.key === 'Escape' && replyingTo) {
      e.preventDefault();
      cancelReply();
      return;
    }
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
    const replyId = replyingTo?.id || null;
    onSendCode({ ...payload, replyToId: replyId });
    replyingTo = null;
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
    const replyId = replyingTo?.id || null;
    onUploadFiles({
      files,
      isFolder,
      folderName: archiveName,
      expiration,
      devIgnore,
      replyToId: replyId,
    });
    replyingTo = null;
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

  async function copyCodeRaw(msgId, codeContent) {
    await copyToClipboard(codeContent);
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

<svelte:window onpointerdown={handleWindowPointerDown} />

<div
  role="region"
  aria-label="Chat messages and drop area"
  class="relative flex-1 w-full h-full flex flex-col bg-slate-900 overflow-hidden min-h-0"
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
  <header class="h-14 sm:h-16 px-3 sm:px-6 bg-slate-950/90 border-b border-slate-800/80 flex items-center justify-between flex-shrink-0 backdrop-blur-sm z-10">
    <div class="flex items-center gap-2 sm:gap-3.5 min-w-0">
      <!-- Mobile Back Button -->
      {#if onBackToPeers}
        <button
          type="button"
          onclick={onBackToPeers}
          class="md:hidden flex items-center gap-1.5 px-2.5 py-1.5 -ml-1 text-slate-300 hover:text-white bg-slate-900/90 hover:bg-slate-800 border border-slate-750 rounded-xl text-xs font-mono transition-colors cursor-pointer flex-shrink-0"
          title="Back to peers list"
          aria-label="Back to peer list"
        >
          <ChevronLeft class="w-4 h-4 text-cyan-400" />
          <span class="text-xs font-medium">Peers</span>
          {#if totalUnreadCount > 0}
            <span class="px-1.5 py-0.2 text-[10px] font-bold rounded-full bg-cyan-500 text-slate-950 font-mono">
              {totalUnreadCount}
            </span>
          {/if}
        </button>
      {/if}

      {#if isBroadcast}
        <div class="w-8 h-8 sm:w-10 sm:h-10 rounded-xl bg-gradient-to-tr from-cyan-500/20 to-amber-500/20 border border-cyan-500/30 flex items-center justify-center text-cyan-400 flex-shrink-0">
          <Radio class="w-4 h-4 sm:w-5 sm:h-5 animate-pulse" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-1.5 sm:gap-2">
            <h1 class="text-xs sm:text-sm font-semibold text-slate-100 font-mono tracking-tight truncate">
              LAN Broadcast
            </h1>
            <span class="text-[9px] sm:text-[10px] px-1.5 sm:px-2 py-0.5 rounded-full bg-cyan-950 text-cyan-400 font-bold border border-cyan-800/50 uppercase tracking-wider flex-shrink-0">
              All Peers
            </span>
          </div>
          <p class="text-[10px] sm:text-xs text-slate-400 font-mono mt-0.5 truncate">
            <span class="hidden sm:inline">Messages and files are visible to all connected LAN developers</span>
            <span class="sm:hidden">Public feed • LAN network</span>
          </p>
        </div>
      {:else}
        <div class="relative w-8 h-8 sm:w-10 sm:h-10 rounded-xl bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-200 text-xs sm:text-sm flex-shrink-0">
          {(selectedPeer?.display_name || 'P')[0]?.toUpperCase()}
          {#if selectedPeer?.is_online}
            <div class="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 sm:w-3 sm:h-3 rounded-full bg-emerald-400 border-2 border-slate-900 animate-pulse shadow-sm shadow-emerald-400"></div>
          {:else}
            <div class="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 sm:w-3 sm:h-3 rounded-full bg-slate-600 border-2 border-slate-900"></div>
          {/if}
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-1.5 sm:gap-2">
            <h1 class="text-xs sm:text-sm font-semibold text-slate-100 font-mono truncate">
              {selectedPeer?.display_name}
            </h1>
            {#if selectedPeer?.is_online}
              <span class="text-[9px] sm:text-[10px] px-1.5 sm:px-2 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 font-bold border border-emerald-800/40 font-mono flex-shrink-0">
                Online
              </span>
            {:else}
              <span class="text-[9px] sm:text-[10px] px-1.5 sm:px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono flex-shrink-0">
                Offline
              </span>
            {/if}
          </div>
          <p class="text-[10px] sm:text-xs text-slate-400 font-mono mt-0.5 flex items-center gap-1.5 truncate">
            <span class="truncate">IP: <strong class="text-slate-300">{selectedPeer?.ip_address}</strong></span>
            <span>•</span>
            <span class="flex-shrink-0">Direct 1-to-1</span>
          </p>
        </div>
      {/if}
    </div>

    <!-- Right header stats & notification trigger -->
    <div class="flex items-center gap-2 sm:gap-3 text-xs text-slate-400 font-mono flex-shrink-0">
      {#if onOpenNotificationSettings}
        <button
          type="button"
          onclick={onOpenNotificationSettings}
          class="relative flex items-center gap-1.5 px-2.5 py-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-750 hover:border-slate-700 text-slate-300 hover:text-white rounded-lg transition-colors cursor-pointer"
          title="Notification Settings"
          aria-label="Notification Settings"
        >
          {#if !notificationSettings?.enabled}
            <BellOff class="w-3.5 h-3.5 text-slate-500" />
          {:else if !notificationSettings?.soundEnabled}
            <Bell class="w-3.5 h-3.5 text-slate-400" />
          {:else}
            <BellRing class="w-3.5 h-3.5 text-cyan-400" />
          {/if}
          <span class="hidden sm:inline text-[11px]">Alerts</span>
          {#if notificationSettings?.enabled && notificationSettings?.desktopEnabled}
            <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse"></span>
          {/if}
        </button>
      {/if}

      {#if onOpenSecuritySettings}
        <button
          type="button"
          onclick={onOpenSecuritySettings}
          class="flex items-center gap-1.5 px-2.5 py-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-750 hover:border-slate-700 text-slate-300 hover:text-cyan-300 rounded-lg transition-colors cursor-pointer"
          title="Device Security & Access Password"
          aria-label="Security Settings"
        >
          <ShieldCheck class="w-3.5 h-3.5 text-cyan-400" />
          <span class="hidden sm:inline text-[11px]">Security</span>
        </button>
      {/if}

      <div class="hidden sm:flex items-center gap-1.5 px-3 py-1.5 bg-slate-900 rounded-lg border border-slate-800">
        <Terminal class="w-3.5 h-3.5 text-cyan-400" />
        <span>DevDrop Channel</span>
      </div>
    </div>
  </header>

  <!-- Smart Paste Toast Prompt -->
  {#if smartPastePrompt}
    <div class="bg-cyan-950/90 border-b border-cyan-800 px-3 sm:px-6 py-2 sm:py-2.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs text-cyan-200 animate-in slide-in-from-top duration-200 z-10 shadow-lg flex-shrink-0">
      <div class="flex items-center gap-2 sm:gap-2.5 min-w-0">
        <Sparkles class="w-4 h-4 text-cyan-400 flex-shrink-0 animate-spin" />
        <span class="font-medium text-[11px] sm:text-xs truncate">
          Multi-line code detected ({smartPastePrompt.linesCount} lines). Open in Code Snippet Editor?
        </span>
      </div>
      <div class="flex items-center gap-2 flex-shrink-0 self-end sm:self-auto">
        <button
          onclick={openSmartPasteInEditor}
          class="px-2.5 sm:px-3 py-1 bg-cyan-600 hover:bg-cyan-500 text-white font-semibold text-xs rounded-md shadow transition-colors cursor-pointer"
        >
          Open in Editor
        </button>
        <button
          onclick={dismissSmartPaste}
          class="px-2 py-1 text-slate-400 hover:text-white text-xs transition-colors"
        >
          Dismiss
        </button>
      </div>
    </div>
  {/if}

  <!-- Message History Scroll Area -->
  <div
    bind:this={messagesContainer}
    onscroll={handleContainerScroll}
    class="flex-1 overflow-y-auto min-h-0 p-3 sm:p-5 md:p-6 space-y-3 sm:space-y-4 md:space-y-5 select-text [overflow-anchor:none]"
    style="overflow-anchor: none;"
  >
    {#if isLoadingMessages}
      <div class="h-full flex flex-col items-center justify-center text-center p-8 text-slate-500">
        <Loader2 class="w-8 h-8 text-cyan-400 animate-spin mb-3" />
        <span class="text-xs font-mono text-slate-400">Loading conversation...</span>
      </div>
    {:else if messages.length === 0}
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
      <!-- Lazy load top indicator or beginning marker -->
      {#if isLoadingOlder}
        <div class="flex items-center justify-center py-2.5 text-xs font-mono text-cyan-400 gap-2 bg-slate-950/70 border border-slate-800 rounded-xl my-2">
          <Loader2 class="w-4 h-4 animate-spin text-cyan-400" />
          <span>Loading earlier messages...</span>
        </div>
      {:else if hasMore}
        <div class="flex items-center justify-center my-3">
          <button
            type="button"
            onclick={triggerLoadOlder}
            class="flex items-center gap-2 px-4 py-1.5 rounded-full bg-slate-950/80 hover:bg-slate-800 text-slate-300 hover:text-cyan-300 border border-slate-800 hover:border-cyan-500/50 text-xs font-mono transition-all shadow-md active:scale-95 cursor-pointer"
          >
            <ChevronUp class="w-3.5 h-3.5 text-cyan-400" />
            <span>Load earlier messages</span>
          </button>
        </div>
      {:else}
        <div class="flex items-center justify-center my-3">
          <div class="flex items-center gap-2 px-3 py-1 rounded-full bg-slate-950/70 border border-slate-800 text-[11px] font-mono text-slate-500 shadow-sm">
            <Radio class="w-3 h-3 text-cyan-400/70" />
            <span>Beginning of conversation history</span>
          </div>
        </div>
      {/if}

      {#each messages as msg, index (msg.id)}
        {@const isMe = msg.sender_id === currentUser?.id}
        {@const prevSnippet = msg.type === 'code' ? findPreviousSnippet(index) : null}
        {@const targetReply = msg.reply_to || (msg.reply_to_id ? messages.find((m) => m.id === msg.reply_to_id) : null)}
        {@const isHighlighted = highlightedMsgId === msg.id}
        {@const msgMentionsMe = !isMe && isUserMentioned(msg.body, currentUser, peers)}

        <div
          id={`msg-${msg.id}`}
          class="flex flex-col {isMe ? 'items-end' : 'items-start'} group transition-all duration-300 rounded-2xl {isHighlighted
            ? 'ring-2 ring-cyan-400 bg-cyan-950/40 p-2 shadow-lg shadow-cyan-500/20'
            : msgMentionsMe
              ? 'ring-1 ring-amber-500/50 bg-amber-950/15 p-1.5 shadow-sm shadow-amber-500/10'
              : 'p-0.5'}"
        >
          <!-- Sender Info, Timestamp & Reply/Mention actions -->
          <div class="flex items-center gap-1.5 sm:gap-2 mb-1 px-1 text-[11px] text-slate-400 font-mono">
            <span class="font-semibold {isMe ? 'text-cyan-400' : 'text-slate-300'}">
              {isMe ? 'You' : getSenderDisplayName(msg.sender_id)}
            </span>
            <span>•</span>
            <span>{formatRelativeTime(msg.created_at)}</span>

            {#if msgMentionsMe}
              <span class="inline-flex items-center gap-1 px-1.5 py-0.2 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/40 text-[10px] font-mono font-semibold">
                <AtSign class="w-2.5 h-2.5 text-amber-400" />
                <span>Mentioned you</span>
              </span>
            {/if}

            <!-- Quick Reply button on hover / touch -->
            <button
              onclick={() => startReply(msg)}
              class="opacity-75 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity ml-1.5 flex items-center gap-1 text-[10px] text-slate-400 hover:text-cyan-300 hover:bg-slate-800/80 px-1.5 py-0.5 rounded cursor-pointer"
              title="Reply to this message"
            >
              <Reply class="w-3 h-3 text-cyan-400" />
              <span>Reply</span>
            </button>

            <!-- Quick Mention button on hover / touch -->
            {#if !isMe}
              <button
                onclick={() => insertMention(getSenderDisplayName(msg.sender_id))}
                class="opacity-75 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity ml-0.5 flex items-center gap-1 text-[10px] text-slate-400 hover:text-cyan-300 hover:bg-slate-800/80 px-1.5 py-0.5 rounded cursor-pointer"
                title="Mention {getSenderDisplayName(msg.sender_id)}"
              >
                <AtSign class="w-3 h-3 text-cyan-400" />
                <span>Mention</span>
              </button>
            {/if}

            <!-- Quick React trigger button -->
            <button
              onclick={() => toggleReactionPicker(msg.id)}
              class="reaction-trigger-btn opacity-75 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity ml-0.5 flex items-center gap-1 text-[10px] text-slate-400 hover:text-cyan-300 hover:bg-slate-800/80 px-1.5 py-0.5 rounded cursor-pointer"
              title="React with emoji"
            >
              <SmilePlus class="w-3 h-3 text-cyan-400" />
              <span>React</span>
            </button>
          </div>

          <!-- Floating Quick-Reaction Bar (reveals on hover) -->
          <div class="opacity-0 group-hover:opacity-100 transition-all duration-150 mb-1 flex items-center gap-0.5 px-2 py-0.5 rounded-full bg-slate-950/95 backdrop-blur-md border border-slate-750/90 shadow-2xl z-20 animate-reaction-bar pointer-events-none group-hover:pointer-events-auto">
            {#each QUICK_REACTIONS.slice(0, 8) as em}
              <button
                type="button"
                onclick={(e) => {
                  e.stopPropagation();
                  handleToggleReaction(msg.id, em);
                }}
                class="text-base sm:text-lg hover:scale-135 active:scale-95 transition-transform duration-100 px-1 py-0.5 flex items-center justify-center cursor-pointer select-none"
                title="React with {em}"
              >
                <span>{em}</span>
              </button>
            {/each}
            <button
              type="button"
              onclick={(e) => {
                e.stopPropagation();
                toggleReactionPicker(msg.id);
              }}
              class="w-6 h-6 ml-0.5 rounded-full bg-slate-800/90 hover:bg-slate-750 text-slate-300 hover:text-cyan-300 flex items-center justify-center transition-all cursor-pointer active:scale-95"
              title="More emoji reactions"
            >
              <SmilePlus class="w-3.5 h-3.5" />
            </button>
          </div>

          <!-- Full Emoji Reaction Picker Popover (if open for this message) -->
          {#if activeReactionPickerMsgId === msg.id}
            <div
              class="relative z-40 my-1.5 animate-in fade-in zoom-in-95 duration-150"
              role="region"
              aria-label="Reaction Picker"
            >
              <EmojiPicker
                mode="reaction"
                title="React to message"
                onSelect={(em) => handleToggleReaction(msg.id, em)}
                onClose={() => (activeReactionPickerMsgId = null)}
              />
            </div>
          {/if}

          <!-- Quoted Reply Card (if replying to another message) -->
          {#if targetReply}
            <button
              type="button"
              onclick={() => scrollToMessage(targetReply.id)}
              class="mb-1.5 max-w-[92%] sm:max-w-2xl text-left px-2.5 sm:px-3 py-1.5 rounded-xl border-l-3 border-cyan-400 bg-slate-950/80 hover:bg-slate-800/90 text-xs transition-colors cursor-pointer group/quote flex items-start gap-2 shadow-sm"
              title="Jump to quoted message"
            >
              <Reply class="w-3 h-3 text-cyan-400 mt-0.5 flex-shrink-0 group-hover/quote:translate-x-0.5 transition-transform" />
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5 text-[10px] font-mono font-semibold text-cyan-300">
                  <span>{getSenderDisplayName(targetReply.sender_id)}</span>
                  {#if targetReply.type === 'code'}
                    <span class="text-[9px] px-1 py-0.2 rounded bg-cyan-950 text-cyan-400 border border-cyan-800/80 uppercase font-mono">
                      {targetReply.snippet?.language || 'code'}
                    </span>
                  {:else if targetReply.type === 'file'}
                    <span class="text-[9px] px-1 py-0.2 rounded bg-amber-950 text-amber-400 border border-amber-800/80 uppercase font-mono">
                      file
                    </span>
                  {/if}
                </div>
                <p class="text-[11px] text-slate-300 truncate mt-0.5 font-sans">
                  {getReplySnippet(targetReply)}
                </p>
              </div>
            </button>
          {/if}

          <!-- Message Body by Type -->
          {#if msg.type === 'text'}
            {@const emojiOnly = getEmojiOnlyInfo(msg.body)}
            {#if emojiOnly.isEmojiOnly}
              <!-- Big Emoji Rendering for 1-3 emojis -->
              <div class="relative py-1 px-1.5 select-text {isMe ? 'text-right' : 'text-left'}">
                <span class="big-emoji-display cursor-default" title={msg.body}>
                  {msg.body.trim()}
                </span>
              </div>
            {:else}
              {@const textPreview = getTextPreview(msg.body, isExpanded(msg.id))}
              {@const firstUrl = extractFirstUrl(msg.body)}
              <div
                class="relative max-w-[88%] sm:max-w-xl md:max-w-2xl px-3.5 py-2 sm:px-4 sm:py-2.5 rounded-2xl text-xs sm:text-sm leading-relaxed whitespace-pre-wrap break-words font-sans shadow-md {isMe
                  ? 'bg-cyan-600 text-white rounded-br-xs'
                  : 'bg-slate-800 border border-slate-700 text-slate-100 rounded-bl-xs'}"
              >
              <div class="{!isExpanded(msg.id) && textPreview.isLong ? 'max-h-52 overflow-hidden relative' : ''} {isExpanded(msg.id) && textPreview.totalLength > 1500 ? 'max-h-[500px] overflow-y-auto pr-1' : ''}">
                {#each parseMessageSegments(textPreview.displayText, peers, currentUser) as segment}
                  {#if segment.type === 'link'}
                    <a
                      href={segment.href}
                      target="_blank"
                      rel="noopener noreferrer"
                      class="font-medium underline underline-offset-2 break-all transition-colors {isMe
                        ? 'text-cyan-100 hover:text-white decoration-cyan-300/70 hover:decoration-white'
                        : 'text-cyan-400 hover:text-cyan-300 decoration-cyan-500/50 hover:decoration-cyan-300'}"
                      onclick={(e) => e.stopPropagation()}
                    >{segment.text}</a>
                  {:else if segment.type === 'mention'}
                    {#if segment.isAll}
                      <span
                        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] sm:text-xs font-mono font-bold align-baseline mx-0.5 {isMe
                          ? 'bg-purple-900/70 text-purple-200 border border-purple-400/50'
                          : 'bg-purple-950/80 text-purple-300 border border-purple-700/60'}"
                        title="Mentioned all LAN peers"
                      >
                        <Radio class="w-3 h-3 text-purple-400 flex-shrink-0 animate-pulse" />
                        <span>{segment.text}</span>
                      </span>
                    {:else if segment.isMe}
                      <span
                        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] sm:text-xs font-mono font-bold align-baseline mx-0.5 shadow-xs {isMe
                          ? 'bg-amber-400/30 text-amber-100 border border-amber-300/60 ring-1 ring-amber-400/40'
                          : 'bg-amber-500/20 text-amber-300 border border-amber-400/50 ring-1 ring-amber-400/30'}"
                        title="You were mentioned in this message!"
                      >
                        <AtSign class="w-3 h-3 text-amber-400 flex-shrink-0" />
                        <span>{segment.text}</span>
                      </span>
                    {:else}
                      <button
                        type="button"
                        onclick={(e) => {
                          e.stopPropagation();
                          handleMentionClick(segment);
                        }}
                        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] sm:text-xs font-mono font-medium align-baseline mx-0.5 transition-all cursor-pointer {isMe
                          ? 'bg-cyan-700/60 hover:bg-cyan-700 text-cyan-100 border border-cyan-400/40'
                          : 'bg-cyan-950/80 hover:bg-cyan-900/90 text-cyan-300 hover:text-white border border-cyan-800/60 hover:border-cyan-600'}"
                        title={segment.peer ? `Direct message ${segment.peer.display_name} (${segment.peer.ip_address})` : `Mentioned: ${segment.name}`}
                      >
                        <AtSign class="w-3 h-3 text-cyan-400 flex-shrink-0" />
                        <span>{segment.text}</span>
                        {#if segment.peer?.is_online}
                          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 flex-shrink-0"></span>
                        {/if}
                      </button>
                    {/if}
                  {:else}
                    {segment.text}
                  {/if}
                {/each}
              </div>

              {#if firstUrl}
                <LinkPreviewCard url={firstUrl} {isMe} />
              {/if}

              {#if textPreview.isLong}
                <div class="mt-2 pt-1.5 border-t {isMe ? 'border-cyan-500/50' : 'border-slate-700/80'} flex items-center justify-between gap-3 text-xs font-mono">
                  <span class="text-[11px] {isMe ? 'text-cyan-100/80' : 'text-slate-400'}">
                    {#if !isExpanded(msg.id)}
                      {textPreview.remainingLines > 0 ? `+${textPreview.remainingLines} lines hidden` : 'Long message collapsed'}
                    {:else}
                      Full message expanded
                    {/if}
                  </span>
                  <button
                    type="button"
                    onclick={() => toggleExpand(msg.id)}
                    class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md text-[11px] font-mono font-medium transition-colors cursor-pointer {isMe
                      ? 'bg-cyan-700 hover:bg-cyan-800 text-white shadow-sm'
                      : 'bg-slate-900/90 hover:bg-slate-900 text-cyan-300 border border-slate-700 shadow-sm'}"
                  >
                    {#if isExpanded(msg.id)}
                      <ChevronUp class="w-3.5 h-3.5" />
                      <span>Show less</span>
                    {:else}
                      <ChevronDown class="w-3.5 h-3.5" />
                      <span>Show more</span>
                    {/if}
                  </button>
                </div>
              {/if}
            </div>
          {/if}

          {:else if msg.type === 'code' && msg.snippet}
            {@const isMarkdown = normalizeLang(msg.snippet.language) === 'markdown'}
            {@const inPreview = isMarkdown && isMarkdownPreview(msg.id)}
            {@const codeInfo = getCodePreview(msg.snippet.code_content, msg.snippet.language, isExpanded(msg.id))}
            <!-- Code Snippet Card -->
            <div class="w-full max-w-[96%] sm:max-w-2xl md:max-w-3xl rounded-xl overflow-hidden border border-slate-750 bg-slate-950 shadow-xl">
              <!-- Code Card Header -->
              <div class="flex items-center justify-between px-2.5 sm:px-4 py-1.5 sm:py-2 bg-slate-900/90 border-b border-slate-800 gap-1.5">
                <div class="flex items-center gap-1.5 sm:gap-2 min-w-0">
                  <span class="text-[10px] sm:text-xs font-bold font-mono px-1.5 sm:px-2 py-0.5 rounded bg-cyan-950 text-cyan-400 border border-cyan-800/60 uppercase flex-shrink-0">
                    {msg.snippet.language}
                  </span>
                  <span class="text-[10px] sm:text-[11px] font-mono px-1.5 sm:px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700 flex-shrink-0">
                    {codeInfo.totalLines} lines
                  </span>
                  {#if inPreview}
                    <span class="text-[9px] sm:text-[10px] font-mono font-semibold px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-300 border border-emerald-800/60 uppercase flex-shrink-0">
                      Preview
                    </span>
                  {/if}
                  {#if msg.body}
                    <span class="text-xs text-slate-300 font-mono truncate max-w-[120px] sm:max-w-md hidden xs:inline">
                      {#each parseMessageSegments(msg.body, peers, currentUser) as segment}
                        {#if segment.type === 'mention'}
                          <span class="text-cyan-400 font-semibold">{segment.text}</span>
                        {:else if segment.type === 'link'}
                          <a href={segment.href} target="_blank" rel="noopener noreferrer" class="underline text-cyan-400">{segment.text}</a>
                        {:else}
                          {segment.text}
                        {/if}
                      {/each}
                    </span>
                  {/if}
                </div>

                <div class="flex items-center gap-1 sm:gap-2 flex-shrink-0">
                  <!-- Markdown Preview / Code Toggle Button: ONLY for Markdown -->
                  {#if isMarkdown}
                    <button
                      onclick={() => toggleMarkdownPreview(msg.id)}
                      class="flex items-center gap-1 px-2 sm:px-2.5 py-1 text-xs rounded-lg transition-colors cursor-pointer font-medium {inPreview
                        ? 'text-cyan-300 bg-cyan-950/80 hover:bg-cyan-900/80 border border-cyan-700/60 shadow-sm'
                        : 'text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 border border-slate-700'}"
                      title={inPreview ? "Switch to Code view" : "Preview rendered Markdown"}
                    >
                      {#if inPreview}
                        <Code class="w-3.5 h-3.5 text-cyan-400" />
                        <span class="hidden sm:inline">Code</span>
                      {:else}
                        <Eye class="w-3.5 h-3.5 text-cyan-400" />
                        <span class="hidden sm:inline">Preview</span>
                      {/if}
                    </button>
                  {/if}

                  <!-- Diff Button if previous snippet exists -->
                  {#if prevSnippet}
                    <button
                      onclick={() => openDiff(prevSnippet, msg)}
                      class="flex items-center gap-1 px-2 sm:px-2.5 py-1 text-xs text-indigo-300 hover:text-indigo-200 bg-indigo-950/60 hover:bg-indigo-900/70 border border-indigo-700/50 rounded-lg transition-colors cursor-pointer"
                      title="Compare side-by-side with previous snippet"
                    >
                      <GitCompare class="w-3.5 h-3.5" />
                      <span class="hidden sm:inline">Diff</span>
                    </button>
                  {/if}

                  <!-- Copy Raw Button -->
                  <button
                    onclick={() => copyCodeRaw(msg.id, msg.snippet.code_content)}
                    class="flex items-center gap-1 px-2 sm:px-2.5 py-1 text-xs text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-lg transition-colors cursor-pointer"
                    title="Copy Raw Code"
                  >
                    {#if copiedId === msg.id}
                      <Check class="w-3.5 h-3.5 text-emerald-400" />
                      <span class="text-emerald-400 text-xs hidden sm:inline">Copied!</span>
                    {:else}
                      <Copy class="w-3.5 h-3.5" />
                      <span class="hidden sm:inline">Copy</span>
                    {/if}
                  </button>

                  <!-- Toggle Expand/Collapse in Header for long snippets -->
                  {#if codeInfo.isLong}
                    <button
                      onclick={() => toggleExpand(msg.id)}
                      class="flex items-center gap-1 px-2 sm:px-2.5 py-1 text-xs text-cyan-300 hover:text-white bg-cyan-950/60 hover:bg-cyan-900/70 border border-cyan-800/60 rounded-lg transition-colors cursor-pointer"
                      title={isExpanded(msg.id) ? (inPreview ? "Collapse preview" : "Collapse snippet") : (inPreview ? "Expand preview" : "Expand snippet")}
                    >
                      {#if isExpanded(msg.id)}
                        <ChevronUp class="w-3.5 h-3.5" />
                        <span class="hidden sm:inline">Less</span>
                      {:else}
                        <ChevronDown class="w-3.5 h-3.5" />
                        <span class="hidden sm:inline">More</span>
                      {/if}
                    </button>
                  {/if}
                </div>
              </div>

              <!-- Snippet Body: Preview or Code Lines -->
              {#if inPreview}
                <!-- Rendered Markdown View -->
                <div class="relative bg-slate-950">
                  <div class="p-3.5 sm:p-5 overflow-x-auto text-slate-200 select-text {isExpanded(msg.id) ? 'max-h-[600px] overflow-y-auto' : (codeInfo.isLong ? 'max-h-80 overflow-hidden' : '')}">
                    <div class="markdown-preview">
                      {@html renderMarkdown(msg.snippet.code_content)}
                    </div>
                  </div>

                  <!-- Collapsed Bottom Bar for long preview -->
                  {#if codeInfo.isLong && !isExpanded(msg.id)}
                    <div class="relative bg-gradient-to-b from-slate-950/50 via-slate-900/95 to-slate-900 border-t border-slate-800/80 px-3 sm:px-4 py-1.5 sm:py-2 flex items-center justify-between gap-2">
                      <span class="text-[10px] sm:text-[11px] font-mono text-slate-400 truncate">
                        Preview collapsed ({codeInfo.totalLines} lines in source)
                      </span>
                      <button
                        type="button"
                        onclick={() => toggleExpand(msg.id)}
                        class="flex items-center gap-1 sm:gap-1.5 px-2.5 sm:px-3 py-1 bg-cyan-950 hover:bg-cyan-900/80 border border-cyan-800 text-cyan-300 hover:text-white rounded-lg text-xs font-mono font-medium shadow transition-colors cursor-pointer active:scale-95 flex-shrink-0"
                      >
                        <ChevronDown class="w-3.5 h-3.5" />
                        <span class="hidden sm:inline">Expand full preview</span>
                        <span class="sm:hidden">Expand</span>
                      </button>
                    </div>
                  {:else if codeInfo.isLong && isExpanded(msg.id)}
                    <!-- Expanded Bottom Collapse Footer -->
                    <div class="bg-slate-900/90 border-t border-slate-800 px-3 sm:px-4 py-1.5 flex items-center justify-between text-xs font-mono">
                      <span class="text-[10px] sm:text-[11px] text-slate-400">
                        Full preview expanded
                      </span>
                      <button
                        type="button"
                        onclick={() => toggleExpand(msg.id)}
                        class="flex items-center gap-1 px-2.5 py-0.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded transition-colors cursor-pointer"
                      >
                        <ChevronUp class="w-3.5 h-3.5" />
                        <span>Collapse</span>
                      </button>
                    </div>
                  {/if}
                </div>
              {:else}
                <!-- Code Lines with numbering -->
                <div class="relative">
                  <div class="p-2.5 sm:p-3 overflow-x-auto text-[11px] sm:text-xs font-mono leading-relaxed bg-slate-950 text-slate-200 {isExpanded(msg.id) ? 'max-h-[520px] overflow-y-auto' : ''}">
                    <table class="border-collapse w-full">
                      <tbody>
                        {#each codeInfo.displayLines as line, lIdx}
                          <tr class="hover:bg-slate-900/70 transition-colors">
                            <td class="pr-2 sm:pr-4 py-0.5 text-right text-slate-600 select-none w-8 sm:w-10 font-mono text-[10px] sm:text-[11px] align-top">
                              {lIdx + 1}
                            </td>
                            <td class="py-0.5 whitespace-pre font-mono text-slate-200 break-normal">
                              {@html line || '&nbsp;'}
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  </div>

                  <!-- Collapsed Bottom Bar with Gradient Overlay -->
                  {#if codeInfo.isLong && !isExpanded(msg.id)}
                    <div class="relative bg-gradient-to-b from-slate-950/50 via-slate-900/95 to-slate-900 border-t border-slate-800/80 px-3 sm:px-4 py-1.5 sm:py-2 flex items-center justify-between gap-2">
                      <span class="text-[10px] sm:text-[11px] font-mono text-slate-400 truncate">
                        14 of {codeInfo.totalLines} lines ({codeInfo.hiddenLines} hidden)
                      </span>
                      <button
                        type="button"
                        onclick={() => toggleExpand(msg.id)}
                        class="flex items-center gap-1 sm:gap-1.5 px-2.5 sm:px-3 py-1 bg-cyan-950 hover:bg-cyan-900/80 border border-cyan-800 text-cyan-300 hover:text-white rounded-lg text-xs font-mono font-medium shadow transition-colors cursor-pointer active:scale-95 flex-shrink-0"
                      >
                        <ChevronDown class="w-3.5 h-3.5" />
                        <span class="hidden sm:inline">Expand snippet ({codeInfo.totalLines} lines)</span>
                        <span class="sm:hidden">Expand ({codeInfo.totalLines})</span>
                      </button>
                    </div>
                  {:else if codeInfo.isLong && isExpanded(msg.id)}
                    <!-- Expanded Bottom Collapse Footer -->
                    <div class="bg-slate-900/90 border-t border-slate-800 px-3 sm:px-4 py-1.5 flex items-center justify-between text-xs font-mono">
                      <span class="text-[10px] sm:text-[11px] text-slate-400">
                        All {codeInfo.totalLines} lines visible
                      </span>
                      <button
                        type="button"
                        onclick={() => toggleExpand(msg.id)}
                        class="flex items-center gap-1 px-2.5 py-0.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded transition-colors cursor-pointer"
                      >
                        <ChevronUp class="w-3.5 h-3.5" />
                        <span>Collapse</span>
                      </button>
                    </div>
                  {/if}
                </div>
              {/if}
            </div>

          {:else if msg.type === 'file' && msg.transfer}
            <!-- File or Folder Transfer Bubble -->
            <div class="w-full max-w-[92%] sm:max-w-md">
              <FileCard transfer={msg.transfer} isOutgoing={isMe} />
            </div>
          {/if}

          <!-- Reaction Badges Under Message -->
          {#if msg.reactions && msg.reactions.length > 0}
            <div class="flex flex-wrap items-center gap-1.5 mt-1.5 px-0.5 {isMe ? 'justify-end' : 'justify-start'}">
              {#each msg.reactions as grp (grp.emoji)}
                {@const hasMyReaction = grp.user_ids && grp.user_ids.includes(currentUser?.id)}
                {@const animKey = `${msg.id}_${grp.emoji}`}
                <button
                  type="button"
                  onclick={() => handleToggleReaction(msg.id, grp.emoji)}
                  class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono transition-all duration-150 cursor-pointer select-none active:scale-95 shadow-sm {hasMyReaction
                    ? 'bg-cyan-500/25 hover:bg-cyan-500/35 border border-cyan-400/60 text-cyan-200 ring-1 ring-cyan-500/40 shadow-xs shadow-cyan-500/20 font-semibold'
                    : 'bg-slate-900/90 hover:bg-slate-800/90 border border-slate-750 text-slate-300 font-medium hover:border-slate-600'} {recentlyReactedAnim[animKey] ? 'animate-reaction-pop' : ''}"
                  title="{grp.user_names?.length ? grp.user_names.join(', ') : `${grp.count} reaction`}"
                >
                  <span class="text-sm leading-none">{grp.emoji}</span>
                  <span class="text-[11px] font-bold">{grp.count}</span>
                </button>
              {/each}

              <!-- Add Reaction Mini Button (+) -->
              <button
                type="button"
                onclick={() => toggleReactionPicker(msg.id)}
                class="reaction-trigger-btn inline-flex items-center justify-center w-6 h-6 rounded-full bg-slate-900/80 hover:bg-slate-800 text-slate-400 hover:text-cyan-300 border border-slate-750 hover:border-cyan-500/50 transition-all cursor-pointer active:scale-95 shadow-sm"
                title="Add reaction"
              >
                <SmilePlus class="w-3.5 h-3.5" />
              </button>
            </div>
          {/if}
        </div>
      {/each}
    {/if}
  </div>

  <!-- Floating Scroll To Bottom Button -->
  {#if !isNearBottom && messages.length > 0}
    <button
      type="button"
      onclick={() => scrollToBottom(true)}
      class="absolute bottom-20 sm:bottom-24 right-4 sm:right-8 z-30 flex items-center gap-1.5 px-3 py-2 rounded-full bg-slate-900/95 hover:bg-slate-800 text-slate-200 border border-slate-750 hover:border-cyan-500/60 shadow-2xl backdrop-blur-md text-xs font-mono transition-all duration-200 active:scale-95 cursor-pointer group"
      title="Scroll to latest messages"
    >
      <ChevronDown class="w-4 h-4 text-cyan-400 group-hover:translate-y-0.5 transition-transform" />
      {#if hasNewUnreadBelow}
        <span class="w-2 h-2 rounded-full bg-cyan-400 animate-pulse"></span>
        <span class="text-cyan-300 font-semibold text-[11px]">New messages</span>
      {:else}
        <span class="text-slate-400 group-hover:text-slate-200 text-[11px]">Latest</span>
      {/if}
    </button>
  {/if}

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
  <footer class="p-2 sm:p-3.5 pb-[max(0.5rem,env(safe-area-inset-bottom))] bg-slate-950/95 border-t border-slate-800 flex-shrink-0">
    <div class="relative flex flex-col bg-slate-900 border border-slate-700/80 rounded-xl sm:rounded-2xl shadow-xl focus-within:border-cyan-500/80 focus-within:ring-1 focus-within:ring-cyan-500/40 transition-all">
      
      <!-- Replying Banner -->
      {#if replyingTo}
        <div class="flex items-center justify-between px-3 py-1.5 sm:px-3.5 sm:py-2 bg-slate-950/90 border-b border-slate-800 rounded-t-xl sm:rounded-t-2xl animate-in slide-in-from-bottom duration-150 text-xs">
          <div class="flex items-center gap-2 min-w-0">
            <div class="w-5 h-5 sm:w-6 sm:h-6 rounded-lg bg-cyan-950 text-cyan-400 border border-cyan-800/60 flex items-center justify-center flex-shrink-0">
              <Reply class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
            </div>
            <div class="min-w-0">
              <div class="flex items-center gap-1.5">
                <span class="text-[10px] sm:text-[11px] font-mono text-slate-400">Replying to</span>
                <span class="text-[10px] sm:text-[11px] font-mono font-bold text-cyan-300 truncate max-w-[120px] sm:max-w-none">
                  {getSenderDisplayName(replyingTo.sender_id)}
                </span>
                {#if replyingTo.type === 'code'}
                  <span class="text-[9px] px-1.5 py-0.2 rounded bg-cyan-950 text-cyan-400 border border-cyan-800 uppercase font-mono">
                    {replyingTo.snippet?.language || 'code'}
                  </span>
                {:else if replyingTo.type === 'file'}
                  <span class="text-[9px] px-1.5 py-0.2 rounded bg-amber-950 text-amber-400 border border-amber-800 uppercase font-mono">
                    file
                  </span>
                {/if}
              </div>
              <p class="text-[10px] sm:text-[11px] text-slate-400 truncate max-w-[180px] xs:max-w-xs sm:max-w-md font-sans">
                {getReplySnippet(replyingTo)}
              </p>
            </div>
          </div>

          <button
            onclick={cancelReply}
            class="p-1 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors cursor-pointer flex-shrink-0"
            title="Cancel reply (Esc)"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
      {/if}

      <!-- Mention Autocomplete Popover -->
      {#if mentionMenuOpen && mentionCandidates.length > 0}
        <div
          class="absolute bottom-full left-0 right-0 sm:left-2 sm:right-auto sm:w-80 mb-2 z-40 bg-slate-950/95 backdrop-blur-md border border-slate-750 rounded-xl shadow-2xl overflow-hidden animate-in fade-in slide-in-from-bottom-2 duration-150"
          role="listbox"
          aria-label="Mention peer suggestions"
        >
          <div class="px-3 py-1.5 border-b border-slate-800/80 bg-slate-900/60 flex items-center justify-between text-[11px] font-mono text-slate-400">
            <span class="flex items-center gap-1.5 font-semibold text-cyan-400">
              <AtSign class="w-3.5 h-3.5" />
              <span>Mention Peer</span>
            </span>
            <span class="text-[10px] text-slate-500 hidden sm:inline">↑↓ navigate • Enter to select</span>
          </div>

          <div class="max-h-56 overflow-y-auto divide-y divide-slate-800/40 p-1">
            {#each mentionCandidates as item, idx (item.id)}
              {@const isSelected = selectedMentionIndex === idx}
              <button
                type="button"
                onpointerdown={(e) => {
                  e.preventDefault();
                  applyMention(item);
                }}
                class="w-full text-left px-2.5 py-1.5 rounded-lg flex items-center gap-2.5 transition-colors cursor-pointer {isSelected
                  ? 'bg-cyan-950/70 border-l-2 border-cyan-400 text-white'
                  : 'text-slate-300 hover:bg-slate-900/80 hover:text-white'}"
              >
                {#if item.isAll}
                  <div class="w-7 h-7 rounded-lg bg-indigo-950/80 text-indigo-400 border border-indigo-800/60 flex items-center justify-center flex-shrink-0">
                    <Radio class="w-3.5 h-3.5 animate-pulse" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-mono font-bold text-indigo-300">@all</span>
                      <span class="text-[9px] px-1 py-0.2 rounded bg-indigo-950 text-indigo-400 border border-indigo-800/60 uppercase font-mono">
                        Broadcast
                      </span>
                    </div>
                    <p class="text-[10px] text-slate-400 truncate font-mono">
                      {item.subtitle}
                    </p>
                  </div>
                {:else}
                  <div class="relative w-7 h-7 rounded-lg bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-300 text-xs flex-shrink-0">
                    {(item.displayName || 'P')[0]?.toUpperCase()}
                    {#if item.isOnline}
                      <div class="absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full bg-emerald-400 border border-slate-900"></div>
                    {/if}
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-mono font-semibold truncate text-slate-200">
                        @{item.displayName}
                      </span>
                      {#if item.isMe}
                        <span class="text-[9px] px-1 py-0.2 rounded bg-slate-800 text-slate-400 border border-slate-700 font-mono">
                          You
                        </span>
                      {/if}
                      {#if item.isOnline}
                        <span class="text-[9px] px-1 py-0.2 rounded bg-emerald-950 text-emerald-400 border border-emerald-800/50 font-mono">
                          Online
                        </span>
                      {/if}
                    </div>
                    <p class="text-[10px] text-slate-400 truncate font-mono">
                      {item.subtitle}
                    </p>
                  </div>
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Input Emoji Picker Popover -->
      {#if isInputEmojiPickerOpen}
        <div
          class="absolute bottom-full left-0 sm:left-2 mb-2 z-50 animate-in fade-in slide-in-from-bottom-2 duration-150"
          role="region"
          aria-label="Send Emoji"
        >
          <EmojiPicker
            onSelect={handleInsertEmoji}
            onClose={() => (isInputEmojiPickerOpen = false)}
            title="Send Emoji"
            mode="input"
          />
        </div>
      {/if}

      <!-- Textarea -->
      <textarea
        bind:this={textareaElement}
        bind:value={textInput}
        onkeydown={handleKeyDown}
        oninput={updateMentionContext}
        onclick={updateMentionContext}
        onkeyup={updateMentionContext}
        placeholder={replyingTo
          ? `Replying to ${getSenderDisplayName(replyingTo.sender_id)}...`
          : (isBroadcast ? "Type message or @name to mention (Shift+Enter for new line)..." : `Type direct message to ${selectedPeer?.display_name}...`)}
        rows="1"
        class="w-full bg-transparent text-slate-100 placeholder-slate-500 text-sm p-2 sm:p-3 focus:outline-none resize-none font-sans min-h-[38px] max-h-32"
      ></textarea>

      <!-- Input Toolbar -->
      <div class="flex items-center justify-between px-2 sm:px-3 pb-2 pt-1 border-t border-slate-800/60 gap-1.5">
        <!-- Left tools: Code drawer, Mention, File, Folder, Dev-Ignore toggle, Expiration -->
        <div class="flex items-center gap-1 sm:gap-1.5 overflow-x-auto no-scrollbar py-0.5 min-w-0">
          <!-- Open Code Drawer -->
          <button
            onclick={() => openCodeDrawerWith('')}
            class="flex items-center gap-1 px-2 sm:px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-cyan-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer flex-shrink-0"
            title="Open Code Snippet Editor"
          >
            <Code class="w-3.5 h-3.5 text-cyan-400" />
            <span class="hidden sm:inline">Add Code</span>
          </button>

          <!-- Emoji Picker Button -->
          <button
            type="button"
            id="input-emoji-btn"
            onclick={() => (isInputEmojiPickerOpen = !isInputEmojiPickerOpen)}
            class="flex items-center gap-1 px-2 sm:px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer flex-shrink-0 {isInputEmojiPickerOpen
              ? 'bg-cyan-950 text-cyan-300 border border-cyan-800 shadow-xs'
              : 'text-slate-300 hover:text-cyan-300 hover:bg-slate-800 border border-slate-750'}"
            title="Insert Emoji (😄)"
          >
            <Smile class="w-3.5 h-3.5 text-cyan-400" />
            <span class="hidden sm:inline">Emoji</span>
          </button>

          <!-- Mention Button -->
          <button
            type="button"
            onclick={triggerMentionButton}
            class="flex items-center gap-1 px-2 sm:px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-cyan-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer flex-shrink-0"
            title="Mention a peer (@)"
          >
            <AtSign class="w-3.5 h-3.5 text-cyan-400" />
            <span class="hidden sm:inline">Mention</span>
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
            class="flex items-center gap-1 px-2 sm:px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-cyan-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer flex-shrink-0"
            title="Attach a file"
          >
            <Paperclip class="w-3.5 h-3.5 text-cyan-400" />
            <span class="hidden sm:inline">File</span>
          </button>

          <!-- Attach Folder -->
          <button
            onclick={() => folderInput.click()}
            class="flex items-center gap-1 px-2 sm:px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-amber-300 hover:bg-slate-800 border border-slate-750 transition-colors cursor-pointer flex-shrink-0"
            title="Attach entire folder (compresses on stream)"
          >
            <FolderUp class="w-3.5 h-3.5 text-amber-400" />
            <span class="hidden sm:inline">Folder</span>
          </button>

          <!-- Dev-Ignore Filter Toggle -->
          <button
            onclick={() => (devIgnore = !devIgnore)}
            class="flex items-center gap-1 px-2 sm:px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all cursor-pointer flex-shrink-0 {devIgnore
              ? 'bg-emerald-950/80 text-emerald-300 border border-emerald-800/80 shadow-xs'
              : 'bg-slate-800 text-slate-400 border border-slate-700 line-through opacity-70'}"
            title="Toggle Dev-Ignore filter (strips .git, node_modules, target, vendor, dist)"
          >
            <Shield class="w-3 h-3 text-emerald-400" />
            <span class="hidden md:inline">Dev-Ignore {devIgnore ? 'ON' : 'OFF'}</span>
          </button>

          <!-- TTL Selector -->
          <div class="flex items-center gap-1 text-xs text-slate-400 font-mono flex-shrink-0">
            <Clock class="w-3 h-3 text-slate-400 hidden xs:inline" />
            <select
              bind:value={expiration}
              class="bg-slate-800 text-slate-300 text-[11px] sm:text-xs rounded-lg px-1.5 sm:px-2 py-1 border border-slate-700 focus:outline-none focus:border-cyan-500 font-mono cursor-pointer"
              title="Transfer Expiration Lifecycle"
            >
              <option value="burn">🔥 1x</option>
              <option value="1h">1h</option>
              <option value="6h">6h</option>
              <option value="24h">24h</option>
            </select>
          </div>
        </div>

        <!-- Right: Send Button -->
        <button
          onclick={handleSendText}
          disabled={!textInput.trim()}
          class="flex items-center gap-1 sm:gap-1.5 px-3 sm:px-4 py-1.5 rounded-xl text-xs font-semibold text-white bg-cyan-600 hover:bg-cyan-500 disabled:opacity-40 disabled:cursor-not-allowed shadow-md shadow-cyan-600/30 transition-all active:scale-95 cursor-pointer flex-shrink-0"
        >
          <Send class="w-3.5 h-3.5" />
          <span class="hidden xs:inline">Send</span>
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
