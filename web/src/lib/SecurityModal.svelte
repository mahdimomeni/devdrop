<script>
  import {
    logoutDevice,
    getAuthDevices,
    detectDeviceName
  } from './api.js';
  import {
    ShieldCheck,
    Laptop,
    LogOut,
    X,
    Loader2,
    Users,
    Info,
    Terminal
  } from 'lucide-svelte';

  let {
    isOpen = false,
    currentUser = null,
    onClose,
    onLocked,
    onNotify
  } = $props();

  let devicesData = $state(null);
  let isLoadingDevices = $state(false);
  let isLocking = $state(false);

  async function loadDevices() {
    if (!isOpen) return;
    isLoadingDevices = true;
    try {
      devicesData = await getAuthDevices();
    } catch (err) {
      console.warn('Failed to fetch auth devices info:', err);
    } finally {
      isLoadingDevices = false;
    }
  }

  $effect(() => {
    if (isOpen) {
      loadDevices();
    }
  });

  async function handleLockDevice() {
    if (isLocking) return;
    isLocking = true;
    try {
      await logoutDevice();
      if (onLocked) onLocked();
    } catch (err) {
      if (onNotify) onNotify('Failed to lock device: ' + err.message, 'error');
    } finally {
      isLocking = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-950/80 backdrop-blur-md overflow-y-auto">
    <div
      class="relative w-full max-w-md bg-slate-900 border border-slate-800 rounded-3xl shadow-2xl p-5 sm:p-6 flex flex-col gap-5 text-slate-100 max-h-[90vh] overflow-y-auto no-scrollbar animate-in fade-in zoom-in-95 duration-200"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-slate-800/80 pb-3.5">
        <div class="flex items-center gap-3">
          <div class="p-2.5 rounded-xl bg-cyan-500/15 border border-cyan-500/25 text-cyan-400">
            <ShieldCheck class="w-6 h-6" />
          </div>
          <div>
            <h2 class="text-base font-bold tracking-tight text-slate-100">
              Security & Trusted Devices
            </h2>
            <p class="text-xs text-slate-400">Device trust and LAN node security</p>
          </div>
        </div>

        <button
          onclick={onClose}
          class="p-2 text-slate-400 hover:text-slate-200 rounded-xl hover:bg-slate-800 transition-colors cursor-pointer"
          title="Close"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Current Device Status Box -->
      <div class="bg-slate-950/70 border border-emerald-500/30 rounded-2xl p-4 flex flex-col gap-3 shadow-inner">
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-start gap-3">
            <div class="p-2 rounded-xl bg-emerald-500/15 text-emerald-400 mt-0.5">
              <Laptop class="w-5 h-5" />
            </div>
            <div class="flex flex-col gap-0.5">
              <div class="flex items-center gap-2">
                <span class="text-xs font-semibold text-slate-200">
                  {devicesData?.current_device?.device_name || detectDeviceName()}
                </span>
                <span class="inline-flex items-center gap-1 text-[10px] font-mono text-emerald-400 bg-emerald-950/80 px-2 py-0.5 rounded-full border border-emerald-800/60 font-medium">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                  Trusted
                </span>
              </div>
              <span class="text-[11px] text-slate-400 font-mono">
                IP: {devicesData?.current_device?.ip_address || currentUser?.ip_address || 'Connected LAN'}
              </span>
            </div>
          </div>
        </div>

        <!-- Lock device button -->
        <div class="pt-2 border-t border-slate-800/60 flex items-center justify-between">
          <span class="text-[11px] text-slate-400">Revoke trust on this browser:</span>
          <button
            type="button"
            onclick={handleLockDevice}
            disabled={isLocking}
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-800/80 hover:bg-rose-950/60 hover:text-rose-300 hover:border-rose-800/60 border border-slate-700/80 text-xs font-medium text-slate-300 transition-all cursor-pointer disabled:opacity-50"
            title="Lock this device so it requires password on next visit"
          >
            {#if isLocking}
              <Loader2 class="w-3.5 h-3.5 animate-spin" />
              <span>Locking...</span>
            {:else}
              <LogOut class="w-3.5 h-3.5 text-rose-400" />
              <span>Lock Device</span>
            {/if}
          </button>
        </div>
      </div>

      <!-- Total Trusted Devices Stats -->
      {#if devicesData?.total_trusted_devices !== undefined}
        <div class="flex items-center justify-between px-3.5 py-2.5 rounded-xl bg-slate-950/40 border border-slate-800/60 text-xs text-slate-400">
          <span class="flex items-center gap-2">
            <Users class="w-4 h-4 text-cyan-400" />
            Total trusted devices on this LAN node:
          </span>
          <span class="font-mono font-bold text-cyan-300 bg-cyan-950/60 px-2 py-0.5 rounded border border-cyan-800/40">
            {devicesData.total_trusted_devices}
          </span>
        </div>
      {/if}

      <!-- CLI / ENV Password Information Box -->
      <div class="bg-slate-950/60 border border-slate-800 rounded-2xl p-3.5 flex flex-col gap-2">
        <div class="flex items-center gap-2 text-cyan-400 text-xs font-semibold">
          <Terminal class="w-4 h-4 text-cyan-400" />
          <span>Password Management</span>
        </div>
        <p class="text-[11px] text-slate-400 leading-relaxed">
          For network security, the workspace password cannot be modified from the browser. It is controlled exclusively on the server host via CLI or environment variable:
        </p>
        <div class="bg-black/60 rounded-lg p-2 font-mono text-[10px] text-slate-300 border border-slate-800/80 space-y-1">
          <div><span class="text-cyan-400">./devdrop</span> -password "your-new-secret"</div>
          <div><span class="text-cyan-400">DEVDROP_PASSWORD</span>="your-new-secret" ./devdrop</div>
        </div>
        <p class="text-[10px] text-slate-500 italic">
          * Changing the server password automatically revokes previous device trusts, requiring devices to enter the new password once.
        </p>
      </div>

    </div>
  </div>
{/if}
