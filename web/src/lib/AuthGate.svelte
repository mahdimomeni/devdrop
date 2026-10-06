<script>
  import {
    loginWithPassword,
    detectDeviceName
  } from './api.js';
  import {
    Shield,
    ShieldCheck,
    Lock,
    Eye,
    EyeOff,
    Loader2,
    AlertCircle,
    Laptop
  } from 'lucide-svelte';

  let {
    onAuthenticated
  } = $props();

  let password = $state('');
  let showPassword = $state(false);
  let trustDevice = $state(true);
  let deviceName = $state(detectDeviceName());
  let isLoading = $state(false);
  let errorMessage = $state('');
  let isEditingDeviceName = $state(false);

  async function handleSubmit(e) {
    if (e) e.preventDefault();
    errorMessage = '';

    if (!password.trim()) {
      errorMessage = 'Please enter the access password.';
      return;
    }

    isLoading = true;
    try {
      await loginWithPassword(password, trustDevice, deviceName);
      if (onAuthenticated) {
        onAuthenticated();
      }
    } catch (err) {
      errorMessage = err.message || 'Incorrect password';
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 bg-slate-950/90 backdrop-blur-xl overflow-y-auto">
  <!-- Glowing Background Orbs -->
  <div class="fixed -top-32 -left-32 w-96 h-96 bg-cyan-600/15 rounded-full blur-3xl pointer-events-none"></div>
  <div class="fixed -bottom-32 -right-32 w-96 h-96 bg-indigo-600/15 rounded-full blur-3xl pointer-events-none"></div>

  <div class="relative w-full max-w-md bg-slate-900/90 border border-slate-800 rounded-3xl shadow-2xl shadow-cyan-950/30 p-6 sm:p-8 flex flex-col gap-6 backdrop-blur-2xl animate-in fade-in zoom-in-95 duration-200">
    
    <!-- Top Branding & Icon -->
    <div class="flex flex-col items-center text-center gap-3">
      <div class="relative flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-tr from-cyan-500/20 via-indigo-500/20 to-cyan-500/10 border border-cyan-500/30 text-cyan-400 shadow-inner">
        <Lock class="w-8 h-8 text-cyan-400" />
        <div class="absolute -bottom-1 -right-1 w-6 h-6 rounded-full bg-slate-900 border border-slate-700 flex items-center justify-center text-cyan-400 shadow">
          <ShieldCheck class="w-3.5 h-3.5" />
        </div>
      </div>

      <div class="flex flex-col gap-1">
        <div class="flex items-center justify-center gap-2">
          <span class="text-xs font-mono font-bold tracking-widest uppercase text-cyan-400 bg-cyan-950/80 px-2.5 py-0.5 rounded-full border border-cyan-800/60">
            DevDrop LAN
          </span>
          <span class="text-[11px] font-mono text-slate-400 flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
            Protected
          </span>
        </div>
        <h1 class="text-2xl font-bold text-slate-100 tracking-tight mt-1">
          Enter Password to Join
        </h1>
        <p class="text-xs text-slate-400 max-w-xs mx-auto leading-relaxed">
          This DevDrop node is protected. Enter the access password once to trust this device and gain instant access.
        </p>
      </div>
    </div>

    <!-- Error Alert Banner -->
    {#if errorMessage}
      <div class="flex items-start gap-2.5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs animate-in fade-in duration-150">
        <AlertCircle class="w-4 h-4 flex-shrink-0 mt-0.5 text-rose-400" />
        <span class="flex-1 leading-normal">{errorMessage}</span>
      </div>
    {/if}

    <!-- Form -->
    <form onsubmit={handleSubmit} class="flex flex-col gap-4">
      <!-- Password Field -->
      <div class="flex flex-col gap-1.5">
        <label for="password-input" class="text-xs font-medium text-slate-300 flex items-center justify-between">
          <span>Access Password</span>
        </label>
        <div class="relative">
          <input
            id="password-input"
            name="password"
            type={showPassword ? 'text' : 'password'}
            autocomplete="current-password"
            enterkeyhint="done"
            required
            placeholder="Enter password..."
            bind:value={password}
            disabled={isLoading}
            class="w-full bg-slate-950/80 border border-slate-700/80 rounded-xl px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all pr-11 font-mono tracking-wide"
          />
          <button
            type="button"
            onclick={() => (showPassword = !showPassword)}
            tabindex="-1"
            class="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 text-slate-400 hover:text-slate-200 transition-colors rounded-lg cursor-pointer"
            title={showPassword ? 'Hide password' : 'Show password'}
          >
            {#if showPassword}
              <EyeOff class="w-4 h-4" />
            {:else}
              <Eye class="w-4 h-4" />
            {/if}
          </button>
        </div>
      </div>

      <!-- Device Trust & Identification Card -->
      <div class="bg-slate-950/60 border border-slate-800/80 rounded-2xl p-3.5 flex flex-col gap-3">
        <label class="flex items-start gap-3 cursor-pointer select-none">
          <input
            type="checkbox"
            bind:checked={trustDevice}
            disabled={isLoading}
            class="mt-1 w-4 h-4 rounded border-slate-700 bg-slate-900 text-cyan-500 focus:ring-cyan-500/30 accent-cyan-500 cursor-pointer"
          />
          <div class="flex flex-col gap-0.5">
            <span class="text-xs font-semibold text-slate-200 flex items-center gap-1.5">
              Trust this device
              <span class="text-[10px] bg-cyan-950 text-cyan-400 font-normal px-1.5 py-0.2 rounded border border-cyan-800/40">
                Recommended
              </span>
            </span>
            <span class="text-[11px] text-slate-400 leading-normal">
              Stay verified on this browser so you won't need to enter the password on future visits.
            </span>
          </div>
        </label>

        <!-- Device Name identification -->
        <div class="flex items-center justify-between gap-2 pt-2 border-t border-slate-800/60 text-xs">
          <div class="flex items-center gap-2 text-slate-400 min-w-0">
            <Laptop class="w-3.5 h-3.5 text-cyan-400 flex-shrink-0" />
            <span class="text-slate-400 font-mono text-[11px] flex-shrink-0">Device:</span>
            {#if isEditingDeviceName}
              <input
                type="text"
                bind:value={deviceName}
                maxlength="50"
                class="bg-slate-900 border border-slate-700 rounded px-2 py-0.5 text-slate-200 text-xs focus:outline-none focus:border-cyan-500 min-w-0"
                onblur={() => (isEditingDeviceName = false)}
                onkeydown={(e) => e.key === 'Enter' && (isEditingDeviceName = false)}
              />
            {:else}
              <span class="text-slate-200 font-mono text-[11px] truncate">{deviceName}</span>
            {/if}
          </div>
          {#if !isEditingDeviceName}
            <button
              type="button"
              onclick={() => (isEditingDeviceName = true)}
              class="text-[11px] text-cyan-400 hover:text-cyan-300 font-medium underline-offset-2 hover:underline flex-shrink-0 cursor-pointer"
            >
              Rename
            </button>
          {/if}
        </div>
      </div>

      <!-- Submit Button -->
      <button
        type="submit"
        disabled={isLoading}
        class="mt-2 w-full flex items-center justify-center gap-2 py-3 px-4 rounded-xl font-medium text-sm text-slate-950 bg-gradient-to-r from-cyan-400 via-cyan-300 to-indigo-400 hover:from-cyan-300 hover:to-indigo-300 active:scale-[0.99] transition-all duration-150 shadow-lg shadow-cyan-500/20 disabled:opacity-50 disabled:cursor-not-allowed font-semibold cursor-pointer"
      >
        {#if isLoading}
          <Loader2 class="w-4 h-4 animate-spin text-slate-950" />
          <span>Verifying...</span>
        {:else}
          <Shield class="w-4 h-4 text-slate-950" />
          <span>Unlock & Trust Device</span>
        {/if}
      </button>
    </form>

    <!-- Footer Security Note -->
    <div class="text-center text-[11px] text-slate-500 font-mono">
      <span>Password managed via server CLI / ENV</span>
    </div>

  </div>
</div>
