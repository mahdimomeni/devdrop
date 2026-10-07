<script>
  import {
    Download,
    Laptop,
    Smartphone,
    Share2,
    PlusSquare,
    CheckCircle2,
    X,
    Sparkles,
    ShieldCheck,
    Zap,
    ExternalLink,
    Layers,
    Monitor,
    Copy,
    Check,
    AlertTriangle
  } from 'lucide-svelte';
  import { promptInstall } from './pwaService.js';
  import { copyToClipboard } from './api.js';

  let {
    isOpen = false,
    pwaInfo = { isInstalled: false, isInstallable: true, platform: 'desktop', isIOS: false, isAndroid: false, isSecureContext: true, origin: '' },
    onClose = null
  } = $props();

  let activeTab = $state('auto'); // 'auto' | 'desktop' | 'ios' | 'android'
  let isInstalling = $state(false);
  let installResult = $state(null);
  let copiedOrigin = $state(false);

  function handleCopyOrigin() {
    const origin = pwaInfo?.origin || (typeof window !== 'undefined' ? window.location.origin : '');
    copyToClipboard(origin);
    copiedOrigin = true;
    setTimeout(() => {
      copiedOrigin = false;
    }, 2000);
  }

  let currentPlatform = $derived.by(() => {
    if (activeTab !== 'auto') return activeTab;
    if (pwaInfo.isIOS) return 'ios';
    if (pwaInfo.isAndroid) return 'android';
    return 'desktop';
  });

  async function handleInstallClick() {
    isInstalling = true;
    installResult = null;
    try {
      const res = await promptInstall();
      if (res.outcome === 'accepted' || res.installed) {
        installResult = 'success';
        setTimeout(() => {
          if (onClose) onClose();
        }, 1500);
      } else if (res.outcome === 'manual_instructions') {
        if (pwaInfo.isIOS) {
          activeTab = 'ios';
        } else {
          activeTab = currentPlatform;
        }
      } else {
        installResult = 'dismissed';
      }
    } catch (err) {
      console.warn('Install error:', err);
    } finally {
      isInstalling = false;
    }
  }

  function handleBackdropClick(e) {
    if (e.target === e.currentTarget && onClose) {
      onClose();
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Escape' && onClose) {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div
    role="presentation"
    onclick={handleBackdropClick}
    class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-950/80 backdrop-blur-md animate-in fade-in duration-200"
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="install-modal-title"
      class="w-full max-w-lg bg-slate-900 border border-slate-750/90 rounded-2xl shadow-2xl shadow-cyan-950/20 overflow-hidden flex flex-col animate-in zoom-in-95 duration-200"
    >
      <!-- Modal Header -->
      <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/60">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-gradient-to-tr from-cyan-500/20 to-indigo-500/20 border border-cyan-500/40 flex items-center justify-center text-cyan-400">
            <Download class="w-5 h-5 text-cyan-400" />
          </div>
          <div>
            <h2 id="install-modal-title" class="text-sm sm:text-base font-bold text-white font-mono flex items-center gap-2">
              Install DevDrop App
            </h2>
            <p class="text-[11px] text-slate-400 font-mono">
              Use DevDrop as a standalone app on your PC or phone
            </p>
          </div>
        </div>

        <button
          type="button"
          onclick={onClose}
          class="p-1.5 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
          aria-label="Close dialog"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Platform Selection Tabs -->
      <div class="px-5 pt-3 pb-2 bg-slate-900 flex items-center gap-1.5 border-b border-slate-800 text-xs font-mono">
        <span class="text-slate-500 text-[11px] mr-1 hidden sm:inline">Platform:</span>
        <button
          type="button"
          onclick={() => (activeTab = 'desktop')}
          class="px-2.5 py-1 rounded-lg transition-colors flex items-center gap-1.5 cursor-pointer {currentPlatform === 'desktop' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'}"
        >
          <Laptop class="w-3.5 h-3.5" />
          <span>Desktop (PC / Mac)</span>
        </button>

        <button
          type="button"
          onclick={() => (activeTab = 'android')}
          class="px-2.5 py-1 rounded-lg transition-colors flex items-center gap-1.5 cursor-pointer {currentPlatform === 'android' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'}"
        >
          <Smartphone class="w-3.5 h-3.5" />
          <span>Android</span>
        </button>

        <button
          type="button"
          onclick={() => (activeTab = 'ios')}
          class="px-2.5 py-1 rounded-lg transition-colors flex items-center gap-1.5 cursor-pointer {currentPlatform === 'ios' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'}"
        >
          <Sparkles class="w-3.5 h-3.5" />
          <span>iPhone / iPad</span>
        </button>
      </div>

      <!-- Modal Body Content -->
      <div class="p-5 flex flex-col gap-4 max-h-[70vh] overflow-y-auto">
        {#if pwaInfo.isInstalled}
          <div class="p-4 rounded-xl bg-emerald-950/40 border border-emerald-500/40 flex items-center gap-3 text-emerald-300 font-mono text-xs">
            <CheckCircle2 class="w-5 h-5 text-emerald-400 flex-shrink-0" />
            <div>
              <span class="font-bold">DevDrop is already installed!</span>
              <p class="text-[11px] text-emerald-400/80 mt-0.5">
                You are running the app directly from your home screen or desktop launcher.
              </p>
            </div>
          </div>
        {/if}

        <!-- App Preview Badge Card -->
        <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center gap-3.5">
          <img src="/icons/icon-192.png" alt="DevDrop App Icon" class="w-12 h-12 rounded-xl border border-slate-750 shadow-md flex-shrink-0" />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="font-bold text-slate-100 text-sm font-mono truncate">DevDrop LAN</span>
              <span class="text-[9px] px-1.5 py-0.5 rounded-md bg-cyan-950 text-cyan-300 border border-cyan-800/50 font-semibold uppercase">PWA</span>
            </div>
            <p class="text-[11px] text-slate-400 font-mono mt-0.5 truncate">
              Zero-config LAN messaging, code & file streaming
            </p>
          </div>
        </div>

        <!-- Platform Specific Instructions -->
        {#if currentPlatform === 'desktop'}
          <!-- Desktop Install Guide -->
          <div class="space-y-3 font-mono text-xs text-slate-300">
            <div class="p-3.5 rounded-xl bg-slate-800/60 border border-slate-750 space-y-2">
              <div class="font-semibold text-cyan-300 flex items-center gap-1.5">
                <Monitor class="w-4 h-4" /> Why install on PC?
              </div>
              <ul class="text-[11px] text-slate-300 space-y-1.5 pl-1 list-disc list-inside">
                <li>Runs in a clean, dedicated window without browser URL bar or tabs.</li>
                <li>Pins directly to your Windows Taskbar, Start Menu, or macOS Dock.</li>
                <li>Native OS notifications for LAN mentions and file drop transfers.</li>
                <li>Instant startup with cached offline shell.</li>
              </ul>
            </div>

            <div class="text-[11px] text-slate-400 leading-relaxed">
              Click <strong class="text-slate-200">Install Now</strong> below. If your browser does not show the prompt automatically, click the install icon (<span class="text-cyan-400 font-bold">⊕</span> or computer icon) in the right side of the browser's address bar.
            </div>
          </div>
        {:else if currentPlatform === 'android'}
          <!-- Android Install Guide -->
          <div class="space-y-3 font-mono text-xs text-slate-300">
            <!-- Alert & Fix for Chrome "This app cannot be installed" -->
            <div class="p-3.5 rounded-xl bg-amber-950/40 border border-amber-500/50 text-amber-200 font-mono text-xs space-y-2">
              <div class="flex items-center gap-2 font-bold text-amber-300">
                <AlertTriangle class="w-4 h-4 text-amber-400 flex-shrink-0" />
                <span>Fix: Chrome says "This app cannot be installed"</span>
              </div>
              <p class="text-[11px] text-amber-200/90 leading-relaxed">
                Mobile Chrome strictly requires a <strong>Secure Context (HTTPS)</strong> to install PWAs. Over plain HTTP on a LAN IP (<code class="bg-slate-900/80 px-1 py-0.5 rounded text-amber-300">{pwaInfo?.origin || 'http://192.10.105.53:8080'}</code>), Chrome disables the install button by default.
              </p>

              <div class="p-2.5 rounded-lg bg-slate-950/90 border border-amber-900/60 text-[11px] space-y-2">
                <div class="font-bold text-white flex items-center justify-between">
                  <span>⚡ 30-Second Fix in Chrome:</span>
                  <button
                    type="button"
                    onclick={handleCopyOrigin}
                    class="px-2 py-0.5 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 text-[10px] border border-amber-500/40 cursor-pointer flex items-center gap-1 font-mono transition-colors"
                  >
                    {#if copiedOrigin}
                      <Check class="w-3 h-3 text-emerald-400" /> Copied!
                    {:else}
                      <Copy class="w-3 h-3" /> Copy Origin URL
                    {/if}
                  </button>
                </div>
                <ol class="list-decimal list-inside space-y-1 text-slate-300 pl-0.5">
                  <li>
                    In Chrome, navigate to:<br/>
                    <code class="text-amber-300 select-all font-mono text-[10px] bg-slate-900 px-1.5 py-0.5 rounded border border-slate-800 inline-block mt-0.5">chrome://flags/#unsafely-treat-insecure-origin-as-secure</code>
                  </li>
                  <li>
                    Set to <strong class="text-emerald-400">Enabled</strong> and paste your DevDrop URL:<br/>
                    <code class="text-cyan-300 select-all font-mono text-[10px] bg-slate-900 px-1.5 py-0.5 rounded border border-slate-800 inline-block mt-0.5">{pwaInfo?.origin || 'http://192.10.105.53:8080'}</code>
                  </li>
                  <li>
                    Tap <strong class="text-white">Relaunch</strong> at the bottom of Chrome.
                  </li>
                  <li>
                    Return to DevDrop and tap <strong class="text-cyan-300">Install</strong>. It will install immediately!
                  </li>
                </ol>
              </div>
            </div>

            <div class="p-3.5 rounded-xl bg-slate-800/60 border border-slate-750 space-y-2">
              <div class="font-semibold text-cyan-300 flex items-center gap-1.5">
                <Smartphone class="w-4 h-4" /> Phone Experience
              </div>
              <ul class="text-[11px] text-slate-300 space-y-1.5 pl-1 list-disc list-inside">
                <li>Installs directly to your home screen and app drawer.</li>
                <li>Full-screen immersive LAN chat & code view.</li>
                <li>Zero app store downloads or developer accounts required.</li>
              </ul>
            </div>
          </div>
        {:else if currentPlatform === 'ios'}
          <!-- iOS Safari Install Guide -->
          <div class="space-y-3 font-mono text-xs text-slate-300">
            <div class="p-3.5 rounded-xl bg-slate-800/60 border border-slate-750 space-y-2">
              <div class="font-semibold text-cyan-300 flex items-center gap-1.5">
                <Sparkles class="w-4 h-4" /> Add to iPhone or iPad Home Screen
              </div>
              <p class="text-[11px] text-slate-400 leading-relaxed">
                Apple requires using Safari to install Progressive Web Apps:
              </p>
            </div>

            <!-- Step by step visual instructions -->
            <div class="space-y-2.5">
              <div class="flex items-start gap-3 p-2.5 rounded-xl bg-slate-950/70 border border-slate-800">
                <div class="w-6 h-6 rounded-lg bg-cyan-500/20 text-cyan-400 flex items-center justify-center font-bold text-xs flex-shrink-0">
                  1
                </div>
                <div class="text-[11px] leading-snug">
                  Tap the <strong class="text-cyan-300 inline-flex items-center gap-1"><Share2 class="w-3.5 h-3.5" /> Share</strong> icon in the Safari bottom toolbar.
                </div>
              </div>

              <div class="flex items-start gap-3 p-2.5 rounded-xl bg-slate-950/70 border border-slate-800">
                <div class="w-6 h-6 rounded-lg bg-cyan-500/20 text-cyan-400 flex items-center justify-center font-bold text-xs flex-shrink-0">
                  2
                </div>
                <div class="text-[11px] leading-snug">
                  Scroll down the share sheet and tap <strong class="text-cyan-300 inline-flex items-center gap-1"><PlusSquare class="w-3.5 h-3.5" /> Add to Home Screen</strong>.
                </div>
              </div>

              <div class="flex items-start gap-3 p-2.5 rounded-xl bg-slate-950/70 border border-slate-800">
                <div class="w-6 h-6 rounded-lg bg-cyan-500/20 text-cyan-400 flex items-center justify-center font-bold text-xs flex-shrink-0">
                  3
                </div>
                <div class="text-[11px] leading-snug">
                  Tap <strong class="text-emerald-400 font-bold">Add</strong> in the top-right corner to place DevDrop on your home screen.
                </div>
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Modal Footer Action -->
      <div class="px-5 py-3.5 bg-slate-950 border-t border-slate-800 flex items-center justify-between gap-3">
        <span class="text-[11px] text-slate-500 font-mono">
          LAN Offline Supported
        </span>

        <div class="flex items-center gap-2">
          <button
            type="button"
            onclick={onClose}
            class="px-3 py-1.5 text-xs font-mono text-slate-400 hover:text-white rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
          >
            Close
          </button>

          {#if currentPlatform !== 'ios'}
            <button
              type="button"
              onclick={handleInstallClick}
              disabled={isInstalling}
              class="px-4 py-1.5 bg-gradient-to-r from-cyan-600 to-indigo-600 hover:from-cyan-500 hover:to-indigo-500 text-white rounded-lg text-xs font-mono font-semibold transition-all shadow-md shadow-cyan-950/50 flex items-center gap-2 cursor-pointer disabled:opacity-50"
            >
              <Download class="w-3.5 h-3.5" />
              <span>{isInstalling ? 'Installing...' : 'Install Now'}</span>
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
