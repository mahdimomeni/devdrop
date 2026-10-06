<script>
  import {
    Users,
    Radio,
    Search,
    Edit3,
    Check,
    X,
    Wifi,
    Laptop,
    Sparkles,
    Circle,
    UserCheck,
    MessageSquare,
    Bell,
    BellRing,
    BellOff
  } from 'lucide-svelte';
  import { formatRelativeTime } from './api.js';

  let {
    currentUser = null,
    peers = [],
    selectedPeerId = 'broadcast',
    unreadCounts = {},
    notificationSettings = null,
    onOpenNotificationSettings = null,
    onSelectPeer,
    onUpdateName,
    onCloseMobile = null,
  } = $props();

  let searchQuery = $state('');
  let isEditingName = $state(false);
  let editNameValue = $state('');

  function startEditing() {
    editNameValue = currentUser?.display_name || '';
    isEditingName = true;
  }

  function saveEditing() {
    if (editNameValue.trim()) {
      onUpdateName(editNameValue.trim());
    }
    isEditingName = false;
  }

  function cancelEditing() {
    isEditingName = false;
  }

  // Filter peers excluding current user
  let filteredPeers = $derived.by(() => {
    const q = searchQuery.toLowerCase().trim();
    return peers
      .filter((p) => p.id !== currentUser?.id)
      .filter((p) => {
        if (!q) return true;
        return (
          p.display_name?.toLowerCase().includes(q) ||
          p.ip_address?.toLowerCase().includes(q)
        );
      })
      .sort((a, b) => {
        // Online peers first
        if (a.is_online !== b.is_online) return a.is_online ? -1 : 1;
        // Then by last seen
        return new Date(b.last_seen_at) - new Date(a.last_seen_at);
      });
  });

  let onlineCount = $derived.by(() => {
    return peers.filter((p) => p.is_online).length;
  });
</script>

<aside class="w-full md:w-80 h-full flex flex-col bg-slate-950 border-r border-slate-800/80 select-none flex-shrink-0 min-h-0">
  <!-- My Profile Section -->
  <div class="p-3.5 sm:p-4 border-b border-slate-800/90 bg-slate-900/60 pt-[max(0.875rem,env(safe-area-inset-top))] flex-shrink-0">
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <div class="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse shadow-sm shadow-emerald-400/50"></div>
        <span class="text-[11px] font-mono font-medium uppercase tracking-wider text-slate-400">
          DevDrop LAN Node
        </span>
      </div>
      <div class="flex items-center gap-1.5 sm:gap-2">
        <span class="text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 border border-emerald-800/50">
          {onlineCount} {onlineCount === 1 ? 'Peer' : 'Peers'} Online
        </span>
        {#if onOpenNotificationSettings}
          <button
            type="button"
            onclick={onOpenNotificationSettings}
            class="p-1 text-slate-400 hover:text-cyan-300 hover:bg-slate-800 rounded-md border border-slate-800/80 hover:border-cyan-800/60 transition-colors cursor-pointer"
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
          </button>
        {/if}
        {#if onCloseMobile}
          <button
            type="button"
            onclick={onCloseMobile}
            class="md:hidden flex items-center gap-1.5 px-2.5 py-1 bg-cyan-950 hover:bg-cyan-900 text-cyan-300 border border-cyan-800/60 rounded-lg text-xs font-mono font-medium transition-colors cursor-pointer"
            title="Return to conversation"
          >
            <MessageSquare class="w-3.5 h-3.5" />
            <span>Chat</span>
          </button>
        {/if}
      </div>
    </div>

    <!-- My Avatar & Name -->
    <div class="flex items-center gap-3">
      <div class="relative w-11 h-11 rounded-xl bg-gradient-to-br from-cyan-500 to-indigo-600 flex items-center justify-center font-bold text-white text-base shadow-md shadow-cyan-500/20 flex-shrink-0">
        {(currentUser?.display_name || 'D')[0]?.toUpperCase()}
        <div class="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-emerald-500 border-2 border-slate-900"></div>
      </div>

      <div class="min-w-0 flex-1">
        {#if isEditingName}
          <div class="flex items-center gap-1.5">
            <input
              type="text"
              bind:value={editNameValue}
              onkeydown={(e) => {
                if (e.key === 'Enter') saveEditing();
                if (e.key === 'Escape') cancelEditing();
              }}
              class="w-full bg-slate-950 text-white text-xs px-2 py-1 rounded border border-cyan-500 focus:outline-none font-mono"
            />
            <button
              onclick={saveEditing}
              class="p-1 hover:bg-emerald-500/20 text-emerald-400 rounded transition-colors"
              title="Save Name"
            >
              <Check class="w-4 h-4" />
            </button>
            <button
              onclick={cancelEditing}
              class="p-1 hover:bg-rose-500/20 text-rose-400 rounded transition-colors"
              title="Cancel"
            >
              <X class="w-4 h-4" />
            </button>
          </div>
        {:else}
          <button
            type="button"
            class="flex items-center gap-1.5 group cursor-pointer text-left bg-transparent border-0 p-0"
            onclick={startEditing}
          >
            <span class="text-sm font-semibold text-slate-100 truncate font-mono group-hover:text-cyan-400 transition-colors">
              {currentUser?.display_name || 'Connecting...'}
            </span>
            <Edit3 class="w-3.5 h-3.5 text-slate-500 group-hover:text-cyan-400 transition-colors opacity-70 group-hover:opacity-100 flex-shrink-0" />
          </button>
          <div class="flex items-center gap-1.5 text-[11px] text-slate-400 font-mono mt-0.5">
            <Wifi class="w-3 h-3 text-cyan-400" />
            <span class="truncate">{currentUser?.ip_address || '127.0.0.1'}</span>
          </div>
        {/if}
      </div>
    </div>
  </div>

  <!-- Search Filter -->
  <div class="p-3 border-b border-slate-800/60 bg-slate-950 flex-shrink-0">
    <div class="relative">
      <Search class="w-3.5 h-3.5 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Search peers by name or IP..."
        class="w-full bg-slate-900/90 text-slate-200 text-xs pl-8 pr-3 py-1.5 rounded-lg border border-slate-800 placeholder-slate-500 focus:outline-none focus:border-cyan-500/60 font-sans transition-colors"
      />
      {#if searchQuery}
        <button
          onclick={() => (searchQuery = '')}
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
        >
          <X class="w-3 h-3" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Broadcast & Peer Channel List -->
  <div class="flex-1 overflow-y-auto min-h-0 p-2 space-y-1">
    <!-- Broadcast Channel -->
    <button
      onclick={() => onSelectPeer('broadcast')}
      class="w-full flex items-center justify-between p-2.5 rounded-xl text-left transition-all duration-150 {selectedPeerId === 'broadcast'
        ? 'bg-cyan-500/15 border border-cyan-500/30 text-white shadow-sm shadow-cyan-500/10'
        : 'text-slate-300 hover:bg-slate-900 border border-transparent'}"
    >
      <div class="flex items-center gap-3 min-w-0">
        <div class="w-9 h-9 rounded-lg bg-gradient-to-tr from-amber-500/20 to-cyan-500/20 border border-cyan-500/30 flex items-center justify-center text-cyan-400 flex-shrink-0">
          <Radio class="w-4 h-4 animate-pulse" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-1.5">
            <span class="text-xs font-semibold font-mono tracking-tight">LAN Broadcast</span>
            <span class="text-[9px] uppercase px-1.5 py-0.2 rounded bg-cyan-950 text-cyan-400 font-bold border border-cyan-800/40">
              All Peers
            </span>
          </div>
          <p class="text-[11px] text-slate-400 truncate mt-0.5">
            Public feed across local network
          </p>
        </div>
      </div>

      {#if unreadCounts['broadcast'] > 0}
        <span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-cyan-500 text-slate-950 font-mono shadow-sm">
          {unreadCounts['broadcast']}
        </span>
      {/if}
    </button>

    <!-- Peers Section Label -->
    <div class="px-3 pt-3 pb-1 text-[10px] font-mono uppercase tracking-wider text-slate-500 flex items-center justify-between">
      <span>Direct LAN Peers ({filteredPeers.length})</span>
    </div>

    <!-- Peer Items -->
    {#if filteredPeers.length === 0}
      <div class="p-6 text-center text-xs text-slate-500 font-mono">
        {#if searchQuery}
          No peers match "{searchQuery}"
        {:else}
          Waiting for LAN peers to connect...
        {/if}
      </div>
    {:else}
      {#each filteredPeers as peer (peer.id)}
        <button
          onclick={() => onSelectPeer(peer.id)}
          class="w-full flex items-center justify-between p-2.5 rounded-xl text-left transition-all duration-150 {selectedPeerId === peer.id
            ? 'bg-slate-800/90 border border-slate-700 text-white shadow-sm'
            : 'text-slate-300 hover:bg-slate-900 border border-transparent'}"
        >
          <div class="flex items-center gap-3 min-w-0">
            <!-- Peer Avatar with status badge -->
            <div class="relative w-9 h-9 rounded-lg bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-300 text-xs flex-shrink-0">
              {(peer.display_name || 'U')[0]?.toUpperCase()}
              {#if peer.is_online}
                <div class="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-400 border border-slate-900 animate-pulse shadow-sm shadow-emerald-400"></div>
              {:else}
                <div class="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-slate-600 border border-slate-900"></div>
              {/if}
            </div>

            <!-- Details -->
            <div class="min-w-0">
              <div class="flex items-center gap-1.5">
                <span class="text-xs font-medium font-mono truncate text-slate-200">
                  {peer.display_name}
                </span>
              </div>
              <div class="flex items-center gap-2 text-[10px] text-slate-500 font-mono mt-0.5">
                <span>{peer.ip_address}</span>
                <span>•</span>
                {#if peer.is_online}
                  <span class="text-emerald-400">online</span>
                {:else}
                  <span>{formatRelativeTime(peer.last_seen_at)}</span>
                {/if}
              </div>
            </div>
          </div>

          <!-- Unread Badge -->
          {#if unreadCounts[peer.id] > 0}
            <span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-cyan-500 text-slate-950 font-mono shadow-sm">
              {unreadCounts[peer.id]}
            </span>
          {/if}
        </button>
      {/each}
    {/if}
  </div>

  <!-- LAN Footer Status -->
  <div class="p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] bg-slate-950 border-t border-slate-800/80 text-[10px] text-slate-500 font-mono flex items-center justify-between">
    <span class="flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
      <span>Zero-Config LAN</span>
    </span>
    <span class="text-slate-600">DevDrop v1.0</span>
  </div>
</aside>
