<script>
  import * as Diff from 'diff';
  import { X, GitCompare, Copy, Check } from 'lucide-svelte';
  import { copyToClipboard } from './api.js';

  let { oldCode = '', newCode = '', oldLabel = 'Snippet A', newLabel = 'Snippet B', onClose } = $props();

  let copied = $state(false);

  // Compute line diff
  let diffParts = $derived.by(() => {
    return Diff.diffLines(oldCode, newCode);
  });

  async function copyNew() {
    await copyToClipboard(newCode);
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-0 sm:p-4 animate-in fade-in duration-200">
  <div class="relative w-full h-full sm:h-auto max-w-5xl sm:max-h-[90vh] flex flex-col bg-slate-900 border-0 sm:border border-slate-700/80 rounded-none sm:rounded-2xl shadow-2xl overflow-hidden">
    <!-- Header -->
    <div class="flex items-center justify-between px-3 sm:px-6 py-3 sm:py-4 bg-slate-800/80 border-b border-slate-700/70 pt-[max(0.75rem,env(safe-area-inset-top))]">
      <div class="flex items-center space-x-2 sm:space-x-3 min-w-0">
        <div class="p-1.5 sm:p-2 bg-indigo-500/20 text-indigo-400 rounded-lg flex-shrink-0">
          <GitCompare class="w-4 h-4 sm:w-5 sm:h-5" />
        </div>
        <div class="min-w-0">
          <h2 class="text-sm sm:text-base font-semibold text-slate-100 flex items-center gap-1.5 sm:gap-2 truncate">
            <span>Diff Comparison</span>
            <span class="text-[10px] sm:text-xs px-1.5 sm:px-2 py-0.5 rounded-full bg-slate-700 text-slate-300 font-mono">Diff</span>
          </h2>
          <p class="text-[10px] sm:text-xs text-slate-400 font-mono truncate">
            Red = Removed (-) · Green = Added (+)
          </p>
        </div>
      </div>

      <div class="flex items-center space-x-1.5 sm:space-x-2 flex-shrink-0">
        <button
          onclick={copyNew}
          class="flex items-center gap-1 px-2.5 sm:px-3 py-1.5 text-xs font-medium text-slate-300 bg-slate-800 hover:bg-slate-700 border border-slate-600 rounded-lg transition-colors cursor-pointer"
        >
          {#if copied}
            <Check class="w-3.5 h-3.5 text-emerald-400" />
            <span class="text-emerald-400 text-xs hidden xs:inline">Copied Latest</span>
          {:else}
            <Copy class="w-3.5 h-3.5" />
            <span class="hidden xs:inline">Copy Latest</span>
          {/if}
        </button>

        <button
          onclick={onClose}
          class="p-1.5 sm:p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors cursor-pointer"
          title="Close Diff"
        >
          <X class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Comparison Subheader -->
    <div class="grid grid-cols-2 bg-slate-950/60 border-b border-slate-800 px-3 sm:px-6 py-1.5 sm:py-2 text-[10px] sm:text-xs font-mono">
      <div class="text-rose-400/80 flex items-center gap-1 truncate">
        <span>---</span> <span class="truncate">{oldLabel}</span>
      </div>
      <div class="text-emerald-400/80 flex items-center gap-1 truncate">
        <span>+++</span> <span class="truncate">{newLabel}</span>
      </div>
    </div>

    <!-- Diff Content Body -->
    <div class="flex-1 overflow-auto p-2.5 sm:p-4 bg-slate-950 font-mono text-[11px] sm:text-xs leading-relaxed select-text">
      {#each diffParts as part}
        {#if part.added}
          <div class="bg-emerald-950/40 text-emerald-300 border-l-2 border-emerald-500 px-2 sm:px-3 py-0.5 whitespace-pre-wrap">
            <span class="inline-block w-4 text-emerald-500 font-bold select-none">+</span>{part.value}
          </div>
        {:else if part.removed}
          <div class="bg-rose-950/40 text-rose-300 border-l-2 border-rose-500 px-2 sm:px-3 py-0.5 whitespace-pre-wrap opacity-80">
            <span class="inline-block w-4 text-rose-500 font-bold select-none">-</span>{part.value}
          </div>
        {:else}
          <div class="text-slate-400 px-2 sm:px-3 py-0.5 whitespace-pre-wrap">
            <span class="inline-block w-4 text-slate-600 select-none"> </span>{part.value}
          </div>
        {/if}
      {/each}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end px-4 sm:px-6 py-2.5 sm:py-3 bg-slate-900 border-t border-slate-800 text-xs text-slate-400 pb-[max(0.625rem,env(safe-area-inset-bottom))]">
      <button
        onclick={onClose}
        class="w-full sm:w-auto px-5 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 font-medium rounded-lg transition-colors cursor-pointer text-center"
      >
        Done
      </button>
    </div>
  </div>
</div>
