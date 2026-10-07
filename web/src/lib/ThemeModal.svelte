<script>
  import { onMount } from 'svelte';
  import {
    Palette,
    X,
    Check,
    Sun,
    Moon,
    Sparkles,
    Terminal,
    Radio,
    Flame,
    Shield,
    RotateCcw,
    Sliders,
    Eye,
    Laptop,
    CheckCircle2
  } from 'lucide-svelte';
  import {
    BASE_THEMES,
    ACCENT_COLORS,
    THEME_PRESETS,
    getDefaultThemeSettings
  } from './themeService.js';

  let {
    isOpen = false,
    settings = null,
    onUpdateSettings = null,
    onClose = null
  } = $props();

  // Active tab: 'presets' | 'custom'
  let activeTab = $state('presets');

  let currentBase = $derived(settings?.baseTheme || 'dark');
  let currentAccent = $derived(settings?.accentColor || 'cyan');

  let activeBaseObj = $derived(
    BASE_THEMES.find((t) => t.id === currentBase) || BASE_THEMES[0]
  );
  let activeAccentObj = $derived(
    ACCENT_COLORS.find((a) => a.id === currentAccent) || ACCENT_COLORS[0]
  );

  function handleSelectBase(baseId) {
    if (!onUpdateSettings) return;
    onUpdateSettings({
      baseTheme: baseId,
      accentColor: currentAccent,
    });
  }

  function handleSelectAccent(accentId) {
    if (!onUpdateSettings) return;
    onUpdateSettings({
      baseTheme: currentBase,
      accentColor: accentId,
    });
  }

  function handleApplyPreset(preset) {
    if (!onUpdateSettings) return;
    onUpdateSettings({
      baseTheme: preset.baseTheme,
      accentColor: preset.accentColor,
    });
  }

  function handleResetDefault() {
    if (!onUpdateSettings) return;
    onUpdateSettings(getDefaultThemeSettings());
  }

  function handleKeyDown(e) {
    if (e.key === 'Escape' && isOpen && onClose) {
      onClose();
    }
  }

  function getPresetIcon(iconName) {
    switch (iconName) {
      case 'Moon': return Moon;
      case 'Sun': return Sun;
      case 'Terminal': return Terminal;
      case 'Sparkles': return Sparkles;
      case 'Flame': return Flame;
      case 'Shield': return Shield;
      case 'Radio': return Radio;
      default: return Sparkles;
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <!-- Modal Backdrop -->
  <div
    role="presentation"
    onclick={onClose}
    class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/75 backdrop-blur-sm animate-in fade-in duration-200"
  >
    <!-- Modal Card -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      role="dialog"
      tabindex="-1"
      aria-modal="true"
      aria-labelledby="theme-modal-title"
      class="bg-slate-900 border border-slate-750 rounded-2xl w-full max-w-2xl max-h-[92vh] flex flex-col shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Header -->
      <div class="px-4 sm:px-6 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/60 flex-shrink-0">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-cyan-950/80 border border-cyan-800/80 flex items-center justify-center text-cyan-400 shadow-sm">
            <Palette class="w-4 h-4" />
          </div>
          <div>
            <h2 id="theme-modal-title" class="text-sm sm:text-base font-semibold text-slate-100 font-mono">
              Appearance & Themes
            </h2>
            <p class="text-[11px] text-slate-400 font-mono">
              Custom surface palettes and vibrant color accents
            </p>
          </div>
        </div>

        <button
          type="button"
          onclick={onClose}
          class="p-1.5 text-slate-400 hover:text-slate-100 hover:bg-slate-800 rounded-lg transition-colors cursor-pointer"
          aria-label="Close dialog"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Navigation Tabs: Presets vs Custom Palette -->
      <div class="px-4 sm:px-6 pt-3 pb-2 border-b border-slate-800/80 bg-slate-900/90 flex items-center justify-between gap-2 flex-shrink-0">
        <div class="flex items-center gap-1.5 p-1 bg-slate-950 rounded-xl border border-slate-800 text-xs font-mono">
          <button
            type="button"
            onclick={() => (activeTab = 'presets')}
            class="px-3 py-1.5 rounded-lg font-medium transition-all cursor-pointer flex items-center gap-1.5 {activeTab === 'presets'
              ? 'bg-cyan-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200'}"
          >
            <Sparkles class="w-3.5 h-3.5" />
            <span>Curated Presets</span>
          </button>
          <button
            type="button"
            onclick={() => (activeTab = 'custom')}
            class="px-3 py-1.5 rounded-lg font-medium transition-all cursor-pointer flex items-center gap-1.5 {activeTab === 'custom'
              ? 'bg-cyan-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200'}"
          >
            <Sliders class="w-3.5 h-3.5" />
            <span>Customize (Mix & Match)</span>
          </button>
        </div>

        <!-- Quick Summary Badge -->
        <div class="hidden sm:flex items-center gap-2 text-[11px] font-mono text-slate-400">
          <span class="flex items-center gap-1 px-2 py-0.5 rounded-md bg-slate-800 border border-slate-700/80">
            <span class="w-2 h-2 rounded-full" style="background-color: {activeBaseObj.bg}; border: 1px solid {activeBaseObj.border};"></span>
            <span class="text-slate-300 font-medium">{activeBaseObj.name}</span>
          </span>
          <span>+</span>
          <span class="flex items-center gap-1 px-2 py-0.5 rounded-md bg-slate-800 border border-slate-700/80">
            <span class="w-2 h-2 rounded-full" style="background-color: {activeAccentObj.hex}; box-shadow: 0 0 6px {activeAccentObj.hex};"></span>
            <span class="text-slate-300 font-medium">{activeAccentObj.name}</span>
          </span>
        </div>
      </div>

      <!-- Scrollable Content -->
      <div class="flex-1 overflow-y-auto p-4 sm:p-6 space-y-6">
        
        {#if activeTab === 'presets'}
          <!-- Curated Presets Section -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs font-mono font-bold uppercase tracking-wider text-slate-300">
                1-Click Curated Presets
              </span>
              <span class="text-[11px] font-mono text-slate-400">
                Handcrafted color pairings
              </span>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              {#each THEME_PRESETS as preset (preset.id)}
                {@const isSelected = currentBase === preset.baseTheme && currentAccent === preset.accentColor}
                {@const presetBase = BASE_THEMES.find((t) => t.id === preset.baseTheme) || BASE_THEMES[0]}
                {@const presetAccent = ACCENT_COLORS.find((a) => a.id === preset.accentColor) || ACCENT_COLORS[0]}
                {@const IconComp = getPresetIcon(preset.icon)}

                <button
                  type="button"
                  onclick={() => handleApplyPreset(preset)}
                  class="group relative flex items-center justify-between p-3 rounded-xl border text-left transition-all duration-150 cursor-pointer {isSelected
                    ? 'bg-cyan-950/40 border-cyan-500 ring-1 ring-cyan-500/50 shadow-md shadow-cyan-500/10'
                    : 'bg-slate-950/70 hover:bg-slate-800/80 border-slate-800 hover:border-slate-700'}"
                >
                  <div class="flex items-center gap-3 min-w-0">
                    <div
                      class="w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 transition-transform group-hover:scale-105 border"
                      style="background-color: {presetBase.surface}; border-color: {presetBase.border}; color: {presetAccent.hex};"
                    >
                      <IconComp class="w-4 h-4" />
                    </div>
                    <div class="min-w-0">
                      <div class="flex items-center gap-2">
                        <span class="text-xs font-mono font-semibold text-slate-100 truncate">
                          {preset.name}
                        </span>
                        {#if isSelected}
                          <span class="text-[9px] px-1.5 py-0.2 rounded font-bold font-mono bg-cyan-500 text-slate-950">
                            Active
                          </span>
                        {/if}
                      </div>
                      <p class="text-[11px] text-slate-400 truncate mt-0.5">
                        {preset.tagline}
                      </p>
                    </div>
                  </div>

                  <!-- Color Swatches & Checkmark -->
                  <div class="flex items-center gap-2 flex-shrink-0 ml-2">
                    <div class="flex items-center -space-x-1">
                      <div
                        class="w-4 h-4 rounded-full border border-slate-700 shadow-xs"
                        style="background-color: {presetBase.bg};"
                        title="Surface: {presetBase.name}"
                      ></div>
                      <div
                        class="w-4 h-4 rounded-full border border-slate-700 shadow-xs"
                        style="background-color: {presetAccent.hex};"
                        title="Accent: {presetAccent.name}"
                      ></div>
                    </div>
                    <div class="w-5 h-5 rounded-full flex items-center justify-center {isSelected ? 'bg-cyan-500 text-slate-950' : 'text-slate-600'}">
                      {#if isSelected}
                        <Check class="w-3.5 h-3.5 stroke-[3]" />
                      {/if}
                    </div>
                  </div>
                </button>
              {/each}
            </div>
          </div>
        {/if}

        {#if activeTab === 'custom'}
          <!-- 1. Base Themes (Surface Palette) -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs font-mono font-bold uppercase tracking-wider text-slate-300">
                1. Select Base Surface Theme
              </span>
              <span class="text-[11px] font-mono text-slate-400">
                Sets background & card surfaces
              </span>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2.5">
              {#each BASE_THEMES as theme (theme.id)}
                {@const isSelected = currentBase === theme.id}
                <button
                  type="button"
                  onclick={() => handleSelectBase(theme.id)}
                  class="group relative flex flex-col p-2.5 rounded-xl border text-left transition-all duration-150 cursor-pointer overflow-hidden {isSelected
                    ? 'border-cyan-500 ring-2 ring-cyan-500/40 bg-slate-800/90 shadow-lg shadow-cyan-500/10'
                    : 'bg-slate-950/70 hover:bg-slate-800/60 border-slate-800 hover:border-slate-700'}"
                >
                  <!-- Mini UI Preview Box -->
                  <div
                    class="w-full h-14 rounded-lg p-1.5 flex flex-col justify-between mb-2 border overflow-hidden transition-transform group-hover:scale-[1.02]"
                    style="background-color: {theme.bg}; border-color: {theme.border};"
                  >
                    <!-- Mini Top Bar -->
                    <div class="flex items-center justify-between px-1 py-0.5 rounded" style="background-color: {theme.surface};">
                      <div class="flex items-center gap-1">
                        <span class="w-1.5 h-1.5 rounded-full" style="background-color: {activeAccentObj.hex};"></span>
                        <span class="w-8 h-1 rounded" style="background-color: {theme.text}; opacity: 0.8;"></span>
                      </div>
                      <span class="w-3 h-1 rounded" style="background-color: {theme.text}; opacity: 0.4;"></span>
                    </div>

                    <!-- Mini Message Bubble -->
                    <div class="flex items-center gap-1">
                      <div class="px-1.5 py-0.5 rounded text-[8px] font-bold" style="background-color: {activeAccentObj.hex}; color: #ffffff;">
                        Aa
                      </div>
                      <div class="flex-1 h-1.5 rounded" style="background-color: {theme.card};"></div>
                    </div>
                  </div>

                  <!-- Details -->
                  <div class="flex items-center justify-between">
                    <span class="text-xs font-mono font-semibold text-slate-100 truncate">
                      {theme.name}
                    </span>
                    {#if isSelected}
                      <CheckCircle2 class="w-3.5 h-3.5 text-cyan-400 flex-shrink-0" />
                    {/if}
                  </div>
                  <span class="text-[10px] text-slate-400 font-mono mt-0.5 truncate">
                    {theme.subtitle}
                  </span>
                </button>
              {/each}
            </div>
          </div>

          <!-- 2. Accent Colors Section -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs font-mono font-bold uppercase tracking-wider text-slate-300">
                2. Select Vibrant Accent Color
              </span>
              <span class="text-[11px] font-mono text-slate-400">
                Highlights, buttons & glows
              </span>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
              {#each ACCENT_COLORS as accent (accent.id)}
                {@const isSelected = currentAccent === accent.id}
                <button
                  type="button"
                  onclick={() => handleSelectAccent(accent.id)}
                  class="flex items-center gap-2.5 p-2.5 rounded-xl border text-left transition-all duration-150 cursor-pointer {isSelected
                    ? 'border-cyan-500 bg-cyan-950/30 ring-1 ring-cyan-500/50 shadow-sm'
                    : 'bg-slate-950/70 hover:bg-slate-800/60 border-slate-800 hover:border-slate-700'}"
                >
                  <div
                    class="relative w-6 h-6 rounded-full flex items-center justify-center flex-shrink-0 shadow-md transition-transform group-hover:scale-110"
                    style="background-color: {accent.hex}; box-shadow: 0 0 10px {accent.glow};"
                  >
                    {#if isSelected}
                      <Check class="w-3.5 h-3.5 text-white stroke-[3]" />
                    {/if}
                  </div>
                  <div class="min-w-0">
                    <span class="block text-xs font-mono font-medium text-slate-100 truncate">
                      {accent.name}
                    </span>
                    <span class="block text-[10px] text-slate-400 font-mono">
                      {accent.hex}
                    </span>
                  </div>
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- 3. Live Interactive Preview Box -->
        <div class="bg-slate-950/90 border border-slate-800 rounded-xl p-3.5 sm:p-4 space-y-3">
          <div class="flex items-center justify-between text-xs font-mono text-slate-400 border-b border-slate-800/80 pb-2">
            <span class="flex items-center gap-1.5 font-semibold text-slate-200">
              <Eye class="w-3.5 h-3.5 text-cyan-400" />
              <span>Real-Time Workspace Preview</span>
            </span>
            <span class="text-[10px] text-cyan-400 font-medium">
              Live updates applied immediately
            </span>
          </div>

          <!-- Mock Message Preview -->
          <div class="space-y-2.5">
            <!-- Incoming message -->
            <div class="flex flex-col items-start max-w-[85%]">
              <span class="text-[10px] font-mono text-slate-400 mb-0.5">Alex (Node 2) • 2m ago</span>
              <div class="px-3 py-1.5 rounded-xl text-xs bg-slate-800 text-slate-200 border border-slate-700">
                Pushed update to DevDrop LAN! Testing the new theme colors.
              </div>
            </div>

            <!-- Outgoing message with chosen accent -->
            <div class="flex flex-col items-end max-w-[85%] ml-auto">
              <span class="text-[10px] font-mono text-cyan-400 mb-0.5 font-semibold">You • Just now</span>
              <div class="px-3 py-1.5 rounded-xl text-xs bg-cyan-600 text-white shadow-md">
                Looks stunning! Colors and contrast adapt seamlessly.
              </div>
            </div>
          </div>

          <!-- Mock controls row -->
          <div class="pt-2 border-t border-slate-800/60 flex items-center justify-between text-xs font-mono">
            <div class="flex items-center gap-1.5">
              <span class="px-2 py-0.5 rounded-full bg-cyan-950 text-cyan-400 text-[10px] font-bold border border-cyan-800/50">
                LAN Broadcast
              </span>
              <span class="text-[10px] text-slate-400">AES-GCM • Active</span>
            </div>
            <button
              type="button"
              class="px-2.5 py-1 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-xs font-bold transition-colors cursor-pointer shadow-sm"
            >
              Send Snippet
            </button>
          </div>
        </div>

      </div>

      <!-- Footer Actions -->
      <div class="px-4 sm:px-6 py-3.5 border-t border-slate-800 bg-slate-950/70 flex items-center justify-between gap-3 flex-shrink-0">
        <button
          type="button"
          onclick={handleResetDefault}
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-slate-400 hover:text-slate-200 hover:bg-slate-800/80 rounded-lg transition-colors cursor-pointer"
          title="Reset back to DevDrop Slate Dark + Electric Cyan"
        >
          <RotateCcw class="w-3.5 h-3.5" />
          <span>Reset Defaults</span>
        </button>

        <button
          type="button"
          onclick={onClose}
          class="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-xl text-xs font-mono font-bold transition-all shadow-md hover:shadow-cyan-500/20 active:scale-95 cursor-pointer"
        >
          Done
        </button>
      </div>
    </div>
  </div>
{/if}
