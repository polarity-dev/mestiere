// Lavagna build: turns website/dist into website/dist-lavagna, one self-contained HTML file per page.
// CSS, fonts, images and the script are inlined (data: URIs), so a frame needs no other file and no
// URL resolution. Pages are flattened (docs/ci/index.html -> docs-ci.html) and links between pages
// point at the flat names. Links to non-page files (llms.txt, *.md) go to https://mestiere.io.
//
//   npm run build && npm run lavagna && lavagna push --yes
import { mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { dirname, extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = dirname(fileURLToPath(import.meta.url));
const SRC = join(ROOT, 'dist');
const OUT = join(ROOT, 'dist-lavagna');
const SITE = 'https://mestiere.io';
const MIME = { '.svg': 'image/svg+xml', '.png': 'image/png', '.woff2': 'font/woff2' };

const files = (dir) => readdirSync(dir).flatMap((name) => {
  const path = join(dir, name);
  return statSync(path).isDirectory() ? files(path) : [path];
});

const read = (url) => readFileSync(join(SRC, url.replace(/^\//, '')));
const dataUri = (url) => {
  const type = MIME[extname(url)];
  if (!type) throw new Error(`no MIME type for ${url}`);
  return `data:${type};base64,${read(url).toString('base64')}`;
};

// "/" -> "index.html", "/docs/ci/" -> "docs-ci.html", "/install/index.html" -> "install.html"
const pageName = (path) => {
  const clean = path.replace(/^\//, '').replace(/(^|\/)index\.html$/, '$1').replace(/\.html$/, '').replace(/\/$/, '');
  return clean ? `${clean.replaceAll('/', '-')}.html` : 'index.html';
};

const pages = files(SRC).filter((file) => file.endsWith('.html')).map((file) => `/${relative(SRC, file).split('\\').join('/')}`);
const pageSet = new Set(pages.map(pageName));

// Root-relative link from <a href>: a page of the site becomes its flat name, anything else the live URL.
const link = (url) => {
  const [, path, suffix = ''] = url.match(/^([^?#]*)(.*)$/);
  const name = pageName(path);
  if ((path.endsWith('/') || path.endsWith('.html')) && pageSet.has(name)) return name + suffix;
  return SITE + url;
};

const baseCss = readFileSync(join(SRC, 'css/site.css'), 'utf8')
  .replace(/url\("(\/[^"]+)"\)/g, (_, url) => `url("${dataUri(url)}")`);
const js = readFileSync(join(SRC, 'js/site.js'), 'utf8');

// Frames must not depend on the reviewer's OS theme: every page is built twice, with the color
// scheme forced to light (<name>.html) and to dark (<name>-dark.html).
const DARK_QUERY = '(prefers-color-scheme: dark)';
if (!baseCss.includes(`@media ${DARK_QUERY}`)) throw new Error('site.css: dark-mode media query not found');
const themes = {
  light: { suffix: '', media: 'not all', scheme: 'light' },
  dark: { suffix: '-dark', media: 'all', scheme: 'dark' },
};
const cssOf = (theme) => `${baseCss.replace(`@media ${DARK_QUERY}`, `@media ${theme.media}`)}\n:root { color-scheme: ${theme.scheme}; }`;
const withSuffix = (name, suffix) => name.replace(/\.html$/, `${suffix}.html`);

// Extra states that are not reachable by URL. Each one is a separate file and frame.
const states = {
  'index.html': [
    { suffix: '-faq-open', apply: (html) => html.replaceAll('<details class="faq-item">', '<details class="faq-item" open>') },
  ],
  // Docs menu of narrow screens, open (only visible in the smartphone frames)
  'docs.html': [{ suffix: '-menu-open', apply: (html) => html.replace(/(<nav class="docs-menu"[^>]*>\s*<details)>/, '$1 open>') }],
};
// The 404 file holds both languages; this state shows the Italian one, as not-found.js does under /it/
states['404.html'] = [{
  suffix: '-it',
  apply: (html) => html.replace('<html lang="en">', '<html lang="it" class="is-it">')
    .replace(/<title>[^<]*<\/title>/, '<title>Pagina non trovata · mestiere</title>'),
}];
// The Italian pages (/it/... -> it-*.html) get the same states
states['it.html'] = states['index.html'];
states['it-docs.html'] = states['docs.html'];

const inline = (html, theme) => {
  const css = cssOf(theme);
  const out = html
    .replaceAll(`media="${DARK_QUERY}"`, `media="${theme.media}"`)
    // Head links that only make sense on the real site, or would need another file
    .replace(/\s*<link rel="(?:preload|manifest|sitemap|describedby|alternate|apple-touch-icon)"[^>]*>/g, '')
    .replace(/\s*<link rel="icon" type="image\/png"[^>]*>/g, '')
    .replace(/<link rel="icon" type="image\/svg\+xml" href="(\/[^"]+)">/, (_, url) => `<link rel="icon" type="image/svg+xml" href="${dataUri(url)}">`)
    .replace(/<link rel="stylesheet" href="\/css\/site\.css">/, () => `<style>\n${css}\n</style>`)
    // Inline scripts cannot be deferred: run it at the end of <body> instead
    .replace(/\s*<script src="\/js\/site\.js" defer><\/script>/, '')
    // 404 language pick by path: on Lavagna each language is its own file (state 404-it)
    .replace(/\s*<script src="\/js\/not-found\.js"><\/script>/, '')
    .replace(/<\/body>/, () => `<script>\n${js}\n</script>\n</body>`)
    .replace(/(<(?:img|source)\b[^>]*\s(?:src|srcset)=")(\/(?!\/)[^"]+)"/g, (_, before, url) => `${before}${dataUri(url)}"`)
    // Dark pages link to dark pages
    .replace(/(<a\b[^>]*\shref=")(\/(?!\/)[^"]*)"/g, (_, before, url) => {
      const target = link(url);
      return `${before}${target.startsWith('http') ? target : target.replace(/^([^#?]+)\.html/, `$1${theme.suffix}.html`)}"`;
    });
  const left = out.match(/\s(?:href|src|srcset)="\/(?!\/)[^"]*"|url\("\/[^"]*"\)/g);
  if (left) throw new Error(`root-relative URLs left: ${left.join(', ')}`);
  return out;
};

rmSync(OUT, { recursive: true, force: true });
mkdirSync(OUT, { recursive: true });
const written = [];
const write = (name, html) => {
  writeFileSync(join(OUT, name), html);
  written.push(name);
};
for (const page of pages) {
  const name = pageName(page);
  const html = readFileSync(join(SRC, page), 'utf8');
  for (const theme of Object.values(themes)) {
    write(withSuffix(name, theme.suffix), inline(html, theme));
    for (const state of states[name] ?? []) {
      const stated = state.apply(html);
      if (stated === html) throw new Error(`${name}: state ${state.suffix} changed nothing`);
      write(withSuffix(name, `${state.suffix}${theme.suffix}`), inline(stated, theme));
    }
  }
}
console.log(`${written.length} self-contained pages -> ${relative(process.cwd(), OUT) || OUT}: ${written.join(', ')}`);
