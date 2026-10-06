<script>
  import { onMount, onDestroy } from 'svelte';
  import { formatBytes, formatExpirationCountdown, getDownloadUrl } from './api.js';
  import { Download, File, FolderArchive, Flame, Clock, AlertTriangle, Image as ImageIcon, ExternalLink } from 'lucide-svelte';

  let { transfer, isOutgoing = false } = $props();

  let countdown = $state({ label: '', expired: false, isBurn: false });
  let intervalId;

  function updateTimer() {
    if (!transfer) return;
    countdown = formatExpirationCountdown(
      transfer.expires_at,
      transfer.burn_on_read,
      transfer.download_count
    );
  }

  onMount(() => {
    updateTimer();
    intervalId = setInterval(updateTimer, 5000);
  });

  onDestroy(() => {
    if (intervalId) clearInterval(intervalId);
  });

  let isImage = $derived.by(() => {
    if (!transfer || transfer.is_folder_zip) return false;
    const ext = transfer.file_name.split('.').pop()?.toLowerCase();
    return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'].includes(ext);
  });

  let downloadUrl = $derived(getDownloadUrl(transfer.message_id));
</script>

<div class="flex flex-col gap-2.5 p-3.5 rounded-xl border transition-all duration-200 {isOutgoing ? 'bg-slate-800/90 border-slate-700/80 text-slate-100' : 'bg-slate-900/90 border-slate-800 text-slate-100'} shadow-lg max-w-md w-full">
  
  <!-- Image Preview if applicable and not expired -->
  {#if isImage && !countdown.expired}
    <div class="relative rounded-lg overflow-hidden border border-slate-700/60 bg-black/40 max-h-60 group">
      <img
        src={downloadUrl}
        alt={transfer.file_name}
        class="w-full h-auto object-contain max-h-56 mx-auto transition-transform group-hover:scale-[1.01]"
        loading="lazy"
      />
      <div class="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent opacity-0 group-hover:opacity-100 transition-opacity flex items-end justify-between p-2">
        <span class="text-[11px] text-white/90 bg-black/60 px-2 py-0.5 rounded font-mono truncate max-w-[200px]">
          {transfer.file_name}
        </span>
        <a
          href={downloadUrl}
          target="_blank"
          rel="noreferrer"
          class="p-1.5 bg-white/20 hover:bg-white/40 text-white rounded backdrop-blur-sm transition-colors"
          title="Open Image"
        >
          <ExternalLink class="w-3.5 h-3.5" />
        </a>
      </div>
    </div>
  {/if}

  <!-- Header Info -->
  <div class="flex items-start justify-between gap-3">
    <div class="flex items-center gap-3 min-w-0">
      <div class="p-2.5 rounded-xl {transfer.is_folder_zip ? 'bg-amber-500/15 text-amber-400 border border-amber-500/20' : 'bg-cyan-500/15 text-cyan-400 border border-cyan-500/20'} flex-shrink-0">
        {#if transfer.is_folder_zip}
          <FolderArchive class="w-6 h-6" />
        {:else if isImage}
          <ImageIcon class="w-6 h-6" />
        {:else}
          <File class="w-6 h-6" />
        {/if}
      </div>

      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="text-sm font-semibold text-slate-100 truncate block font-mono" title={transfer.file_name}>
            {transfer.file_name}
          </span>
          {#if transfer.is_folder_zip}
            <span class="text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30 flex-shrink-0">
              Folder ZIP
            </span>
          {/if}
        </div>
        <div class="flex items-center gap-2 text-xs text-slate-400 mt-0.5 font-mono">
          <span>{formatBytes(transfer.file_size)}</span>
          {#if transfer.download_count > 0}
            <span>•</span>
            <span class="text-slate-400">{transfer.download_count} {transfer.download_count === 1 ? 'dl' : 'dls'}</span>
          {/if}
        </div>
      </div>
    </div>
  </div>

  <!-- Expiration & Status Row -->
  <div class="flex items-center justify-between gap-2 pt-1 border-t border-slate-700/50 text-xs">
    <!-- Countdown Badge -->
    <div class="flex items-center gap-1.5">
      {#if countdown.expired}
        <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium bg-rose-500/15 text-rose-400 border border-rose-500/30">
          <AlertTriangle class="w-3 h-3" />
          <span>{countdown.isBurn ? 'Burned' : 'Expired'}</span>
        </span>
      {:else if transfer.burn_on_read}
        <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium bg-orange-500/15 text-orange-300 border border-orange-500/30 animate-pulse">
          <Flame class="w-3 h-3 text-orange-400" />
          <span>Burn on read (1x)</span>
        </span>
      {:else}
        <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium bg-slate-800 text-cyan-300 border border-slate-700">
          <Clock class="w-3 h-3 text-cyan-400" />
          <span>{countdown.label}</span>
        </span>
      {/if}
    </div>

    <!-- Download Action -->
    {#if countdown.expired}
      <button
        disabled
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-slate-800 text-slate-500 cursor-not-allowed border border-slate-700/40"
      >
        <Download class="w-3.5 h-3.5" />
        <span>Unavailable</span>
      </button>
    {:else}
      <a
        href={downloadUrl}
        download={transfer.file_name}
        class="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-cyan-600 hover:bg-cyan-500 text-white shadow-sm shadow-cyan-600/30 hover:shadow-cyan-500/40 transition-all duration-150 active:scale-95"
      >
        <Download class="w-3.5 h-3.5" />
        <span>Download</span>
      </a>
    {/if}
  </div>

  {#if transfer.burn_on_read && !countdown.expired}
    <p class="text-[10px] text-amber-400/90 font-mono italic">
      ⚠️ Ephemeral transfer: File will permanently self-destruct from server after first download.
    </p>
  {/if}
</div>
