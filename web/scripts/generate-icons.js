import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import sharp from 'sharp';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const iconsDir = path.resolve(__dirname, '../public/icons');
const screenshotsDir = path.resolve(__dirname, '../public/screenshots');

fs.mkdirSync(iconsDir, { recursive: true });
fs.mkdirSync(screenshotsDir, { recursive: true });

// DevDrop Lightning Path (viewBox 0 0 48 46)
const LIGHTNING_PATH = "M25.946 44.938c-.664.845-2.021.375-2.021-.698V33.937a2.26 2.26 0 0 0-2.262-2.262H10.287c-.92 0-1.456-1.04-.92-1.788l7.48-10.471c1.07-1.497 0-3.578-1.842-3.578H1.237c-.92 0-1.456-1.04-.92-1.788L10.013.474c.214-.297.556-.474.92-.474h28.894c.92 0 1.456 1.04.92 1.788l-7.48 10.471c-1.07 1.498 0 3.579 1.842 3.579h11.377c.943 0 1.473 1.088.89 1.83L25.947 44.94z";

function generateIconSvg({ size, isMaskable = false, isApple = false }) {
  const pad = isMaskable ? 0 : size * 0.04;
  const cardSize = size - pad * 2;
  const radius = isMaskable ? 0 : isApple ? size * 0.22 : size * 0.24;

  // Bolt scale:
  // For maskable, logo must fit comfortably within the 80% safe zone circle (diameter 0.8 * size)
  const boltTargetWidth = isMaskable ? size * 0.52 : size * 0.62;
  const scale = boltTargetWidth / 48;
  const boltW = 48 * scale;
  const boltH = 46 * scale;
  const tx = (size - boltW) / 2;
  const ty = (size - boltH) / 2 + (isMaskable ? 0 : size * 0.01);

  return `
<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <!-- Background Gradient -->
    <linearGradient id="bgGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#0b1329" />
      <stop offset="50%" stop-color="#020617" />
      <stop offset="100%" stop-color="#020617" />
    </linearGradient>

    <!-- Ambient Center Glow -->
    <radialGradient id="centerGlow" cx="50%" cy="45%" r="55%">
      <stop offset="0%" stop-color="#06b6d4" stop-opacity="0.32" />
      <stop offset="50%" stop-color="#8b5cf6" stop-opacity="0.18" />
      <stop offset="100%" stop-color="#020617" stop-opacity="0" />
    </radialGradient>

    <!-- Card Border Gradient -->
    <linearGradient id="borderGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#22d3ee" stop-opacity="0.8" />
      <stop offset="50%" stop-color="#818cf8" stop-opacity="0.3" />
      <stop offset="100%" stop-color="#c084fc" stop-opacity="0.8" />
    </linearGradient>

    <!-- Lightning Bolt Gradient -->
    <linearGradient id="boltGrad" x1="15%" y1="0%" x2="85%" y2="100%">
      <stop offset="0%" stop-color="#38bdf8" />
      <stop offset="25%" stop-color="#60a5fa" />
      <stop offset="60%" stop-color="#818cf8" />
      <stop offset="100%" stop-color="#c084fc" />
    </linearGradient>

    <!-- Glow Filter -->
    <filter id="boltGlow" x="-20%" y="-20%" width="140%" height="140%">
      <feGaussianBlur stdDeviation="${size * 0.02}" result="blur" />
      <feMerge>
        <feMergeNode in="blur" />
        <feMergeNode in="blur" />
        <feMergeNode in="SourceGraphic" />
      </feMerge>
    </filter>

    <!-- Outer Drop Shadow -->
    <filter id="cardShadow" x="-10%" y="-10%" width="120%" height="120%">
      <feDropShadow dx="0" dy="${size * 0.02}" stdDeviation="${size * 0.03}" flood-color="#000000" flood-opacity="0.6" />
    </filter>
  </defs>

  ${
    isMaskable
      ? `<rect width="${size}" height="${size}" fill="url(#bgGrad)" />
         <circle cx="${size / 2}" cy="${size / 2}" r="${size * 0.4}" fill="url(#centerGlow)" />`
      : `<rect x="${pad}" y="${pad}" width="${cardSize}" height="${cardSize}" rx="${radius}" fill="url(#bgGrad)" filter="url(#cardShadow)" />
         <rect x="${pad}" y="${pad}" width="${cardSize}" height="${cardSize}" rx="${radius}" fill="url(#centerGlow)" />
         <rect x="${pad + 1}" y="${pad + 1}" width="${cardSize - 2}" height="${cardSize - 2}" rx="${radius - 1}" fill="none" stroke="url(#borderGrad)" stroke-width="${Math.max(1.5, size * 0.008)}" opacity="0.85" />`
  }

  <!-- DevDrop Lightning Bolt Core -->
  <g transform="translate(${tx}, ${ty}) scale(${scale})">
    <!-- Ambient shadow under bolt -->
    <path d="${LIGHTNING_PATH}" fill="#000000" opacity="0.45" transform="translate(0, 1.5)" />
    <!-- Glowing bolt -->
    <path d="${LIGHTNING_PATH}" fill="url(#boltGrad)" filter="url(#boltGlow)" />
    <!-- Crisp bolt overlay -->
    <path d="${LIGHTNING_PATH}" fill="url(#boltGrad)" />
  </g>
</svg>
`;
}

// Generate PWA Screenshot SVG for rich preview in Chrome / Edge / mobile
function generateScreenshotSvg({ width, height, isMobile }) {
  return `
<svg width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="scBg" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#020617" />
      <stop offset="100%" stop-color="#090d1a" />
    </linearGradient>
    <linearGradient id="scBar" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stop-color="#06b6d4" />
      <stop offset="100%" stop-color="#8b5cf6" />
    </linearGradient>
  </defs>

  <!-- App Background -->
  <rect width="${width}" height="${height}" fill="url(#scBg)" />

  <!-- Top Navigation Header -->
  <rect width="${width}" height="64" fill="#0f172a" />
  <rect y="63" width="${width}" height="1" fill="#1e293b" />
  
  <!-- Logo & Title in Header -->
  <rect x="24" y="16" width="32" height="32" rx="8" fill="url(#scBar)" />
  <text x="68" y="37" fill="#f8fafc" font-family="Inter, sans-serif" font-size="16" font-weight="700">DevDrop</text>
  <text x="145" y="37" fill="#38bdf8" font-family="Inter, sans-serif" font-size="12" font-weight="600">LAN COLLAB</text>

  <!-- Status indicators -->
  <circle cx="${width - 120}" cy="32" r="5" fill="#34d399" />
  <text x="${width - 105}" y="36" fill="#94a3b8" font-family="monospace" font-size="12">5 Peers Online</text>

  ${
    isMobile
      ? `
  <!-- Mobile View Content -->
  <rect x="16" y="80" width="${width - 32}" height="70" rx="12" fill="#1e293b" />
  <text x="32" y="110" fill="#f1f5f9" font-family="sans-serif" font-size="15" font-weight="600">Broadcast Channel</text>
  <text x="32" y="132" fill="#94a3b8" font-family="monospace" font-size="12">Instant Zero-Config LAN messaging</text>

  <!-- Mobile Chat Bubble 1 -->
  <rect x="16" y="170" width="${width - 70}" height="80" rx="14" fill="#1e293b" />
  <text x="32" y="196" fill="#38bdf8" font-family="monospace" font-size="12" font-weight="600">alex.dev (192.168.1.42)</text>
  <text x="32" y="222" fill="#f8fafc" font-family="sans-serif" font-size="13">Sending over the latest migration script!</text>

  <!-- Mobile Code Card -->
  <rect x="16" y="270" width="${width - 32}" height="240" rx="14" fill="#090d16" stroke="#334155" stroke-width="1" />
  <rect x="16" y="270" width="${width - 32}" height="36" rx="14" fill="#1e293b" />
  <text x="32" y="293" fill="#38bdf8" font-family="monospace" font-size="12">schema.sql (PostgreSQL)</text>
  <text x="32" y="335" fill="#a78bfa" font-family="monospace" font-size="13">CREATE TABLE peers (</text>
  <text x="52" y="360" fill="#e2e8f0" font-family="monospace" font-size="13">  id UUID PRIMARY KEY,</text>
  <text x="52" y="385" fill="#e2e8f0" font-family="monospace" font-size="13">  display_name TEXT NOT NULL,</text>
  <text x="52" y="410" fill="#e2e8f0" font-family="monospace" font-size="13">  last_seen TIMESTAMP</text>
  <text x="32" y="435" fill="#a78bfa" font-family="monospace" font-size="13">);</text>

  <!-- File Drop Card -->
  <rect x="16" y="530" width="${width - 32}" height="90" rx="14" fill="#1e293b" />
  <text x="32" y="565" fill="#22d3ee" font-family="sans-serif" font-size="14" font-weight="600">📦 release-v2.0.0.tar.gz</text>
  <text x="32" y="590" fill="#94a3b8" font-family="monospace" font-size="12">48.2 MB • Streamed over Gigabit LAN</text>
  `
      : `
  <!-- Desktop Layout: Sidebar + Main Chat -->
  <!-- Sidebar -->
  <rect y="64" width="300" height="${height - 64}" fill="#0a0f1d" />
  <rect x="299" y="64" width="1" height="${height - 64}" fill="#1e293b" />

  <text x="24" y="100" fill="#64748b" font-family="monospace" font-size="11" font-weight="600">CONNECTED LAN PEERS</text>
  
  <!-- Peer 1 -->
  <rect x="16" y="115" width="268" height="50" rx="10" fill="#1e293b" />
  <circle cx="40" cy="140" r="14" fill="#0284c7" />
  <text x="36" y="145" fill="#fff" font-family="sans-serif" font-size="12" font-weight="700">A</text>
  <text x="64" y="137" fill="#f1f5f9" font-family="sans-serif" font-size="13" font-weight="600">Alex (Host)</text>
  <text x="64" y="152" fill="#10b981" font-family="monospace" font-size="10">192.168.1.10 • online</text>

  <!-- Peer 2 -->
  <rect x="16" y="175" width="268" height="50" rx="10" fill="#0f172a" />
  <circle cx="40" cy="200" r="14" fill="#7c3aed" />
  <text x="36" y="205" fill="#fff" font-family="sans-serif" font-size="12" font-weight="700">S</text>
  <text x="64" y="197" fill="#cbd5e1" font-family="sans-serif" font-size="13" font-weight="600">Sarah M.</text>
  <text x="64" y="212" fill="#10b981" font-family="monospace" font-size="10">192.168.1.15 • online</text>

  <!-- Main Chat Area -->
  <rect x="330" y="88" width="${width - 360}" height="70" rx="12" fill="#0f172a" stroke="#1e293b" stroke-width="1" />
  <text x="350" y="118" fill="#f8fafc" font-family="sans-serif" font-size="16" font-weight="600"># LAN Broadcast</text>
  <text x="350" y="140" fill="#94a3b8" font-family="sans-serif" font-size="12">Direct zero-hop peer file transfer and syntax code sharing</text>

  <!-- Message 1: Code snippet -->
  <rect x="330" y="175" width="${width - 400}" height="280" rx="12" fill="#0a0f1d" stroke="#334155" stroke-width="1" />
  <rect x="330" y="175" width="${width - 400}" height="38" rx="12" fill="#1e293b" />
  <text x="350" y="199" fill="#38bdf8" font-family="monospace" font-size="13">main.go • Go 1.22</text>
  <text x="350" y="245" fill="#f43f5e" font-family="monospace" font-size="14">package <tspan fill="#f8fafc">main</tspan></text>
  <text x="350" y="275" fill="#f43f5e" font-family="monospace" font-size="14">func <tspan fill="#38bdf8">StartLanNode</tspan>() error {</text>
  <text x="380" y="305" fill="#e2e8f0" font-family="monospace" font-size="14">  hub := hub.NewHub()</text>
  <text x="380" y="335" fill="#e2e8f0" font-family="monospace" font-size="14">  return hub.ListenAndServe(":8080")</text>
  <text x="350" y="365" fill="#f43f5e" font-family="monospace" font-size="14">}</text>

  <!-- Message 2: File Transfer -->
  <rect x="330" y="475" width="460" height="90" rx="12" fill="#1e293b" stroke="#0284c7" stroke-width="1" />
  <text x="355" y="515" fill="#38bdf8" font-family="sans-serif" font-size="16" font-weight="600">📁 frontend-build.zip (38.4 MB)</text>
  <text x="355" y="540" fill="#94a3b8" font-family="monospace" font-size="12">P2P LAN Transfer Completed • 105 MB/s</text>
  `
  }
</svg>
`;
}

async function buildAllAssets() {
  console.log('Rendering 512x512 standard icon...');
  const svg512 = generateIconSvg({ size: 512, isMaskable: false });
  await sharp(Buffer.from(svg512)).png().toFile(path.join(iconsDir, 'icon-512.png'));

  console.log('Rendering 192x192 standard icon...');
  const svg192 = generateIconSvg({ size: 192, isMaskable: false });
  await sharp(Buffer.from(svg192)).png().toFile(path.join(iconsDir, 'icon-192.png'));

  console.log('Rendering 512x512 maskable icon...');
  const svgMaskable512 = generateIconSvg({ size: 512, isMaskable: true });
  await sharp(Buffer.from(svgMaskable512)).png().toFile(path.join(iconsDir, 'icon-maskable-512.png'));

  console.log('Rendering 192x192 maskable icon...');
  const svgMaskable192 = generateIconSvg({ size: 192, isMaskable: true });
  await sharp(Buffer.from(svgMaskable192)).png().toFile(path.join(iconsDir, 'icon-maskable-192.png'));

  console.log('Rendering 180x180 Apple touch icon...');
  const svgApple = generateIconSvg({ size: 180, isMaskable: false, isApple: true });
  await sharp(Buffer.from(svgApple)).png().toFile(path.join(iconsDir, 'apple-touch-icon.png'));

  console.log('Rendering favicons (32x32, 16x16)...');
  await sharp(Buffer.from(svg512)).resize(32, 32).png().toFile(path.join(iconsDir, 'favicon-32.png'));
  await sharp(Buffer.from(svg512)).resize(16, 16).png().toFile(path.join(iconsDir, 'favicon-16.png'));

  // Also write out standard SVG icon
  fs.writeFileSync(path.join(iconsDir, 'icon.svg'), svg512);

  console.log('Rendering desktop and mobile PWA screenshots...');
  const deskSvg = generateScreenshotSvg({ width: 1280, height: 720, isMobile: false });
  await sharp(Buffer.from(deskSvg)).png().toFile(path.join(screenshotsDir, 'desktop.png'));

  const mobSvg = generateScreenshotSvg({ width: 750, height: 1334, isMobile: true });
  await sharp(Buffer.from(mobSvg)).png().toFile(path.join(screenshotsDir, 'mobile.png'));

  // Clean up test file if present
  try {
    fs.unlinkSync(path.join(iconsDir, 'test_favicon.png'));
  } catch {}

  console.log('✅ Successfully generated all PWA icons and screenshots!');
}

buildAllAssets().catch((err) => {
  console.error('Failed to generate assets:', err);
  process.exit(1);
});
