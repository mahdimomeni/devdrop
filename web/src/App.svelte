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
  import NotificationModal from './lib/NotificationModal.svelte';
  import {
    loadNotificationSettings,
    saveNotificationSettings,
    playNotificationChime,
    showDesktopNotification,
    shouldNotify,
    updateDocumentTitle
  } from './lib/notificationService.js';
  import {
    Wifi,
    WifiOff,
    AlertCircle,
    CheckCircle2,
    Loader2,
    AtSign,
    BellRing,
    MessageSquare,
    X
  } from 'lucide-svelte';
  import { isUserMentioned } from './lib/mentionUtils.js';

  let userId = getUserId();
  let currentUser = $state(null);
  let peers = $state([]);
  let selectedPeerId = $state('broadcast');
  let messages = $state([]);
  let hasMore = $state(false);
  let isLoadingMessages = $state(false);
  let isLoadingOlder = $state(false);
  let unreadCounts = $state({});
  let isWsConnected = $state(false);
  let mobileActiveView = $state('chat'); // 'peers' | 'chat'
  let isAppFocused = $state(typeof document !== 'undefined' ? document.hasFocus() : true);
  let notificationSettings = $state(loadNotificationSettings());
  let isNotificationModalOpen = $state(false);

  let totalUnreads = $derived.by(() => {
    return Object.entries(unreadCounts).reduce((acc, [key, count]) => {
      if (key === selectedPeerId) return acc;
      return acc + (count || 0);
    }, 0);
  });

  // Upload Progress State
  let uploadState = $state({
    isUploading: false,
    progress: 0,
    fileName: '',
  });

  // Toast notification
  let toast = $state(null); // { message, type: 'success' | 'error' | 'mention' | 'notification', title, subtitle, senderName, targetPeerId, onAction }

  let socket = null;

  function showToast(message, type = 'success', extra = {}) {
    toast = { message, type, ...extra };
    const duration = type === 'notification' || type === 'mention' ? 5500 : 3500;
    setTimeout(() => {
      if (toast?.message === message) toast = null;
    }, duration);
  }

  function handleUpdateNotificationSettings(newSettings) {
    notificationSettings = newSettings;
    saveNotificationSettings(newSettings);
  }

  function handleOpenNotificationSettings() {
    isNotificationModalOpen = true;
  }

  // Find currently selected peer object
  let selectedPeer = $derived.by(() => {
    if (selectedPeerId === 'broadcast') return null;
    return peers.find((p) => p.id === selectedPeerId) || null;
  });

  async function loadConversation(peerId) {
    selectedPeerId = peerId;
    mobileActiveView = 'chat';
    // Clear unreads
    unreadCounts = { ...unreadCounts, [peerId]: 0 };
    isLoadingMessages = true;
    hasMore = false;
    messages = [];

    try {
      const res = await getMessages(userId, peerId, 40);
      messages = res.messages;
      hasMore = res.hasMore;
    } catch (err) {
      console.error('Failed to load messages:', err);
    } finally {
      isLoadingMessages = false;
    }
  }

  async function handleLoadOlder() {
    if (isLoadingOlder || !hasMore || messages.length === 0) return false;
    const oldest = messages[0];
    if (!oldest) return false;

    isLoadingOlder = true;
    try {
      const res = await getMessages(userId, selectedPeerId, 40, oldest.id);
      if (res.messages && res.messages.length > 0) {
        const existingIds = new Set(messages.map((m) => m.id));
        const newOldMessages = res.messages.filter((m) => !existingIds.has(m.id));
        if (newOldMessages.length > 0) {
          messages = [...newOldMessages, ...messages];
          hasMore = res.hasMore;
          return true;
        } else {
          hasMore = false;
          return false;
        }
      } else {
        hasMore = false;
        return false;
      }
    } catch (err) {
      console.error('Failed to load older messages:', err);
      return false;
    } finally {
      isLoadingOlder = false;
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

          // Evaluate notification criteria
          const notif = shouldNotify(newMsg, {
            userId,
            currentUser,
            peers,
            selectedPeerId,
            isAppFocused,
            settings: notificationSettings,
          });

          if (notif.shouldNotify) {
            // 1. Synthesize audio chime
            if (notif.shouldPlaySound) {
              playNotificationChime(notif.details.isMention ? 'mention' : 'default');
            }

            // 2. Desktop OS notification banner
            if (notif.shouldShowDesktop) {
              showDesktopNotification({
                title: notif.details.title,
                body: notif.details.body,
                tag: `msg-${newMsg.id}`,
                onClick: () => {
                  loadConversation(notif.targetPeerId);
                  mobileActiveView = 'chat';
                },
              });
            }

            // 3. In-App Toast Alert
            if (notif.shouldShowInAppToast) {
              showToast(notif.details.body, notif.details.isMention ? 'mention' : 'notification', {
                title: notif.details.title,
                subtitle: notif.details.subtitle,
                senderName: notif.details.senderName,
                targetPeerId: notif.targetPeerId,
                onAction: () => {
                  loadConversation(notif.targetPeerId);
                  mobileActiveView = 'chat';
                },
              });
            }

            // 4. Update tab title
            updateDocumentTitle(totalUnreads, notif.details.senderName);
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

  $effect(() => {
    updateDocumentTitle(totalUnreads);
  });

  let cleanupFocusListeners = null;

  onMount(async () => {
    const handleFocus = () => {
      isAppFocused = true;
      updateDocumentTitle(totalUnreads);
    };
    const handleBlur = () => {
      isAppFocused = false;
    };
    const handleVisibility = () => {
      isAppFocused = !document.hidden && document.hasFocus();
      if (isAppFocused) {
        updateDocumentTitle(totalUnreads);
      }
    };

    window.addEventListener('focus', handleFocus);
    window.addEventListener('blur', handleBlur);
    document.addEventListener('visibilitychange', handleVisibility);

    cleanupFocusListeners = () => {
      window.removeEventListener('focus', handleFocus);
      window.removeEventListener('blur', handleBlur);
      document.removeEventListener('visibilitychange', handleVisibility);
    };

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
    if (cleanupFocusListeners) cleanupFocusListeners();
    if (socket) socket.destroy();
  });
</script>

<main class="h-full w-full flex flex-col bg-slate-950 text-slate-100 overflow-hidden font-sans min-h-0">
  
  <!-- Disconnected Network Alert Banner -->
  {#if !isWsConnected}
    <div class="bg-amber-600/90 text-slate-950 px-3 sm:px-4 py-1.5 text-xs font-mono font-medium flex items-center justify-center gap-2 z-50 truncate flex-shrink-0">
      <WifiOff class="w-3.5 h-3.5 flex-shrink-0" />
      <span class="truncate">Reconnecting to DevDrop LAN node...</span>
    </div>
  {/if}

  <!-- Active Upload Progress Floating Bar -->
  {#if uploadState.isUploading}
    <div class="fixed top-3 sm:top-4 right-3 sm:right-4 left-3 sm:left-auto z-50 bg-slate-900 border border-cyan-500/50 rounded-xl p-3 sm:p-3.5 shadow-2xl flex flex-col gap-2 min-w-0 sm:min-w-[280px] animate-in fade-in slide-in-from-top-4 duration-200">
      <div class="flex items-center justify-between text-xs font-mono">
        <span class="flex items-center gap-1.5 text-cyan-400 font-semibold truncate max-w-[180px]">
          <Loader2 class="w-3.5 h-3.5 animate-spin flex-shrink-0" />
          <span class="truncate">{uploadState.fileName}</span>
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
    {#if toast.type === 'notification' || (toast.type === 'mention' && toast.targetPeerId)}
      <div
        class="fixed bottom-20 sm:bottom-6 right-4 sm:right-6 left-4 sm:left-auto z-50 flex items-start gap-3 p-3 sm:p-3.5 rounded-xl border shadow-2xl text-xs font-mono animate-in fade-in slide-in-from-bottom-3 duration-200 bg-slate-900/95 backdrop-blur-md {toast.type === 'mention' ? 'border-amber-500/80 ring-1 ring-amber-500/30 shadow-amber-500/10' : 'border-cyan-500/60 ring-1 ring-cyan-500/20 shadow-cyan-500/10'} text-slate-100 max-w-sm w-full mx-auto sm:mx-0"
      >
        <div class="p-2 rounded-lg {toast.type === 'mention' ? 'bg-amber-500/20 text-amber-400' : 'bg-cyan-500/20 text-cyan-400'} flex-shrink-0 mt-0.5">
          {#if toast.type === 'mention'}
            <AtSign class="w-4 h-4 animate-bounce" />
          {:else}
            <BellRing class="w-4 h-4 animate-pulse" />
          {/if}
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-center justify-between gap-1 mb-0.5">
            <span class="font-bold text-slate-100 truncate text-[11px] sm:text-xs">
              {toast.title || toast.senderName || 'New Message'}
            </span>
            {#if toast.subtitle}
              <span class="text-[9px] px-1.5 py-0.2 rounded font-semibold {toast.type === 'mention' ? 'bg-amber-950 text-amber-300 border border-amber-800/60' : 'bg-cyan-950 text-cyan-300 border border-cyan-800/60'}">
                {toast.subtitle}
              </span>
            {/if}
          </div>
          <p class="text-[11px] text-slate-300 line-clamp-2 leading-tight">
            {toast.message}
          </p>
          {#if toast.onAction}
            <div class="mt-2 flex items-center gap-2">
              <button
                type="button"
                onclick={() => {
                  toast.onAction();
                  toast = null;
                }}
                class="px-2.5 py-1 bg-cyan-600 hover:bg-cyan-500 text-white rounded text-[10px] font-bold transition-colors cursor-pointer"
              >
                Open Chat
              </button>
              <button
                type="button"
                onclick={() => (toast = null)}
                class="px-2 py-1 text-slate-400 hover:text-white text-[10px] transition-colors cursor-pointer"
              >
                Dismiss
              </button>
            </div>
          {/if}
        </div>
        <button
          type="button"
          onclick={() => (toast = null)}
          class="text-slate-400 hover:text-white p-0.5 rounded transition-colors cursor-pointer flex-shrink-0"
          aria-label="Dismiss alert"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      </div>
    {:else}
      <!-- Standard small alert toast -->
      <div class="fixed bottom-20 sm:bottom-6 right-4 sm:right-6 left-4 sm:left-auto z-50 flex items-center gap-2.5 px-3.5 sm:px-4 py-2 sm:py-2.5 rounded-xl border shadow-2xl text-xs font-mono animate-in fade-in slide-in-from-bottom-2 duration-150 {toast.type === 'error' ? 'bg-rose-950 border-rose-800 text-rose-200' : toast.type === 'mention' ? 'bg-amber-950/95 border-amber-500/70 text-amber-200 ring-1 ring-amber-500/30 shadow-amber-500/10' : 'bg-slate-900 border-cyan-500/50 text-slate-100'} max-w-sm mx-auto sm:mx-0">
        {#if toast.type === 'error'}
          <AlertCircle class="w-4 h-4 text-rose-400 flex-shrink-0" />
        {:else if toast.type === 'mention'}
          <AtSign class="w-4 h-4 text-amber-400 flex-shrink-0 animate-bounce" />
        {:else}
          <CheckCircle2 class="w-4 h-4 text-emerald-400 flex-shrink-0" />
        {/if}
        <span class="truncate">{toast.message}</span>
      </div>
    {/if}
  {/if}

  <!-- Notification Settings Modal -->
  <NotificationModal
    isOpen={isNotificationModalOpen}
    settings={notificationSettings}
    onUpdateSettings={handleUpdateNotificationSettings}
    onClose={() => (isNotificationModalOpen = false)}
    onTestNotification={() => {
      showToast('This is a test notification from DevDrop!', 'notification', {
        title: 'DevDrop Notification Test',
        subtitle: 'Test Banner',
        senderName: 'DevDrop Node',
        targetPeerId: selectedPeerId,
        onAction: () => {
          loadConversation(selectedPeerId);
          mobileActiveView = 'chat';
        },
      });
    }}
  />

  <!-- Main App Layout -->
  <div class="flex-1 flex overflow-hidden relative min-h-0 w-full">
    <!-- Left Sidebar: Peer List -->
    <div class="{mobileActiveView === 'peers' ? 'flex' : 'hidden'} md:flex w-full md:w-80 h-full flex-shrink-0 min-h-0">
      <PeerList
        {currentUser}
        {peers}
        {selectedPeerId}
        {unreadCounts}
        {notificationSettings}
        onOpenNotificationSettings={handleOpenNotificationSettings}
        onSelectPeer={(id) => {
          loadConversation(id);
          mobileActiveView = 'chat';
        }}
        onUpdateName={handleUpdateName}
        onCloseMobile={() => (mobileActiveView = 'chat')}
      />
    </div>

    <!-- Main Content Area: Chat View -->
    <div class="{mobileActiveView === 'chat' ? 'flex' : 'hidden'} md:flex flex-1 h-full min-w-0 min-h-0 w-full">
      <ChatView
        {currentUser}
        {selectedPeer}
        {peers}
        {messages}
        {hasMore}
        {isLoadingMessages}
        {isLoadingOlder}
        {notificationSettings}
        onOpenNotificationSettings={handleOpenNotificationSettings}
        onLoadOlder={handleLoadOlder}
        onSendMessage={handleSendMessage}
        onSendCode={handleSendCode}
        onUploadFiles={handleUploadFiles}
        onBackToPeers={() => (mobileActiveView = 'peers')}
        totalUnreadCount={totalUnreads}
        onSelectPeer={(id) => {
          loadConversation(id);
          mobileActiveView = 'chat';
        }}
      />
    </div>
  </div>
</main>
