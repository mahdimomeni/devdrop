// DevDrop PWA Service
// Manages Service Worker lifecycle, installation prompts, platform detection, and updates

let deferredPrompt = null;
let updateWaitingWorker = null;

const listeners = new Set();

export const pwaState = {
  isSupported: typeof window !== 'undefined' && 'serviceWorker' in navigator,
  isSecureContext: typeof window !== 'undefined' ? (window.isSecureContext ?? true) : true,
  isInstalled: false,
  isInstallable: false,
  hasUpdate: false,
  isIOS: false,
  isAndroid: false,
  isDesktop: false,
  platform: 'desktop', // 'ios' | 'android' | 'desktop'
  origin: typeof window !== 'undefined' ? window.location.origin : '',
};

function notifyListeners() {
  for (const listener of listeners) {
    try {
      listener({ ...pwaState });
    } catch (e) {
      console.warn('[PWA] Listener callback error:', e);
    }
  }
}

export function subscribePWA(callback) {
  listeners.add(callback);
  callback({ ...pwaState });
  return () => {
    listeners.delete(callback);
  };
}

/**
 * Check whether the app is currently running as a standalone PWA
 */
export function checkIsStandalone() {
  if (typeof window === 'undefined') return false;
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    window.matchMedia('(display-mode: window-controls-overlay)').matches ||
    window.navigator.standalone === true ||
    document.referrer.includes('android-app://')
  );
}

/**
 * Detect client operating platform
 */
export function detectPlatform() {
  if (typeof window === 'undefined') return 'desktop';
  const ua = window.navigator.userAgent.toLowerCase();
  if (/iphone|ipad|ipod/.test(ua) || (window.navigator.platform === 'MacIntel' && window.navigator.maxTouchPoints > 1)) {
    return 'ios';
  }
  if (/android/.test(ua)) {
    return 'android';
  }
  return 'desktop';
}

/**
 * Initialize Service Worker and PWA Install hooks
 */
export function initPWA() {
  if (typeof window === 'undefined') return;

  const platform = detectPlatform();
  pwaState.platform = platform;
  pwaState.isIOS = platform === 'ios';
  pwaState.isAndroid = platform === 'android';
  pwaState.isDesktop = platform === 'desktop';
  pwaState.isInstalled = checkIsStandalone();
  pwaState.isSecureContext = window.isSecureContext ?? true;
  pwaState.origin = window.location.origin || '';

  // On iOS Safari, beforeinstallprompt never fires; if not standalone, it can be added to home screen
  if (pwaState.isIOS && !pwaState.isInstalled) {
    pwaState.isInstallable = true;
  }

  // 1. Listen for browser install prompt (Chrome, Edge, Android, Opera)
  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault();
    deferredPrompt = e;
    pwaState.isInstallable = true;
    notifyListeners();
    console.log('[PWA] DevDrop install prompt ready');
  });

  // 2. Listen for successful installation
  window.addEventListener('appinstalled', () => {
    deferredPrompt = null;
    pwaState.isInstalled = true;
    pwaState.isInstallable = false;
    notifyListeners();
    console.log('[PWA] DevDrop successfully installed on device!');
  });

  // 3. Register Service Worker
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      navigator.serviceWorker
        .register('/sw.js', { scope: '/' })
        .then((registration) => {
          console.log('[PWA] Service Worker registered with scope:', registration.scope);

          // Check if an updated worker is already waiting
          if (registration.waiting) {
            updateWaitingWorker = registration.waiting;
            pwaState.hasUpdate = true;
            notifyListeners();
          }

          // Listen for new worker installed
          registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing;
            if (!newWorker) return;

            newWorker.addEventListener('statechange', () => {
              if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                // New update available!
                updateWaitingWorker = newWorker;
                pwaState.hasUpdate = true;
                notifyListeners();
                console.log('[PWA] New DevDrop version available');
              }
            });
          });
        })
        .catch((err) => {
          console.warn('[PWA] Service Worker registration failed:', err);
        });

      // Reload window when new service worker takes control
      let refreshing = false;
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        if (!refreshing) {
          refreshing = true;
          window.location.reload();
        }
      });
    });
  }

  notifyListeners();
}

/**
 * Trigger the native installation prompt
 * @returns {Promise<{ installed: boolean, outcome: string }>}
 */
export async function promptInstall() {
  if (pwaState.isInstalled) {
    return { installed: true, outcome: 'already_installed' };
  }

  if (deferredPrompt) {
    try {
      deferredPrompt.prompt();
      const choiceResult = await deferredPrompt.userChoice;
      console.log('[PWA] User choice:', choiceResult.outcome);
      if (choiceResult.outcome === 'accepted') {
        pwaState.isInstalled = true;
        pwaState.isInstallable = false;
        deferredPrompt = null;
        notifyListeners();
        return { installed: true, outcome: 'accepted' };
      } else {
        return { installed: false, outcome: 'dismissed' };
      }
    } catch (err) {
      console.warn('[PWA] prompt error:', err);
      return { installed: false, outcome: 'error' };
    }
  }

  // If no deferred prompt (e.g. iOS or already installed or unsupported browser), indicate manual instructions needed
  return { installed: false, outcome: 'manual_instructions' };
}

/**
 * Apply pending Service Worker update
 */
export function applyUpdate() {
  if (updateWaitingWorker) {
    updateWaitingWorker.postMessage({ type: 'SKIP_WAITING' });
  } else {
    window.location.reload();
  }
}
