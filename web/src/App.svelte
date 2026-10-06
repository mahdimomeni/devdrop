<script>
  import { onMount, onDestroy } from 'svelte';
  import {
    getUserId,
    getProfile,
    getPeers,
    getMessages,
    sendTextMessage,
    sendCodeMessage,
    uploadTransfer,
    updateDisplayName,
    DevDropSocket
  } from './lib/api.js';
  import PeerList from './lib/PeerList.svelte';
  import ChatView from './lib/ChatView.svelte';
  import { Wifi, WifiOff, AlertCircle, CheckCircle2, Loader2 } from 'lucide-svelte';

  let userId = getUserId();
  let currentUser = $state(null);
  let peers = $state([]);
  let selectedPeerId = $state('broadcast');
  let messages = $state([]);
  let unreadCounts = $state({});
  let isWsConnected = $state(false);

  // Upload Progress State
  let uploadState = $state({
    isUploading: false,
    progress: 0,
    fileName: '',
  });

  // Toast notification
  let toast = $state(null); // { message, type: 'success' | 'error' }

  let socket = null;

  function showToast(message, type = 'success') {
    toast = { message, type };
    setTimeout(() => {
      if (toast?.message === message) toast = null;
    }, 3500);
  }

  // Find currently selected peer object
  let selectedPeer = $derived.by(() => {
    if (selectedPeerId === 'broadcast') return null;
    return peers.find((p) => p.id === selectedPeerId) || null;
  });

  async function loadConversation(peerId) {
    selectedPeerId = peerId;
    // Clear unreads
    unreadCounts = { ...unreadCounts, [peerId]: 0 };

    try {
      const msgs = await getMessages(userId, peerId);
      messages = msgs;
    } catch (err) {
      console.error('Failed to load messages:', err);
    }
  }

  async function handleSendMessage(body, replyToId = null) {
    try {
      await sendTextMessage(userId, selectedPeerId, body, replyToId);
    } catch (err) {
      showToast('Failed to send message: ' + err.message, 'error');
    }
  }

  async function handleSendCode(payload) {
    try {
      await sendCodeMessage(
        userId,
        selectedPeerId,
        payload.comment,
        payload.language,
        payload.code,
        payload.replyToId || null
      );
      showToast('Code snippet dropped to LAN');
    } catch (err) {
      showToast('Failed to send code: ' + err.message, 'error');
    }
  }

  async function handleUploadFiles(opts) {
    const { files, isFolder, folderName, expiration, devIgnore, replyToId } = opts;
    if (!files || files.length === 0) return;

    uploadState = {
      isUploading: true,
      progress: 0,
      fileName: isFolder ? `${folderName}.zip` : files[0].name,
    };

    const formData = new FormData();
    formData.append('sender_id', userId);
    formData.append('receiver_id', selectedPeerId);
    formData.append('expiration', expiration);
    formData.append('burn_on_read', expiration === 'burn' ? 'true' : 'false');
    formData.append('dev_ignore', devIgnore ? 'true' : 'false');
    formData.append('is_folder', isFolder ? 'true' : 'false');
    if (isFolder) {
      formData.append('folder_name', folderName);
    }
    if (replyToId) {
      formData.append('reply_to_id', replyToId);
    }

    // Append file parts
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      // If relativePath exists from webkitdirectory or drop
      const path = file.webkitRelativePath || file.name;
      formData.append('files', file, path);
    }

    try {
      await uploadTransfer(formData, (percent) => {
        uploadState = { ...uploadState, progress: percent };
      });
      showToast(isFolder ? `Folder "${folderName}" compressed & uploaded` : `File "${files[0].name}" uploaded`);
    } catch (err) {
      showToast('Upload failed: ' + err.message, 'error');
    } finally {
      uploadState = { isUploading: false, progress: 0, fileName: '' };
    }
  }

  async function handleUpdateName(newName) {
    try {
      const res = await updateDisplayName(userId, newName);
      if (res.user) {
        currentUser = res.user;
        showToast(`Display name updated to "${newName}"`);
      }
    } catch (err) {
      showToast('Failed to update name: ' + err.message, 'error');
    }
  }

  function handleWsEvent(event) {
    switch (event.type) {
      case 'ws_open':
        isWsConnected = true;
        break;

      case 'ws_close':
        isWsConnected = false;
        break;

      case 'init':
        if (event.payload?.me) currentUser = event.payload.me;
        if (event.payload?.peers) peers = event.payload.peers;
        break;

      case 'peers':
        if (Array.isArray(event.payload)) {
          peers = event.payload;
        }
        break;

      case 'presence':
        if (event.payload) {
          const { user_id, is_online } = event.payload;
          peers = peers.map((p) =>
            p.id === user_id ? { ...p, is_online } : p
          );
        }
        break;

      case 'user_updated':
        if (event.payload) {
          const updated = event.payload;
          if (updated.id === userId) {
            currentUser = updated;
          }
          peers = peers.map((p) => (p.id === updated.id ? updated : p));
        }
        break;

      case 'message':
        if (event.payload) {
          const newMsg = event.payload;
          // Determine if message belongs to current view
          const isCurrentView =
            (selectedPeerId === 'broadcast' && (newMsg.receiver_id === 'broadcast' || newMsg.receiver_id === 'all')) ||
            (selectedPeerId !== 'broadcast' &&
              ((newMsg.sender_id === selectedPeerId && newMsg.receiver_id === userId) ||
                (newMsg.sender_id === userId && newMsg.receiver_id === selectedPeerId)));

          if (isCurrentView) {
            // Append message if not duplicate
            if (!messages.some((m) => m.id === newMsg.id)) {
              messages = [...messages, newMsg];
            }
          } else {
            // Increment unread count for the channel
            const unreadKey =
              newMsg.receiver_id === 'broadcast' || newMsg.receiver_id === 'all'
                ? 'broadcast'
                : newMsg.sender_id;
            unreadCounts = {
              ...unreadCounts,
              [unreadKey]: (unreadCounts[unreadKey] || 0) + 1,
            };
          }
        }
        break;

      case 'file_expired':
        if (event.payload?.message_id) {
          const targetId = event.payload.message_id;
          messages = messages.map((m) => {
            if (m.id === targetId && m.transfer) {
              return {
                ...m,
                transfer: { ...m.transfer, is_expired: true },
              };
            }
            return m;
          });
        }
        break;
    }
  }

  onMount(async () => {
    // 1. Fetch initial profile
    try {
      const prof = await getProfile(userId);
      currentUser = prof;
    } catch (err) {
      console.warn('Initial profile load failed:', err);
    }

    // 2. Fetch initial peers
    try {
      const p = await getPeers();
      peers = p;
    } catch (err) {
      console.warn('Initial peers load failed:', err);
    }

    // 3. Load initial broadcast conversation
    await loadConversation('broadcast');

    // 4. Connect WebSocket
    socket = new DevDropSocket(userId, handleWsEvent);
  });

  onDestroy(() => {
    if (socket) socket.destroy();
  });
</script>

<main class="h-screen w-screen flex flex-col bg-slate-950 text-slate-100 overflow-hidden font-sans">
  
  <!-- Disconnected Network Alert Banner -->
  {#if !isWsConnected}
    <div class="bg-amber-600/90 text-slate-950 px-4 py-1.5 text-xs font-mono font-medium flex items-center justify-center gap-2 z-50">
      <WifiOff class="w-3.5 h-3.5" />
      <span>Reconnecting to DevDrop LAN node...</span>
    </div>
  {/if}

  <!-- Active Upload Progress Floating Bar -->
  {#if uploadState.isUploading}
    <div class="fixed top-4 right-4 z-50 bg-slate-900 border border-cyan-500/50 rounded-xl p-3.5 shadow-2xl flex flex-col gap-2 min-w-[280px] animate-in fade-in slide-in-from-top-4 duration-200">
      <div class="flex items-center justify-between text-xs font-mono">
        <span class="flex items-center gap-1.5 text-cyan-400 font-semibold truncate max-w-[180px]">
          <Loader2 class="w-3.5 h-3.5 animate-spin" />
          {uploadState.fileName}
        </span>
        <span class="text-slate-300 font-bold">{uploadState.progress}%</span>
      </div>
      <div class="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden">
        <div
          class="bg-gradient-to-r from-cyan-500 to-indigo-500 h-full rounded-full transition-all duration-150"
          style="width: {uploadState.progress}%"
        ></div>
      </div>
      <span class="text-[10px] text-slate-500 font-mono">Streaming 32KB buffers to LAN server</span>
    </div>
  {/if}

  <!-- Floating Toast Alert -->
  {#if toast}
    <div class="fixed bottom-6 right-6 z-50 flex items-center gap-2.5 px-4 py-2.5 rounded-xl border shadow-2xl text-xs font-mono animate-in fade-in slide-in-from-bottom-2 duration-150 {toast.type === 'error' ? 'bg-rose-950 border-rose-800 text-rose-200' : 'bg-slate-900 border-cyan-500/50 text-slate-100'}">
      {#if toast.type === 'error'}
        <AlertCircle class="w-4 h-4 text-rose-400 flex-shrink-0" />
      {:else}
        <CheckCircle2 class="w-4 h-4 text-emerald-400 flex-shrink-0" />
      {/if}
      <span>{toast.message}</span>
    </div>
  {/if}

  <!-- Main App Layout -->
  <div class="flex-1 flex overflow-hidden">
    <!-- Left Sidebar: Peer List -->
    <PeerList
      {currentUser}
      {peers}
      {selectedPeerId}
      {unreadCounts}
      onSelectPeer={loadConversation}
      onUpdateName={handleUpdateName}
    />

    <!-- Main Content Area: Chat View -->
    <ChatView
      {currentUser}
      {selectedPeer}
      {peers}
      {messages}
      onSendMessage={handleSendMessage}
      onSendCode={handleSendCode}
      onUploadFiles={handleUploadFiles}
    />
  </div>
</main>
