<script>
  import {
    Bell,
    BellRing,
    BellOff,
    Volume2,
    VolumeX,
    Sparkles,
    Check,
    X,
    ShieldAlert,
    Monitor,
    Info,
    AtSign,
    Radio
  } from 'lucide-svelte';
  import {
    getNotificationPermission,
    requestNotificationPermission,
    isNotificationSupported,
    playNotificationChime,
    showDesktopNotification
  } from './notificationService.js';

  let {
    isOpen = false,
    settings = {
      enabled: true,
      soundEnabled: true,
      desktopEnabled: true,
      scope: 'dms_mentions',
    },
    onUpdateSettings,
    onClose,
    onTestNotification = null,
  } = $props();

  let permissionState = $state(getNotificationPermission());
  let isRequesting = $state(false);
  let testSuccessToast = $state(false);

  // Sync permission state when modal opens
  $effect(() => {
    if (isOpen) {
      permissionState = getNotificationPermission();
    }
  });

  async function handleRequestPermission() {
    isRequesting = true;
    try {
      const res = await requestNotificationPermission();
      permissionState = res;
      if (res === 'granted') {
        onUpdateSettings({ ...settings, desktopEnabled: true });
      }
    } finally {
      isRequesting = false;
    }
  }

  function toggleMaster() {
    onUpdateSettings({ ...settings, enabled: !settings.enabled });
  }

  function toggleSound() {
    onUpdateSettings({ ...settings, soundEnabled: !settings.soundEnabled });
  }

  function toggleDesktop() {
    onUpdateSettings({ ...settings, desktopEnabled: !settings.desktopEnabled });
  }

  function setScope(newScope) {
    onUpdateSettings({ ...settings, scope: newScope });
  }

  function handleTriggerTest() {
    playNotificationChime(settings.scope === 'dms_mentions' ? 'mention' : 'default');

    if (permissionState === 'granted' && settings.desktopEnabled && settings.enabled) {
      showDesktopNotification({
        title: 'DevDrop • Notification Test',
        body: 'Audio and desktop alerts are configured successfully!',
        tag: 'test-notification',
      });
    }

    if (onTestNotification) {
      onTestNotification();
    }

    testSuccessToast = true;
    setTimeout(() => {
      testSuccessToast = false;
    }, 3000);
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && isOpen) {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
  <!-- Backdrop -->
  <div
    role="presentation"
    onclick={onClose}
    class="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-3 sm:p-4 animate-in fade-in duration-150"
  >
    <!-- Modal Window -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      role="dialog"
      tabindex="-1"
      aria-modal="true"
      aria-label="Notification Preferences"
      onclick={(e) => e.stopPropagation()}
      class="bg-slate-900 border border-slate-750/90 rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden flex flex-col animate-in zoom-in-95 duration-150 text-slate-100 max-h-[90vh]"
    >
      <!-- Modal Header -->
      <div class="px-5 py-4 border-b border-slate-800 bg-slate-950/50 flex items-center justify-between flex-shrink-0">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-cyan-500/15 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
            {#if !settings.enabled}
              <BellOff class="w-4 h-4 text-slate-400" />
            {:else if settings.soundEnabled}
              <BellRing class="w-4 h-4 text-cyan-400" />
            {:else}
              <Bell class="w-4 h-4 text-cyan-400" />
            {/if}
          </div>
          <div>
            <h2 class="text-sm sm:text-base font-semibold font-mono text-white flex items-center gap-2">
              Notifications & Alerts
            </h2>
            <p class="text-xs text-slate-400 font-mono">
              Desktop alerts, audio chimes, and mention triggers
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

      <!-- Modal Body -->
      <div class="p-5 space-y-4 overflow-y-auto min-h-0 text-xs font-mono">
        <!-- Master Enable Switch -->
        <div class="p-3.5 rounded-xl bg-slate-950/60 border border-slate-800/90 flex items-center justify-between gap-4">
          <div>
            <span class="font-semibold text-slate-200 block text-xs sm:text-sm">
              Enable All Notifications
            </span>
            <span class="text-[11px] text-slate-400">
              Master switch for desktop alerts, in-app banners, and chimes
            </span>
          </div>

          <button
            type="button"
            onclick={toggleMaster}
            class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {settings.enabled ? 'bg-cyan-500' : 'bg-slate-700'}"
            role="switch"
            aria-checked={settings.enabled}
            aria-label="Enable all notifications"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out {settings.enabled ? 'translate-x-5' : 'translate-x-0'}"
            ></span>
          </button>
        </div>

        {#if settings.enabled}
          <!-- Browser Desktop Notifications Card -->
          <div class="p-4 rounded-xl bg-slate-950/40 border border-slate-800/80 space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Monitor class="w-4 h-4 text-cyan-400 flex-shrink-0" />
                <span class="font-semibold text-slate-200">System Desktop Notifications</span>
              </div>

              {#if permissionState === 'granted'}
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 border border-emerald-800/60 text-[10px] font-bold">
                  <Check class="w-3 h-3" /> Allowed
                </span>
              {:else if permissionState === 'denied'}
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-rose-950/80 text-rose-400 border border-rose-800/60 text-[10px] font-bold">
                  <ShieldAlert class="w-3 h-3" /> Blocked
                </span>
              {:else if permissionState === 'unsupported'}
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 border border-slate-700 text-[10px]">
                  LAN HTTP
                </span>
              {:else}
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-amber-950/80 text-amber-300 border border-amber-800/60 text-[10px] font-bold">
                  Prompt Needed
                </span>
              {/if}
            </div>

            <p class="text-[11px] text-slate-400 leading-relaxed">
              Receives native OS popups even when DevDrop is minimized or another application has focus.
            </p>

            {#if permissionState === 'default'}
              <div class="pt-1">
                <button
                  type="button"
                  onclick={handleRequestPermission}
                  disabled={isRequesting}
                  class="w-full flex items-center justify-center gap-2 px-3 py-2 bg-gradient-to-r from-cyan-600 to-indigo-600 hover:from-cyan-500 hover:to-indigo-500 text-white font-semibold rounded-lg shadow-md transition-all cursor-pointer text-xs disabled:opacity-50"
                >
                  <Bell class="w-3.5 h-3.5" />
                  <span>{isRequesting ? 'Requesting Permission...' : 'Allow Browser Notifications'}</span>
                </button>
              </div>
            {:else if permissionState === 'granted'}
              <div class="flex items-center justify-between pt-1">
                <span class="text-[11px] text-slate-300">Deliver desktop banner popups</span>
                <button
                  type="button"
                  onclick={toggleDesktop}
                  class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {settings.desktopEnabled ? 'bg-cyan-500' : 'bg-slate-700'}"
                  role="switch"
                  aria-checked={settings.desktopEnabled}
                  aria-label="Toggle desktop banners"
                >
                  <span
                    class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {settings.desktopEnabled ? 'translate-x-4' : 'translate-x-0'}"
                  ></span>
                </button>
              </div>
            {:else if permissionState === 'denied'}
              <div class="p-2.5 rounded-lg bg-rose-950/30 border border-rose-900/50 text-[11px] text-rose-300 flex items-start gap-2">
                <ShieldAlert class="w-4 h-4 text-rose-400 flex-shrink-0 mt-0.5" />
                <span>
                  Desktop alerts were blocked by your browser settings. To enable them, click the lock/settings icon in your browser address bar and set <strong>Notifications</strong> to <strong>Allow</strong>.
                </span>
              </div>
            {:else if permissionState === 'unsupported'}
              <div class="p-2.5 rounded-lg bg-slate-800/50 border border-slate-700/60 text-[11px] text-slate-300 flex items-start gap-2">
                <Info class="w-4 h-4 text-cyan-400 flex-shrink-0 mt-0.5" />
                <span>
                  Browser desktop popups require HTTPS or localhost. On LAN IP over plain HTTP, in-app banners and audio chimes remain fully active!
                </span>
              </div>
            {/if}
          </div>

          <!-- Sound Effects Card -->
          <div class="p-4 rounded-xl bg-slate-950/40 border border-slate-800/80 space-y-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                {#if settings.soundEnabled}
                  <Volume2 class="w-4 h-4 text-cyan-400" />
                {:else}
                  <VolumeX class="w-4 h-4 text-slate-500" />
                {/if}
                <span class="font-semibold text-slate-200">Audio Chimes</span>
              </div>

              <button
                type="button"
                onclick={toggleSound}
                class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {settings.soundEnabled ? 'bg-cyan-500' : 'bg-slate-700'}"
                role="switch"
                aria-checked={settings.soundEnabled}
                aria-label="Toggle audio chimes"
              >
                <span
                  class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {settings.soundEnabled ? 'translate-x-4' : 'translate-x-0'}"
                ></span>
              </button>
            </div>
            <p class="text-[11px] text-slate-400">
              Plays a subtle, harmonious Web Audio chime synthesized in real-time when new messages arrive.
            </p>
          </div>

          <!-- Notification Trigger Scope Card -->
          <div class="p-4 rounded-xl bg-slate-950/40 border border-slate-800/80 space-y-2.5">
            <div class="flex items-center gap-2">
              <AtSign class="w-4 h-4 text-amber-400" />
              <span class="font-semibold text-slate-200">Notify Me On</span>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
              <button
                type="button"
                onclick={() => setScope('dms_mentions')}
                class="p-2.5 rounded-lg border text-left transition-all cursor-pointer flex flex-col gap-1 {settings.scope === 'dms_mentions' ? 'bg-cyan-950/50 border-cyan-500/80 ring-1 ring-cyan-500/30 text-white' : 'bg-slate-900/60 border-slate-800 hover:border-slate-700 text-slate-300'}"
              >
                <div class="flex items-center justify-between">
                  <span class="font-semibold text-xs flex items-center gap-1.5">
                    <AtSign class="w-3.5 h-3.5 text-cyan-400" />
                    DMs & Mentions
                  </span>
                  {#if settings.scope === 'dms_mentions'}
                    <Check class="w-3.5 h-3.5 text-cyan-400" />
                  {/if}
                </div>
                <span class="text-[10px] text-slate-400">
                  Only when messaged directly or mentioned
                </span>
              </button>

              <button
                type="button"
                onclick={() => setScope('all')}
                class="p-2.5 rounded-lg border text-left transition-all cursor-pointer flex flex-col gap-1 {settings.scope === 'all' ? 'bg-cyan-950/50 border-cyan-500/80 ring-1 ring-cyan-500/30 text-white' : 'bg-slate-900/60 border-slate-800 hover:border-slate-700 text-slate-300'}"
              >
                <div class="flex items-center justify-between">
                  <span class="font-semibold text-xs flex items-center gap-1.5">
                    <Radio class="w-3.5 h-3.5 text-cyan-400" />
                    All Messages
                  </span>
                  {#if settings.scope === 'all'}
                    <Check class="w-3.5 h-3.5 text-cyan-400" />
                  {/if}
                </div>
                <span class="text-[10px] text-slate-400">
                  Every incoming LAN Broadcast & peer message
                </span>
              </button>
            </div>
          </div>
        {/if}
      </div>

      <!-- Modal Footer -->
      <div class="px-5 py-3.5 border-t border-slate-800 bg-slate-950/80 flex items-center justify-between flex-shrink-0">
        <button
          type="button"
          onclick={handleTriggerTest}
          class="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-750 text-cyan-300 border border-slate-700 rounded-lg text-xs font-mono transition-colors cursor-pointer"
        >
          <Sparkles class="w-3.5 h-3.5 text-cyan-400" />
          <span>{testSuccessToast ? 'Chimed & Sent!' : 'Test Notification'}</span>
        </button>

        <button
          type="button"
          onclick={onClose}
          class="px-4 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white font-semibold text-xs font-mono rounded-lg transition-colors cursor-pointer"
        >
          Done
        </button>
      </div>
    </div>
  </div>
{/if}
