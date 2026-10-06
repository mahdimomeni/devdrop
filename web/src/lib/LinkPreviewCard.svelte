<script>
  import { onMount } from 'svelte';
  import { ExternalLink, Globe, X, Image as ImageIcon } from 'lucide-svelte';
  import { fetchLinkPreview } from './api.js';

  let {
    url,
    isMe = false,
  } = $props();

  let preview = $state(null);
  let isLoading = $state(true);
  let isDismissed = $state(false);
  let imageFailed = $state(false);
  let faviconFailed = $state(false);

  onMount(() => {
    let active = true;

    if (!url) {
      isLoading = false;
      return;
    }

    fetchLinkPreview(url).then((data) => {
      if (active) {
        preview = data;
        isLoading = false;
      }
    }).catch(() => {
      if (active) {
        isLoading = false;
      }
    });

    return () => {
      active = false;
    };
  });

  function handleDismiss(e) {
    e.stopPropagation();
    e.preventDefault();
    isDismissed = true;
  }
</script>

{#if !isDismissed}
  {#if isLoading}
    <!-- Loading Shimmer Skeleton -->
    <div
      class="mt-2 w-full rounded-xl p-2.5 sm:p-3 border text-left transition-all select-none animate-pulse {isMe
        ? 'bg-cyan-950/40 border-cyan-400/20 border-l-[3.5px] border-l-cyan-300'
        : 'bg-slate-900/80 border-slate-750/80 border-l-[3.5px] border-l-cyan-400'}"
    >
      <div class="flex items-center gap-2 mb-2">
        <div class="w-3.5 h-3.5 rounded-full bg-slate-700/60"></div>
        <div class="h-2.5 w-24 rounded bg-slate-700/60"></div>
      </div>
      <div class="h-3.5 w-4/5 rounded bg-slate-700/70 mb-2"></div>
      <div class="space-y-1">
        <div class="h-2.5 w-full rounded bg-slate-700/50"></div>
        <div class="h-2.5 w-2/3 rounded bg-slate-700/50"></div>
      </div>
    </div>
  {:else if preview && (preview.title || preview.description || preview.image)}
    <!-- Rich Link Preview Card -->
    <a
      href={preview.url || url}
      target="_blank"
      rel="noopener noreferrer"
      class="group/linkcard mt-2 block w-full rounded-xl p-2.5 sm:p-3 border text-left transition-all duration-200 cursor-pointer shadow-md select-text hover:shadow-lg {isMe
        ? 'bg-cyan-950/50 hover:bg-cyan-950/70 border-cyan-400/30 border-l-[3.5px] border-l-white/90 text-white'
        : 'bg-slate-900/85 hover:bg-slate-900 border-slate-750 border-l-[3.5px] border-l-cyan-400 text-slate-100'}"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Header: Site Name + Favicon + External Link Icon + Dismiss Button -->
      <div class="flex items-center justify-between gap-1.5 mb-1 text-[11px] font-mono">
        <div class="flex items-center gap-1.5 min-w-0">
          {#if preview.favicon && !faviconFailed}
            <img
              src={preview.favicon}
              alt=""
              class="w-3.5 h-3.5 rounded-xs object-contain flex-shrink-0"
              onerror={() => { faviconFailed = true; }}
            />
          {:else}
            <Globe class="w-3.5 h-3.5 {isMe ? 'text-cyan-300' : 'text-cyan-400'} flex-shrink-0" />
          {/if}

          <span class="font-bold uppercase tracking-wider truncate {isMe ? 'text-cyan-200' : 'text-cyan-400'}">
            {preview.site_name || 'Link'}
          </span>

          <ExternalLink class="w-3 h-3 text-slate-400 opacity-60 group-hover/linkcard:opacity-100 group-hover/linkcard:translate-x-0.5 group-hover/linkcard:-translate-y-0.5 transition-all flex-shrink-0" />
        </div>

        <!-- Optional Dismiss Preview button -->
        <button
          type="button"
          onclick={handleDismiss}
          class="opacity-0 group-hover/linkcard:opacity-75 hover:!opacity-100 p-0.5 rounded text-slate-400 hover:text-white transition-opacity flex-shrink-0 cursor-pointer"
          title="Dismiss preview"
          aria-label="Dismiss link preview"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      </div>

      <!-- Preview Title -->
      {#if preview.title}
        <h4 class="text-xs sm:text-sm font-bold text-white group-hover/linkcard:text-cyan-300 transition-colors line-clamp-2 leading-snug font-sans">
          {preview.title}
        </h4>
      {/if}

      <!-- Preview Description -->
      {#if preview.description}
        <p class="text-[11px] sm:text-xs line-clamp-3 leading-relaxed mt-1 font-sans {isMe ? 'text-cyan-100/90' : 'text-slate-300/90'}">
          {preview.description}
        </p>
      {/if}

      <!-- Preview Media Image Banner -->
      {#if preview.image && !imageFailed}
        <div class="mt-2.5 rounded-lg overflow-hidden border border-slate-700/60 bg-slate-950/60 max-h-48 sm:max-h-56 relative w-full flex items-center justify-center">
          <img
            src={preview.image}
            alt={preview.title || 'Link preview image'}
            class="w-full h-auto max-h-48 sm:max-h-56 object-cover transition-transform duration-300 group-hover/linkcard:scale-[1.02]"
            loading="lazy"
            onerror={() => { imageFailed = true; }}
          />
        </div>
      {/if}
    </a>
  {/if}
{/if}
