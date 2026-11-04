// Re-lays out the polarity/mestiere board after page heights changed: measures every page of
// dist-lavagna at 1440 and 390 wide (Playwright), resizes each frame to its page and restacks rows
// (160 apart) and groups (400 apart) inside each column. Frame x positions are kept.
//
//   npm run lavagna && lavagna push --yes && node lavagna-layout.mjs
//
// Playwright isn't a dependency of the site: set PLAYWRIGHT_FROM to a package.json of a project
// that has it (default: ../../books-generator/package.json on Giovanni's Mac).
import { execFileSync } from 'node:child_process';
import { readdirSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = dirname(fileURLToPath(import.meta.url));
const DIR = join(ROOT, 'dist-lavagna');
const require = createRequire(process.env.PLAYWRIGHT_FROM || join(ROOT, '../../books-generator/package.json'));
const { chromium } = require('playwright');

const heights = {};
const browser = await chromium.launch();
for (const [w, h] of [[1440, 900], [390, 844]]) {
  const page = await browser.newPage({ viewport: { width: w, height: h } });
  for (const file of readdirSync(DIR)) {
    await page.goto(`file://${DIR}/${file}`);
    await page.evaluate(() => document.fonts.ready);
    (heights[file] ??= {})[w] = Math.max(h, Math.ceil(await page.evaluate(() => document.documentElement.scrollHeight)));
  }
  await page.close();
}
await browser.close();

const cli = (...a) => execFileSync('lavagna', [...a.map(String), '--board', 'main', '--yes'], { cwd: ROOT, encoding: 'utf8' });
const elements = JSON.parse(cli('board', '--json')).elements;
const groups = elements.filter((e) => e.kind === 'section' || e.kind === 'group');
const frames = elements.filter((e) => e.kind === 'frame');
const inside = (f, g) => f.x >= g.x && f.x < g.x + g.width && f.y >= g.y && f.y < g.y + g.height;
let updates = 0;
for (const x0 of [...new Set(groups.map((g) => g.x))]) {
  const column = groups.filter((g) => g.x === x0).sort((a, b) => a.y - b.y);
  let y = column[0].y;
  for (const group of column) {
    const mine = frames.filter((f) => inside(f, group));
    let rowY = y + 160;
    for (const rowTop of [...new Set(mine.map((f) => f.y))].sort((a, b) => a - b)) {
      let tallest = 0;
      for (const f of mine.filter((m) => m.y === rowTop)) {
        const h = heights[f.url.replace(/^dist\//, '')]?.[f.width === 390 ? 390 : 1440] ?? f.height;
        tallest = Math.max(tallest, h);
        if (f.y !== rowY || f.height !== h) { cli('board', 'update', f.id, '--y', rowY, '--height', h); updates++; }
      }
      rowY += tallest + 160;
    }
    const height = rowY - 160 + 120 - y;
    if (group.y !== y || group.height !== height) { cli('board', 'update', group.id, '--y', y, '--height', height); updates++; }
    y += height + 400;
  }
}
console.log(`${updates} board elements updated`);
