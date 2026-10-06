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

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4 animate-in fade-in duration-200">
  <div class="relative w-full max-w-5xl max-h-[90vh] flex flex-col bg-slate-900 border border-slate-700/80 rounded-2xl shadow-2xl overflow-hidden">
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-4 bg-slate-800/80 border-b border-slate-700/70">
      <div class="flex items-center space-x-3">
        <div class="p-2 bg-indigo-500/20 text-indigo-400 rounded-lg">
          <GitCompare class="w-5 h-5" />
        </div>
        <div>
          <h2 class="text-base font-semibold text-slate-100 flex items-center gap-2">
            Code Snippet Comparison
            <span class="text-xs px-2 py-0.5 rounded-full bg-slate-700 text-slate-300 font-mono">Diff</span>
          </h2>
          <p class="text-xs text-slate-400 font-mono">
            Red = Removed (-) · Green = Added (+)
          </p>
        </div>
      </div>

      <div class="flex items-center space-x-2">
        <button
          onclick={copyNew}
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-300 bg-slate-800 hover:bg-slate-700 border border-slate-600 rounded-lg transition-colors"
        >
          {#if copied}
            <Check class="w-3.5 h-3.5 text-emerald-400" />
            <span class="text-emerald-400">Copied Latest</span>
          {:else}
            <Copy class="w-3.5 h-3.5" />
            <span>Copy Latest</span>
          {/if}
        </button>

        <button
          onclick={onClose}
          class="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
          title="Close Diff"
        >
          <X class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Comparison Subheader -->
    <div class="grid grid-cols-2 bg-slate-950/60 border-b border-slate-800 px-6 py-2 text-xs font-mono">
      <div class="text-rose-400/80 flex items-center gap-1">
        <span>---</span> <span>{oldLabel} (Baseline)</span>
      </div>
      <div class="text-emerald-400/80 flex items-center gap-1">
        <span>+++</span> <span>{newLabel} (Revised)</span>
      </div>
    </div>

    <!-- Diff Content Body -->
    <div class="flex-1 overflow-auto p-4 bg-slate-950 font-mono text-xs leading-relaxed select-text">
      {#each diffParts as part}
        {#if part.added}
          <div class="bg-emerald-950/40 text-emerald-300 border-l-2 border-emerald-500 px-3 py-0.5 whitespace-pre-wrap">
            <span class="inline-block w-4 text-emerald-500 font-bold select-none">+</span>{part.value}
          </div>
        {:else if part.removed}
          <div class="bg-rose-950/40 text-rose-300 border-l-2 border-rose-500 px-3 py-0.5 whitespace-pre-wrap opacity-80">
            <span class="inline-block w-4 text-rose-500 font-bold select-none">-</span>{part.value}
          </div>
        {:else}
          <div class="text-slate-400 px-3 py-0.5 whitespace-pre-wrap">
            <span class="inline-block w-4 text-slate-600 select-none"> </span>{part.value}
          </div>
        {/if}
      {/each}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end px-6 py-3 bg-slate-900 border-t border-slate-800 text-xs text-slate-400">
      <button
        onclick={onClose}
        class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 font-medium rounded-lg transition-colors"
      >
        Done
      </button>
    </div>
  </div>
</div>
