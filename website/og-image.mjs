// Social preview images (Open Graph / X cards), one per page and language, rendered at build time.
//
// 1200x630 (1.91:1): the size Facebook, LinkedIn, WhatsApp, Slack, Discord and Telegram expect;
// X crops a 2:1 strip from the middle, so nothing important sits near the top or bottom edge.
// Monochrome like the brand: black on white, Parmigiano Text, the wordmark from static/assets.
//
// The fonts in the repo are woff2 only (license), so they are decompressed in memory for resvg.
// Rendered PNGs are cached by content hash in node_modules/.cache, so live rebuilds stay fast.
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Resvg } from '@resvg/resvg-js';
import opentype from 'opentype.js';
import wawoff2 from 'wawoff2';

const ROOT = dirname(fileURLToPath(import.meta.url));
const CACHE = join(ROOT, 'node_modules/.cache/mestiere-og');
export const OG_WIDTH = 1200;
export const OG_HEIGHT = 630;

const FAMILY = 'Parmigiano Text L';
// One at a time: wawoff2 returns views on its WebAssembly memory, so each result is copied out
// before the next call can reuse that memory.
const fontBuffers = [];
for (const name of ['light', 'light-italic', 'bold']) {
  fontBuffers.push(Buffer.from(await wawoff2.decompress(readFileSync(join(ROOT, `static/fonts/parmigiano-text-${name}.woff2`)))));
}
const resvgOptions = { font: { fontBuffers, loadSystemFonts: false, defaultFontFamily: FAMILY } };

// Wordmark paths (viewBox 427x91), reused as a nested <svg>
const LOGO = readFileSync(join(ROOT, 'static/assets/logo-black.svg'), 'utf8');
const LOGO_VIEWBOX = LOGO.match(/viewBox="([^"]+)"/)[1];
const LOGO_INNER = LOGO.replace(/^[\s\S]*?<svg[^>]*>/, '').replace(/<\/svg>\s*$/, '');
const LOGO_RATIO = 91 / 427;
const wordmark = (x, y, width) => `<svg x="${x}" y="${y}" width="${width}" height="${(width * LOGO_RATIO).toFixed(2)}" viewBox="${LOGO_VIEWBOX}">${LOGO_INNER}</svg>`;

const xml = (s) => String(s).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
const textAttrs = ({ size, weight = 300, italic = false }) => `font-family="${FAMILY}" font-weight="${weight}"${italic ? ' font-style="italic"' : ''} font-size="${size}" fill="#000"`;

// Width of one line of text with the real font metrics (advances and kerning), for word wrap.
// Same order as fontBuffers: light, light italic, bold.
const metrics = fontBuffers.map((buf) => opentype.parse(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength)));
const measure = (text, { size, weight = 300, italic = false }) => (
  metrics[weight >= 600 ? 2 : italic ? 1 : 0].getAdvanceWidth(text, size, { kerning: true })
);

// Greedy word wrap to maxWidth; the last allowed line ends with an ellipsis if text is left.
function wrap(text, style, maxWidth, maxLines) {
  const lines = [];
  let line = '';
  const words = text.split(/\s+/);
  for (let i = 0; i < words.length; i++) {
    const candidate = line ? `${line} ${words[i]}` : words[i];
    if (!line || measure(candidate, style) <= maxWidth) { line = candidate; continue; }
    lines.push(line);
    line = words[i];
    if (lines.length === maxLines) {
      let last = lines.pop();
      while (last.includes(' ') && measure(`${last}…`, style) > maxWidth) last = last.replace(/\s+\S+$/, '');
      lines.push(`${last.replace(/[\s,.;:]+$/, '')}…`);
      return lines;
    }
  }
  if (line) lines.push(line);
  return lines;
}

const tspans = (lines, x, y, lineHeight, anchor = 'start') => lines
  .map((l, i) => `<text x="${x}" y="${y + i * lineHeight}" text-anchor="${anchor}">${xml(l)}</text>`).join('');

const frame = (content, label) => `<svg xmlns="http://www.w3.org/2000/svg" width="${OG_WIDTH}" height="${OG_HEIGHT}" viewBox="0 0 ${OG_WIDTH} ${OG_HEIGHT}" role="img" aria-label="${xml(label)}">
<rect width="${OG_WIDTH}" height="${OG_HEIGHT}" fill="#fff"/>
${content}
</svg>`;

// Home: the wordmark large and centered, the tagline under it (like docs/assets/mestiere-social.png).
export function homeSvg({ tagline }) {
  const width = 620;
  const x = (OG_WIDTH - width) / 2;
  const style = { size: 44, italic: true };
  const lines = wrap(tagline, style, 1000, 2);
  return frame(`${wordmark(x, 170, width)}
<g ${textAttrs(style)}>${tspans(lines, OG_WIDTH / 2, 430, 58, 'middle')}</g>`, tagline);
}

// Doc pages: section eyebrow, page title, description, then the wordmark and the page URL.
export function pageSvg({ eyebrow, title, description, url }) {
  const M = 80;
  const maxWidth = OG_WIDTH - 2 * M;
  const titleStyle = { size: 92 };
  const descStyle = { size: 34, italic: true };
  const titleLines = wrap(title, titleStyle, maxWidth, 2);
  const titleY = 225;
  const descY = titleY + (titleLines.length - 1) * 100 + 78;
  const descLines = wrap(description, descStyle, maxWidth, titleLines.length > 1 ? 2 : 3);
  const urlStyle = { size: 26, italic: true };
  return frame(`<g ${textAttrs({ size: 28, italic: true })}><text x="${M}" y="118">${xml(eyebrow)}</text></g>
<g ${textAttrs(titleStyle)}>${tspans(titleLines, M, titleY, 100)}</g>
<g ${textAttrs(descStyle)}>${tspans(descLines, M, descY, 48)}</g>
<rect x="${M}" y="508" width="${maxWidth}" height="1.5" fill="#000"/>
${wordmark(M, 532, 170)}
<g ${textAttrs(urlStyle)}><text x="${OG_WIDTH - M}" y="566" text-anchor="end">${xml(url)}</text></g>`, `${title}: ${description}`);
}

// SVG -> PNG, through the content-hash cache.
export function renderPng(svg) {
  const key = createHash('sha256').update(svg).digest('hex').slice(0, 32);
  const cached = join(CACHE, `${key}.png`);
  if (existsSync(cached)) return readFileSync(cached);
  const png = new Resvg(svg, { ...resvgOptions, fitTo: { mode: 'width', value: OG_WIDTH } }).render().asPng();
  mkdirSync(CACHE, { recursive: true });
  writeFileSync(cached, png);
  return png;
}
