// Theme & Appearance Service for DevDrop
// Supports multiple base themes (Dark Slate, Midnight OLED, Nord Arctic, Dracula Cosmic, Cyber Matrix, Solar Warm, Studio Light)
// and multiple vibrant accent colors (Cyan, Emerald, Violet, Amber, Rose, Sky, Lime, Orange).

export const THEME_SETTINGS_KEY = 'devdrop_theme_settings';

export const BASE_THEMES = [
  {
    id: 'dark',
    name: 'Slate Dark',
    subtitle: 'Classic DevDrop',
    description: 'High-tech dark slate aesthetic optimized for all displays',
    type: 'dark',
    badge: 'Default',
    bg: '#0b0f19',
    surface: '#0f172a',
    card: '#1e293b',
    border: '#334155',
    text: '#f8fafc',
    metaThemeColor: '#0b0f19',
  },
  {
    id: 'midnight',
    name: 'Midnight OLED',
    subtitle: 'Pitch Black',
    description: 'Pure #000000 black surface with ultra-high contrast for OLED displays',
    type: 'dark',
    badge: 'OLED',
    bg: '#000000',
    surface: '#070709',
    card: '#121316',
    border: '#22252c',
    text: '#ffffff',
    metaThemeColor: '#000000',
  },
  {
    id: 'nord',
    name: 'Nordic Arctic',
    subtitle: 'Frost & Polar Night',
    description: 'Arctic blue-grey palette inspired by cold Nordic landscapes',
    type: 'dark',
    badge: 'Popular',
    bg: '#1d212a',
    surface: '#242933',
    card: '#2e3440',
    border: '#3b4252',
    text: '#eceff4',
    metaThemeColor: '#1d212a',
  },
  {
    id: 'dracula',
    name: 'Dracula Cosmic',
    subtitle: 'Midnight Violet',
    description: 'Deep cosmic purple-tinted dark surface with elegant night ambiance',
    type: 'dark',
    badge: 'Vibrant',
    bg: '#12111b',
    surface: '#191726',
    card: '#221f35',
    border: '#332f50',
    text: '#f5f3fd',
    metaThemeColor: '#12111b',
  },
  {
    id: 'cyberpunk',
    name: 'Cyber Matrix',
    subtitle: 'Terminal Phosphor',
    description: 'Ultra-dark hacker terminal with stealth emerald undertones',
    type: 'dark',
    badge: 'Hacker',
    bg: '#060b08',
    surface: '#0d1611',
    card: '#13221a',
    border: '#1e3629',
    text: '#eafaf0',
    metaThemeColor: '#060b08',
  },
  {
    id: 'warm',
    name: 'Solar Warm',
    subtitle: 'Comfort Sepia',
    description: 'Warm charcoal & paper tones designed for low blue-light eye comfort',
    type: 'dark',
    badge: 'Comfort',
    bg: '#151311',
    surface: '#1c1917',
    card: '#292524',
    border: '#44403c',
    text: '#fafaf9',
    metaThemeColor: '#151311',
  },
  {
    id: 'light',
    name: 'Studio Light',
    subtitle: 'Clean & Crisp',
    description: 'High-clarity studio light mode with refined borders and dark typography',
    type: 'light',
    badge: 'Light',
    bg: '#f8fafc',
    surface: '#ffffff',
    card: '#f1f5f9',
    border: '#cbd5e1',
    text: '#0f172a',
    metaThemeColor: '#ffffff',
  },
];

export const ACCENT_COLORS = [
  {
    id: 'cyan',
    name: 'Electric Cyan',
    label: 'Cyan',
    hex: '#06b6d4',
    lightText: '#0e7490',
    previewHex: '#06b6d4',
    glow: 'rgba(6, 182, 212, 0.45)',
    ringClass: 'ring-cyan-400',
    bgClass: 'bg-cyan-500',
  },
  {
    id: 'emerald',
    name: 'Cyber Emerald',
    label: 'Emerald',
    hex: '#10b981',
    lightText: '#047857',
    previewHex: '#10b981',
    glow: 'rgba(16, 185, 129, 0.45)',
    ringClass: 'ring-emerald-400',
    bgClass: 'bg-emerald-500',
  },
  {
    id: 'violet',
    name: 'Royal Violet',
    label: 'Violet',
    hex: '#8b5cf6',
    lightText: '#6d28d9',
    previewHex: '#8b5cf6',
    glow: 'rgba(139, 92, 246, 0.45)',
    ringClass: 'ring-purple-400',
    bgClass: 'bg-purple-500',
  },
  {
    id: 'amber',
    name: 'Neon Amber',
    label: 'Amber',
    hex: '#f59e0b',
    lightText: '#b45309',
    previewHex: '#f59e0b',
    glow: 'rgba(245, 158, 11, 0.45)',
    ringClass: 'ring-amber-400',
    bgClass: 'bg-amber-500',
  },
  {
    id: 'rose',
    name: 'Crimson Rose',
    label: 'Rose',
    hex: '#f43f5e',
    lightText: '#be123c',
    previewHex: '#f43f5e',
    glow: 'rgba(244, 63, 94, 0.45)',
    ringClass: 'ring-rose-400',
    bgClass: 'bg-rose-500',
  },
  {
    id: 'sky',
    name: 'Ocean Sky',
    label: 'Sky',
    hex: '#0284c7',
    lightText: '#0369a1',
    previewHex: '#38bdf8',
    glow: 'rgba(14, 165, 233, 0.45)',
    ringClass: 'ring-sky-400',
    bgClass: 'bg-sky-500',
  },
  {
    id: 'lime',
    name: 'Hacker Lime',
    label: 'Lime',
    hex: '#84cc16',
    lightText: '#4d7c0f',
    previewHex: '#84cc16',
    glow: 'rgba(132, 204, 22, 0.45)',
    ringClass: 'ring-lime-400',
    bgClass: 'bg-lime-500',
  },
  {
    id: 'orange',
    name: 'Blaze Orange',
    label: 'Orange',
    hex: '#f97316',
    lightText: '#c2410c',
    previewHex: '#f97316',
    glow: 'rgba(249, 115, 22, 0.45)',
    ringClass: 'ring-orange-400',
    bgClass: 'bg-orange-500',
  },
];

export const THEME_PRESETS = [
  {
    id: 'classic',
    name: 'DevDrop Classic',
    baseTheme: 'dark',
    accentColor: 'cyan',
    icon: 'Radio',
    tagline: 'Default cyber tech look',
  },
  {
    id: 'oled_phantom',
    name: 'OLED Phantom',
    baseTheme: 'midnight',
    accentColor: 'violet',
    icon: 'Moon',
    tagline: 'True black & royal violet',
  },
  {
    id: 'matrix_hacker',
    name: 'Matrix Terminal',
    baseTheme: 'cyberpunk',
    accentColor: 'lime',
    icon: 'Terminal',
    tagline: 'Stealth hacker green',
  },
  {
    id: 'nordic_frost',
    name: 'Nordic Frost',
    baseTheme: 'nord',
    accentColor: 'sky',
    icon: 'Sparkles',
    tagline: 'Arctic blue calm',
  },
  {
    id: 'dracula_rose',
    name: 'Cosmic Magenta',
    baseTheme: 'dracula',
    accentColor: 'rose',
    icon: 'Sparkles',
    tagline: 'Deep purple & rose glow',
  },
  {
    id: 'studio_clean',
    name: 'Clean Studio',
    baseTheme: 'light',
    accentColor: 'sky',
    icon: 'Sun',
    tagline: 'Crisp light mode aesthetic',
  },
  {
    id: 'warm_sunset',
    name: 'Warm Sunset',
    baseTheme: 'warm',
    accentColor: 'amber',
    icon: 'Flame',
    tagline: 'Warm sepia & amber glow',
  },
  {
    id: 'stealth_emerald',
    name: 'Stealth Emerald',
    baseTheme: 'midnight',
    accentColor: 'emerald',
    icon: 'Shield',
    tagline: 'Pure black & vivid emerald',
  },
];

/**
 * Default theme configuration
 */
export function getDefaultThemeSettings() {
  return {
    baseTheme: 'dark',
    accentColor: 'cyan',
  };
}

/**
 * Load saved theme settings from localStorage
 */
export function loadThemeSettings() {
  if (typeof localStorage === 'undefined') return getDefaultThemeSettings();
  try {
    const raw = localStorage.getItem(THEME_SETTINGS_KEY);
    if (!raw) return getDefaultThemeSettings();
    const parsed = JSON.parse(raw);
    return {
      baseTheme: parsed.baseTheme || 'dark',
      accentColor: parsed.accentColor || 'cyan',
    };
  } catch (err) {
    console.warn('Failed to load theme settings:', err);
    return getDefaultThemeSettings();
  }
}

/**
 * Save theme settings to localStorage
 */
export function saveThemeSettings(settings) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(THEME_SETTINGS_KEY, JSON.stringify(settings));
  } catch (err) {
    console.error('Failed to save theme settings:', err);
  }
}

/**
 * Apply theme to document DOM elements, data-attributes, and meta tags
 */
export function applyTheme(settings) {
  if (typeof document === 'undefined') return;

  const baseTheme = settings?.baseTheme || 'dark';
  const accentColor = settings?.accentColor || 'cyan';

  const root = document.documentElement;

  // 1. Set data attributes
  root.setAttribute('data-theme', baseTheme);
  root.setAttribute('data-accent', accentColor);

  // 2. Set color-scheme
  const isLight = baseTheme === 'light';
  root.style.colorScheme = isLight ? 'light' : 'dark';

  if (isLight) {
    root.classList.remove('dark');
    root.classList.add('light');
  } else {
    root.classList.remove('light');
    root.classList.add('dark');
  }

  // 3. Update theme-color meta tag for mobile browsers and PWA titlebars
  const activeBase = BASE_THEMES.find((t) => t.id === baseTheme) || BASE_THEMES[0];
  const metaThemeColor = document.querySelector('meta[name="theme-color"]');
  if (metaThemeColor && activeBase?.metaThemeColor) {
    metaThemeColor.setAttribute('content', activeBase.metaThemeColor);
  }

  // 4. Update apple-mobile-web-app-status-bar-style
  const appleStatusMeta = document.querySelector('meta[name="apple-mobile-web-app-status-bar-style"]');
  if (appleStatusMeta) {
    appleStatusMeta.setAttribute('content', isLight ? 'default' : 'black-translucent');
  }
}

/**
 * Initialize theme immediately at script startup
 */
export function initTheme() {
  const current = loadThemeSettings();
  applyTheme(current);
  return current;
}
