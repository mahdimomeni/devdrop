<script>
  import { onMount, tick } from 'svelte';
  import {
    Search,
    X,
    Clock,
    Smile,
    User,
    Cat,
    Utensils,
    Trophy,
    Rocket,
    Laptop,
    Hash,
    Sparkles
  } from 'lucide-svelte';
  import {
    EMOJI_CATEGORIES,
    getRecentEmojis,
    saveRecentEmoji,
    searchEmojis
  } from './emojiData.js';

  let {
    onSelect,
    onClose,
    title = 'Choose Emoji',
    mode = 'input', // 'input' | 'reaction'
  } = $props();

  let searchQuery = $state('');
  let activeTab = $state('all'); // 'recent' | 'all' | category id
  let recentEmojis = $state(getRecentEmojis());
  let searchInput = $state(null);
  let scrollContainer = $state(null);
  let hoveredEmoji = $state(null);

  onMount(() => {
    recentEmojis = getRecentEmojis();
    // Focus search input on mount if not mobile
    if (window.innerWidth >= 640 && searchInput) {
      searchInput.focus();
    }
  });

  let searchResults = $derived.by(() => {
    if (!searchQuery.trim()) return null;
    return searchEmojis(searchQuery);
  });

  function handleSelect(emoji) {
    saveRecentEmoji(emoji);
    recentEmojis = getRecentEmojis();
    if (onSelect) {
      onSelect(emoji);
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose?.();
    }
  }

  function scrollToCategory(catId) {
    activeTab = catId;
    searchQuery = '';
    tick().then(() => {
      if (!scrollContainer) return;
      if (catId === 'recent') {
        scrollContainer.scrollTop = 0;
        return;
      }
      const section = scrollContainer.querySelector(`#cat-sec-${catId}`);
      if (section) {
        section.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    });
  }

  const categoryIcons = {
    smileys: Smile,
    people: User,
    nature: Cat,
    food: Utensils,
    activity: Trophy,
    travel: Rocket,
    objects: Laptop,
    symbols: Hash,
  };
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  class="flex flex-col bg-slate-900/95 backdrop-blur-xl border border-slate-750 rounded-2xl shadow-2xl overflow-hidden w-80 sm:w-88 max-w-[92vw] h-96 sm:h-[410px] select-none text-slate-200 animate-in fade-in zoom-in-95 duration-150 z-50 ring-1 ring-white/10"
  role="dialog"
  aria-label={title}
>
  <!-- Header Bar -->
  <div class="px-3 pt-3 pb-2 flex items-center justify-between border-b border-slate-800/80 gap-2 flex-shrink-0 bg-slate-950/40">
    <div class="flex items-center gap-1.5 min-w-0">
      <span class="text-sm font-semibold text-slate-200 truncate">{title}</span>
      {#if mode === 'reaction'}
        <span class="text-[10px] font-mono px-1.5 py-0.2 rounded-full bg-cyan-950 text-cyan-400 border border-cyan-800/80">
          Reaction
        </span>
      {/if}
    </div>
    {#if onClose}
      <button
        type="button"
        onclick={onClose}
        class="p-1 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors cursor-pointer"
        aria-label="Close emoji picker"
      >
        <X class="w-4 h-4" />
      </button>
    {/if}
  </div>

  <!-- Search Input -->
  <div class="p-2 border-b border-slate-800/70 flex-shrink-0 bg-slate-950/20">
    <div class="relative flex items-center">
      <Search class="w-3.5 h-3.5 text-slate-400 absolute left-2.5 pointer-events-none" />
      <input
        type="text"
        bind:this={searchInput}
        bind:value={searchQuery}
        placeholder="Search emojis (e.g. fire, laugh, rocket)..."
        class="w-full bg-slate-950/80 border border-slate-750 focus:border-cyan-500 rounded-xl pl-8 pr-7 py-1.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none transition-colors"
      />
      {#if searchQuery}
        <button
          type="button"
          onclick={() => {
            searchQuery = '';
            searchInput?.focus();
          }}
          class="absolute right-2 p-0.5 text-slate-400 hover:text-white rounded transition-colors"
          title="Clear search"
        >
          <X class="w-3 h-3" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Category Navigation Bar (when not searching) -->
  {#if !searchResults}
    <div class="flex items-center px-2 py-1 border-b border-slate-800/60 overflow-x-auto no-scrollbar gap-0.5 bg-slate-950/40 flex-shrink-0">
      {#if recentEmojis.length > 0}
        <button
          type="button"
          onclick={() => scrollToCategory('recent')}
          class="p-1.5 rounded-lg text-xs transition-colors cursor-pointer {activeTab === 'recent'
            ? 'bg-cyan-950 text-cyan-300 border border-cyan-800/80'
            : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'}"
          title="Recent / Popular"
        >
          <Clock class="w-3.5 h-3.5" />
        </button>
      {/if}

      {#each EMOJI_CATEGORIES as cat}
        {@const IconComponent = categoryIcons[cat.id] || Smile}
        <button
          type="button"
          onclick={() => scrollToCategory(cat.id)}
          class="p-1.5 rounded-lg text-xs transition-colors cursor-pointer {activeTab === cat.id
            ? 'bg-cyan-950 text-cyan-300 border border-cyan-800/80'
            : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'}"
          title={cat.name}
        >
          <IconComponent class="w-3.5 h-3.5" />
        </button>
      {/each}
    </div>
  {/if}

  <!-- Emoji Scroll Area -->
  <div
    bind:this={scrollContainer}
    class="flex-1 overflow-y-auto min-h-0 p-2 sm:p-2.5 space-y-3"
  >
    <!-- Search Results View -->
    {#if searchResults}
      {#if searchResults.length === 0}
        <div class="h-40 flex flex-col items-center justify-center text-center text-slate-500 p-4">
          <Smile class="w-8 h-8 text-slate-600 mb-2" />
          <span class="text-xs font-mono text-slate-400">No emojis found</span>
          <span class="text-[11px] text-slate-500 mt-0.5">Try searching for something else</span>
        </div>
      {:else}
        <div>
          <div class="text-[11px] font-mono font-semibold text-slate-400 px-1 mb-1.5 flex items-center justify-between">
            <span>Matching emojis ({searchResults.length})</span>
          </div>
          <div class="grid grid-cols-7 sm:grid-cols-8 gap-1">
            {#each searchResults as item}
              <button
                type="button"
                onclick={() => handleSelect(item.emoji)}
                onmouseenter={() => (hoveredEmoji = item)}
                onmouseleave={() => (hoveredEmoji = null)}
                class="w-9 h-9 flex items-center justify-center text-xl sm:text-2xl rounded-xl hover:bg-slate-800/90 active:scale-95 transition-all duration-100 cursor-pointer hover:scale-120 hover:z-10"
                title={item.name}
              >
                <span>{item.emoji}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}

    {:else}
      <!-- Standard Categorized View -->
      
      <!-- Recent / Quick Section -->
      {#if recentEmojis.length > 0}
        <div id="cat-sec-recent">
          <div class="text-[11px] font-mono font-semibold text-slate-400 px-1 mb-1 flex items-center gap-1.5">
            <Clock class="w-3 h-3 text-cyan-400" />
            <span>Recent & Popular</span>
          </div>
          <div class="grid grid-cols-7 sm:grid-cols-8 gap-1">
            {#each recentEmojis as em}
              <button
                type="button"
                onclick={() => handleSelect(em)}
                class="w-9 h-9 flex items-center justify-center text-xl sm:text-2xl rounded-xl hover:bg-slate-800/90 active:scale-95 transition-all duration-100 cursor-pointer hover:scale-125 hover:z-10"
                title={em}
              >
                <span>{em}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}

      <!-- All Category Sections -->
      {#each EMOJI_CATEGORIES as cat}
        <div id={`cat-sec-${cat.id}`} class="pt-1">
          <div class="text-[11px] font-mono font-semibold text-slate-400 px-1 mb-1 flex items-center gap-1.5">
            <span class="text-sm">{cat.icon}</span>
            <span>{cat.name}</span>
          </div>
          <div class="grid grid-cols-7 sm:grid-cols-8 gap-1">
            {#each cat.emojis as item}
              <button
                type="button"
                onclick={() => handleSelect(item.emoji)}
                onmouseenter={() => (hoveredEmoji = item)}
                onmouseleave={() => (hoveredEmoji = null)}
                class="w-9 h-9 flex items-center justify-center text-xl sm:text-2xl rounded-xl hover:bg-slate-800/90 active:scale-95 transition-all duration-100 cursor-pointer hover:scale-125 hover:z-10"
                title={item.name}
              >
                <span>{item.emoji}</span>
              </button>
            {/each}
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Bottom Preview Tooltip Footer -->
  <div class="px-3 py-1.5 border-t border-slate-800/80 bg-slate-950/60 flex items-center justify-between text-[11px] font-mono text-slate-400 flex-shrink-0 min-h-[28px]">
    {#if hoveredEmoji}
      <div class="flex items-center gap-2 truncate">
        <span class="text-base">{hoveredEmoji.emoji}</span>
        <span class="truncate capitalize text-slate-300">{hoveredEmoji.name}</span>
      </div>
    {:else}
      <span class="text-slate-500">Click to select emoji</span>
      <span class="text-[10px] text-slate-600">Esc to close</span>
    {/if}
  </div>
</div>
