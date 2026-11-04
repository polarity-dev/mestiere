// Builds mestiere.io into website/dist.
//
// content/**/*.md  -> one HTML page per file (clean URL: /docs/commands/), plus a
//                     Markdown twin at the same URL + index.md (llms.txt v2 convention)
// content/it/**    -> the Italian site under /it/ (same page set, same slugs, same heading ids)
// templates/       -> layout.html (every page) and home.html / home.it.html (the / body)
// static/          -> copied as is
// generated        -> sitemap.xml, robots.txt, llms.txt, llms-full.txt, 404.html
//
// The build fails on broken internal links and anchors.
import { execFileSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync, readdirSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Marked, Renderer } from 'marked';
import { heroGraphic, notFoundGraphic } from './hero-graphic.mjs';

const ROOT = dirname(fileURLToPath(import.meta.url));
const CONTENT = join(ROOT, 'content');
const STATIC = join(ROOT, 'static');
const TEMPLATES = join(ROOT, 'templates');
const DIST = join(ROOT, 'dist');

const SITE = {
  url: 'https://mestiere.io',
  name: 'mestiere',
  tagline: "Your team's agent skills, versioned in git and installed into every repository.",
  repo: 'https://github.com/polarity-dev/mestiere',
  org: { name: 'Polarity', url: 'https://polarity.dev' },
};

// The site is bilingual. Italian pages live under /it/ with the same paths after the prefix, so
// every page has exactly one counterpart and the language switch never lands on a missing page.
// Content links are always written with the English paths; the build adds /it to them.
const LANGS = {
  en: { code: 'en', prefix: '', locale: 'en_US', short: 'EN' },
  it: { code: 'it', prefix: '/it', locale: 'it_IT', short: 'IT' },
};
const STRINGS = {
  en: {
    skipLink: 'Skip to content', homeLabel: 'mestiere home', mainNav: 'Main', footerNav: 'Footer',
    nav: { install: 'Install', docs: 'Docs', commands: 'Commands' },
    footer: { install: 'Install', start: 'Getting started', commands: 'Command reference', registries: 'Registries', releases: 'Releases' },
    builtBy: 'built by', docs: 'Documentation', docsMenu: 'Documentation menu', toc: 'On this page',
    pagerNav: 'Previous and next page', prev: '← Previous', next: 'Next →', updated: 'Updated',
    markdownVersion: 'Markdown version', switchTo: 'Read in English', ogImageAlt: 'The mestiere logo, white on black',
    loomTitle: 'One skills registry feeding every repository of a team', crumbDocs: 'Docs',
  },
  it: {
    skipLink: 'Vai al contenuto', homeLabel: 'Home di mestiere', mainNav: 'Principale', footerNav: 'Piè di pagina',
    nav: { install: 'Installa', docs: 'Docs', commands: 'Comandi' }, // short: they share one row with the switch on phones
    footer: { install: 'Installazione', start: 'Primi passi', commands: 'Riferimento comandi', registries: 'Registry', releases: 'Release' },
    builtBy: 'fatto da', docs: 'Documentazione', docsMenu: 'Menu della documentazione', toc: 'In questa pagina',
    pagerNav: 'Pagina precedente e successiva', prev: '← Precedente', next: 'Successiva →', updated: 'Aggiornata il',
    markdownVersion: 'Versione Markdown', switchTo: 'Leggi in italiano', ogImageAlt: 'Il logo di mestiere, bianco su nero',
    loomTitle: 'Un registry di skill che alimenta ogni repository di un team', crumbDocs: 'Documentazione',
  },
};
const otherLang = (lang) => (lang === 'en' ? 'it' : 'en');

function git(...args) {
  try {
    return execFileSync('git', args, { cwd: ROOT, encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim();
  } catch {
    return '';
  }
}

// Latest release tag drives every install snippet.
// Build-time version: the highest vX.Y.Z tag on GitHub (local tags can be stale or deleted
// upstream), else the local one. The pages also read the live version at load (site.js).
const semver = (tag) => tag.replace(/^v/, '').split('.').map(Number);
const newest = (tags) => tags.filter((t) => /^v\d+\.\d+\.\d+$/.test(t))
  .sort((a, b) => { const [x, y] = [semver(a), semver(b)]; return x[0] - y[0] || x[1] - y[1] || x[2] - y[2]; }).pop();
const remoteTags = git('ls-remote', '--tags', '--refs', SITE.repo + '.git').split('\n').map((l) => l.split('refs/tags/')[1]).filter(Boolean);
const VERSION = (newest(remoteTags) || git('describe', '--tags', '--abbrev=0', '--match', 'v[0-9]*') || 'v0.1.0').replace(/^v/, '');

// Every version shown in the HTML pages is read live from GitHub by static/js/site.js. Until it
// arrives the slot holds the build-time VERSION, invisible (it keeps the space); then it fades in.
// Without JavaScript CSS reveals the build-time value after a few seconds. Markdown twins, llms.txt and
// JSON-LD keep the build-time value (crawlers don't run scripts).
const VERSION_TOKEN = 'MESTIERELIVEVERSIONTOKEN';
// GitHub mark (Primer Octicons, MIT), served from our own files and inlined so it follows the text color
const GITHUB_ICON = readFileSync(join(STATIC, 'assets/icons/github-mark.svg'), 'utf8')
  .replace(/<!--[\s\S]*?-->\s*/, '').trim()
  .replace('<svg ', '<svg class="icon-github" aria-hidden="true" focusable="false" ');
const liveVersion = (prefix = '') => `<span class="live-version" data-prefix="${prefix}" data-fallback="${VERSION}">${prefix}${VERSION}</span>`;
const withLiveVersion = (html) => html.replaceAll(VERSION_TOKEN, liveVersion());

// Markdown twins, llms.txt and llms-full.txt are static files that agents may read months after
// a build, so they never state a version (the HTML shows it live instead):
// - lines marked <!-- html-only --> are dropped (e.g. "The latest version is ...");
// - any other {{version}} becomes <version>, e.g. mestiere_<version>_darwin_arm64.tar.gz.
const versionlessMarkdown = (md) => md
  .split('\n').filter((line) => !line.includes('<!-- html-only -->')).join('\n')
  .replaceAll('{{version}}', '<version>');
const BUILD_DATE = new Date().toISOString();

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const escapeHtml = (s) => String(s)
  .replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;').replaceAll("'", '&#39;');

const stripTags = (s) => s.replace(/<[^>]+>/g, '').replace(/&lt;/g, '<').replace(/&gt;/g, '>')
  .replace(/&quot;/g, '"').replace(/&#39;/g, "'").replace(/&amp;/g, '&');

const slugify = (s) => stripTags(s).toLowerCase()
  .replace(/[`'".,:;!?()[\]{}<>/\\|*+=~^$@#%&]/g, '')
  .trim().replace(/\s+/g, '-').replace(/-+/g, '-');

const fill = (template, values) => template.replace(/\{\{(\w+)\}\}/g, (m, key) => {
  if (!(key in values)) throw new Error(`template placeholder {{${key}}} has no value`);
  return values[key];
});

// Minimal front matter: `key: value` lines between --- fences.
function parseFrontMatter(source, file) {
  const m = source.match(/^---\n([\s\S]*?)\n---\n/);
  if (!m) throw new Error(`${file}: missing front matter`);
  const data = {};
  for (const line of m[1].split('\n')) {
    const kv = line.match(/^(\w+):\s*(.*)$/);
    if (kv) data[kv[1]] = kv[2].replace(/^"(.*)"$/, '$1');
  }
  return { data, body: source.slice(m[0].length) };
}

function walk(dir) {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });
}

function write(path, content) {
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, content);
}

// Last commit date of a file, so sitemap lastmod only moves when content does.
function lastModified(file) {
  return git('log', '-1', '--format=%cI', '--', relative(ROOT, file)) || BUILD_DATE;
}

// ---------------------------------------------------------------------------
// Markdown
// ---------------------------------------------------------------------------
// Each render collects its h2/h3 headings for the "On this page" list.
function renderMarkdown(markdown) {
  const headings = [];
  const used = new Set();
  const marked = new Marked({ gfm: true });
  marked.use({
    renderer: {
      heading({ tokens, depth }) {
        let html = this.parser.parseInline(tokens);
        // `## Titolo {#english-id}`: translated headings keep the English anchor
        const explicit = html.match(/\s*\{#([a-z0-9-]+)\}\s*$/);
        if (explicit) html = html.slice(0, explicit.index);
        const base = explicit ? explicit[1] : slugify(html);
        let id = base;
        for (let n = 2; used.has(id); n++) id = `${base}-${n}`;
        used.add(id);
        if (depth === 2 || depth === 3) headings.push({ depth, id, text: stripTags(html) });
        return `<h${depth} id="${id}"><a class="anchor" href="#${id}" aria-hidden="true" tabindex="-1">#</a>${html}</h${depth}>\n`;
      },
      table(token) {
        // Default table markup inside a scroll container for narrow screens.
        return `<div class="table-wrap">${Renderer.prototype.table.call(this, token)}</div>\n`;
      },
    },
  });
  return { html: marked.parse(markdown), headings };
}

const stripHeadingIds = (md) => md.replace(/[ \t]*\{#[a-z0-9-]+\}[ \t]*$/gm, '');

// Markdown twins link to other twins, absolute, as llms.txt recommends.
const toMarkdownLinks = (md) => md.replace(/\]\((\/[^)#\s]*)(#[^)\s]*)?\)/g, (m, path, hash = '') => {
  if (!path.endsWith('/')) return `](${SITE.url}${path}${hash})`;
  return `](${SITE.url}${path}index.md${hash})`;
});

// ---------------------------------------------------------------------------
// Pages
// ---------------------------------------------------------------------------
function loadPages(lang) {
  const base = lang === 'en' ? CONTENT : join(CONTENT, lang);
  const files = walk(base).filter((f) => f.endsWith('.md'))
    .filter((f) => lang !== 'en' || !relative(CONTENT, f).startsWith(`it/`));
  return files.map((file) => {
    const { data, body } = parseFrontMatter(readFileSync(file, 'utf8'), file);
    for (const key of ['title', 'description']) {
      if (!data[key]) throw new Error(`${file}: front matter needs "${key}"`);
    }
    if (data.description.length > 160) {
      console.warn(`warning: ${relative(ROOT, file)}: description is ${data.description.length} chars (aim for <= 160)`);
    }
    const rel = relative(base, file).replace(/\.md$/, '');
    // key: the language-neutral path, the same for a page and its translation
    const key = rel === 'index' ? '/' : `/${rel.replace(/(^|\/)index$/, '')}/`.replace('//', '/');
    const path = LANGS[lang].prefix + key;
    const markdown = versionlessMarkdown(body);
    // HTML pages show the version live from GitHub: a token here, the live element after rendering
    const htmlMarkdown = body.replaceAll('{{version}}', VERSION_TOKEN);
    return {
      file,
      lang,
      key,
      path,
      url: SITE.url + path,
      title: data.title,
      seoTitle: data.seoTitle || data.title,
      description: data.description,
      navTitle: data.nav || data.title,
      order: Number(data.order ?? 999),
      markdown,
      htmlMarkdown,
      modified: lastModified(file),
    };
  }).sort((a, b) => a.order - b.order);
}

// Pages in the docs sidebar, in reading order.
const docPages = (pages) => pages.filter((p) => p.path !== '/');

function sidebar(pages, current, t) {
  const items = docPages(pages).map((p) => {
    const attrs = p.path === current.path ? ' aria-current="page"' : '';
    return `<li><a href="${p.path}"${attrs}>${escapeHtml(p.navTitle)}</a></li>`;
  }).join('\n          ');
  // Wide screens: sidebar. Narrow screens: a collapsible menu that names the current page.
  return `<nav class="docs-nav" aria-label="${t.docs}">
        <p class="docs-nav-title">${t.docs}</p>
        <ul>
          ${items}
        </ul>
      </nav>
      <nav class="docs-menu" aria-label="${t.docsMenu}">
        <details>
          <summary><span class="docs-menu-label">${t.docs}</span><span class="docs-menu-current">${escapeHtml(current.navTitle)}</span></summary>
          <ul>
          ${items}
          </ul>
        </details>
      </nav>`;
}

function toc(headings, t) {
  const h2 = headings.filter((h) => h.depth === 2);
  if (h2.length < 2) return '';
  const items = h2.map((h) => `<li><a href="#${h.id}">${escapeHtml(h.text)}</a></li>`).join('\n            ');
  return `<nav class="toc" aria-label="${t.toc}">
          <p class="toc-title">${t.toc}</p>
          <ul>
            ${items}
          </ul>
        </nav>`;
}

function pager(pages, current, t) {
  const list = docPages(pages);
  const i = list.findIndex((p) => p.path === current.path);
  const prev = list[i - 1];
  const next = list[i + 1];
  const link = (p, rel, label) => p
    ? `<a class="pager-${rel}" href="${p.path}" rel="${rel}"><span>${label}</span>${escapeHtml(p.navTitle)}</a>`
    : '<span></span>';
  return `<nav class="pager" aria-label="${t.pagerNav}">
          ${link(prev, 'prev', t.prev)}
          ${link(next, 'next', t.next)}
        </nav>`;
}

// ---------------------------------------------------------------------------
// Structured data (schema.org JSON-LD)
// ---------------------------------------------------------------------------
const ORG = { '@type': 'Organization', '@id': `${SITE.url}/#organization`, name: SITE.org.name, url: SITE.org.url };
const WEBSITE = {
  '@type': 'WebSite', '@id': `${SITE.url}/#website`, url: `${SITE.url}/`, name: SITE.name,
  description: SITE.tagline, inLanguage: ['en', 'it'], publisher: { '@id': ORG['@id'] },
};
const SOFTWARE = {
  '@type': 'SoftwareApplication',
  '@id': `${SITE.url}/#software`,
  name: 'mestiere',
  alternateName: 'mestiere CLI',
  description: 'Command-line tool that installs Agent Skills from git registries into every repository, pinned to an exact version, for Claude Code, Kiro and other coding agents.',
  applicationCategory: 'DeveloperApplication',
  applicationSubCategory: 'Command-line tool',
  operatingSystem: 'macOS, Linux',
  softwareVersion: VERSION,
  url: `${SITE.url}/`,
  downloadUrl: `${SITE.repo}/releases/latest`,
  installUrl: `${SITE.url}/install/`,
  codeRepository: SITE.repo,
  programmingLanguage: 'Go',
  softwareRequirements: 'git',
  image: `${SITE.url}/assets/social.png`,
  publisher: { '@id': ORG['@id'] },
};

function breadcrumbs(page) {
  const prefix = LANGS[page.lang].prefix;
  const items = [{ name: 'mestiere', url: `${SITE.url}${prefix}/` }];
  if (page.key.startsWith('/docs/') && page.key !== '/docs/') items.push({ name: STRINGS[page.lang].crumbDocs, url: `${SITE.url}${prefix}/docs/` });
  items.push({ name: page.navTitle, url: page.url });
  return {
    '@type': 'BreadcrumbList',
    itemListElement: items.map((it, i) => ({ '@type': 'ListItem', position: i + 1, name: it.name, item: it.url })),
  };
}

// FAQ entries come from the "## Frequently asked questions" section of content/index.md.
function extractFaq(markdown) {
  const section = markdown.split(/^## (?:Frequently asked questions|.*\{#frequently-asked-questions\})\s*$/m)[1];
  if (!section) return [];
  return section.split(/^### /m).slice(1).map((chunk) => {
    const [question, ...rest] = chunk.split('\n');
    return { question: question.trim(), answer: rest.join('\n').split(/^## /m)[0].trim() };
  });
}

const jsonLd = (graph) => `<script type="application/ld+json">${JSON.stringify({ '@context': 'https://schema.org', '@graph': graph })
  .replaceAll('<', '\\u003c')}</script>`;

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------
const layout = readFileSync(join(TEMPLATES, 'layout.html'), 'utf8');
const homeTemplates = {
  en: readFileSync(join(TEMPLATES, 'home.html'), 'utf8'),
  it: readFileSync(join(TEMPLATES, 'home.it.html'), 'utf8'),
};

// Language-neutral page paths, filled by build(): links to them get the /it prefix on Italian pages.
const PAGE_KEYS = new Set();

// Adds the language prefix to every internal page link (href="/docs/ci/#x" -> "/it/docs/ci/#x").
// Assets, llms.txt and other files keep their path: they are shared by both languages.
function localizeHtml(html, lang) {
  if (lang === 'en') return html;
  return html.replace(/(\shref=")(\/[^"#?]*)([#?][^"]*)?"/g, (m, attr, path, rest = '') => (
    PAGE_KEYS.has(path) ? `${attr}${LANGS[lang].prefix}${path}${rest}"` : m
  ));
}
function localizeMarkdown(md, lang) {
  if (lang === 'en') return md;
  return md.replace(/\]\((\/[^)#\s]*)(#[^)\s]*)?\)/g, (m, path, hash = '') => (
    PAGE_KEYS.has(path) ? `](${LANGS[lang].prefix}${path}${hash})` : m
  ));
}

// Main navigation: the item of the current section gets aria-current="page".
const NAV = [
  { href: '/install/', id: 'install', match: (key) => key === '/install/' },
  { href: '/docs/', id: 'docs', match: (key) => key.startsWith('/docs/') && key !== '/docs/commands/' },
  { href: '/docs/commands/', id: 'commands', match: (key) => key === '/docs/commands/' },
];
const headerNav = (key, t) => NAV.map((item) => `<li><a href="${item.href}"${item.match(key) ? ' aria-current="page"' : ''}>${t.nav[item.id]}</a></li>`).join('\n          ');
const footerLinks = (t) => [
  ['/install/', t.footer.install], ['/docs/', t.footer.start], ['/docs/commands/', t.footer.commands],
  ['/docs/registries/', t.footer.registries], [`${SITE.repo}/releases`, t.footer.releases], ['/llms.txt', 'llms.txt'],
].map(([href, label]) => `<li><a href="${href}">${label}</a></li>`).join('\n          ');

// The switch links to the same page in the other language. ?lang= tells the CloudFront function
// to remember the choice in a cookie, so the browser-language redirect never overrides it.
function langSwitch(page) {
  const other = otherLang(page.lang);
  // The 404 page has no counterpart: it switches to the other home page
  const href = `${LANGS[other].prefix}${page.key ?? '/'}?lang=${other}`;
  return `<a class="lang-switch" href="${href}" hreflang="${other}" lang="${other}" aria-label="${STRINGS[other].switchTo}">${LANGS[other].short}</a>`;
}
function hreflangLinks(page) {
  if (!page.key) return '';
  const url = (lang) => `${SITE.url}${LANGS[lang].prefix}${page.key}`;
  return [
    `<link rel="alternate" hreflang="en" href="${url('en')}">`,
    `<link rel="alternate" hreflang="it" href="${url('it')}">`,
    `<link rel="alternate" hreflang="x-default" href="${url('en')}">`,
  ].join('\n  ');
}

function renderPage({ page, body, graph, robots = 'index, follow, max-snippet:-1, max-image-preview:large', markdownPath, bodyClass }) {
  const t = STRINGS[page.lang];
  // Brand suffix only when the title doesn't already name the product.
  const title = /mestiere/i.test(page.seoTitle) ? page.seoTitle : `${page.seoTitle} · mestiere`;
  const alternate = markdownPath
    ? `<link rel="alternate" type="text/markdown" href="${markdownPath}" title="${t.markdownVersion}">`
    : '';
  const html = fill(layout, {
    htmlLang: page.lang,
    hreflang: hreflangLinks(page),
    ogLocale: LANGS[page.lang].locale,
    ogLocaleAlternate: LANGS[otherLang(page.lang)].locale,
    ogImageAlt: t.ogImageAlt,
    skipLink: t.skipLink,
    homeHref: '/',
    homeLabel: t.homeLabel,
    mainNavLabel: t.mainNav,
    footerNavLabel: t.footerNav,
    footerLinks: footerLinks(t),
    builtBy: t.builtBy,
    langSwitch: 'LANGSWITCHTOKEN',
    title: escapeHtml(title),
    description: escapeHtml(page.description),
    canonical: page.url,
    robots,
    ogType: page.key === '/' ? 'website' : 'article',
    ogTitle: escapeHtml(page.key === '/' ? 'mestiere' : page.seoTitle),
    alternate,
    jsonld: graph ? jsonLd(graph) : '',
    bodyClass,
    nav: headerNav(page.key ?? '', t),
    body,
    githubIcon: GITHUB_ICON,
    liveVersionV: liveVersion('v'),
    year: String(new Date().getFullYear()),
  });
  // Localize every page link first, then add the switch, which must keep the other language's path
  return localizeHtml(html, page.lang).replace('LANGSWITCHTOKEN', langSwitch(page));
}

function renderDoc(pages, page) {
  const t = STRINGS[page.lang];
  const rendered = renderMarkdown(page.htmlMarkdown);
  const html = withLiveVersion(rendered.html);
  const { headings } = rendered;
  const body = `<div class="container docs-layout">
      ${sidebar(pages, page, t)}
      <article class="doc" id="main">
        <header class="doc-header">
          <h1>${escapeHtml(page.title)}</h1>
          <p class="doc-lead">${escapeHtml(page.description)}</p>
        </header>
        ${toc(headings, t)}
        <div class="doc-body">
${html}
        </div>
        ${pager(pages, page, t)}
        <p class="doc-meta">${t.updated} <time datetime="${page.modified}">${page.modified.slice(0, 10)}</time> · <a href="${page.path}index.md">${t.markdownVersion}</a></p>
      </article>
    </div>`;
  const graph = [
    ORG, WEBSITE,
    {
      '@type': 'TechArticle',
      '@id': `${page.url}#article`,
      headline: page.title,
      description: page.description,
      url: page.url,
      inLanguage: page.lang,
      dateModified: page.modified,
      isPartOf: { '@id': WEBSITE['@id'] },
      about: { '@id': SOFTWARE['@id'] },
      author: { '@id': ORG['@id'] },
      publisher: { '@id': ORG['@id'] },
      image: `${SITE.url}/assets/social.png`,
    },
    breadcrumbs(page),
  ];
  return renderPage({ page, body, graph, markdownPath: `${page.path}index.md`, bodyClass: 'page-doc' });
}

function renderHome(pages, page) {
  const faq = extractFaq(page.markdown);
  const faqHtml = faq.map(({ question, answer }) => {
    const { html } = renderMarkdown(answer);
    return `<details class="faq-item">
            <summary><h3>${escapeHtml(question)}</h3></summary>
            <div class="faq-answer">${html}</div>
          </details>`;
  }).join('\n          ');
  const body = fill(homeTemplates[page.lang], { githubIcon: GITHUB_ICON, liveVersionV: liveVersion('v'), faq: faqHtml, heroGraphic: heroGraphic(STRINGS[page.lang].loomTitle) });
  const graph = [
    ORG, WEBSITE, SOFTWARE,
    {
      '@type': 'WebPage', '@id': `${page.url}#webpage`, url: page.url, name: page.seoTitle,
      description: page.description, isPartOf: { '@id': WEBSITE['@id'] }, about: { '@id': SOFTWARE['@id'] },
      primaryImageOfPage: `${SITE.url}/assets/social.png`, inLanguage: page.lang,
    },
    {
      '@type': 'FAQPage', '@id': `${page.url}#faq`,
      mainEntity: faq.map(({ question, answer }) => ({
        '@type': 'Question', name: question,
        acceptedAnswer: { '@type': 'Answer', text: stripTags(renderMarkdown(answer).html).trim() },
      })),
    },
  ];
  return renderPage({ page, body, graph, markdownPath: `${page.path}index.md`, bodyClass: 'page-home' });
}

// CloudFront serves the same /404.html for every missing path, in both languages. The file holds
// the English and the Italian page; js/not-found.js, run in <head> before the first paint, shows
// the Italian one under /it/. A reader always sees a single language.
const NOT_FOUND = {
  en: {
    seoTitle: 'Page not found', description: 'This page does not exist on mestiere.io.',
    eyebrow: 'Error 404', title: "This page isn't in the registry",
    text: 'The link may be old, or the page has moved. Your skills are still where you left them.',
    docs: 'Read the docs', home: 'Back to home', graphic: 'A thread from the registry snapped before reaching the 404 repository',
  },
  it: {
    seoTitle: 'Pagina non trovata', description: 'Questa pagina non esiste su mestiere.io.',
    eyebrow: 'Errore 404', title: 'Questa pagina non è nel registry',
    text: 'Il link potrebbe essere vecchio, o la pagina si è spostata. Le tue skill sono ancora dove le hai lasciate.',
    docs: 'Leggi la documentazione', home: 'Torna alla home', graphic: 'Un filo del registry si è spezzato prima di arrivare al repository 404',
  },
};
function render404() {
  const pageOf = (lang) => {
    const s = NOT_FOUND[lang];
    const id = lang === 'en' ? '' : `-${lang}`;
    const page = { lang, key: null, path: '/404.html', url: `${SITE.url}/404.html`, seoTitle: s.seoTitle, description: s.description };
    const body = `<div class="container not-found" id="main${id}">
      <div class="not-found-text">
        <p class="eyebrow">${s.eyebrow}</p>
        <h1>${s.title}</h1>
        <p class="lead">${s.text}</p>
        <p class="actions">
          <a class="button" href="/docs/">${s.docs}</a>
          <a class="button button-primary" href="/">${s.home}</a>
        </p>
      </div>
      <div class="not-found-graphic">
        ${notFoundGraphic(s.graphic, id)}
      </div>
    </div>`;
    return renderPage({ page, body, robots: 'noindex, follow', bodyClass: 'page-404' });
  };
  const bodyOf = (html) => html.match(/<body[^>]*>([\s\S]*)<\/body>/)[1];
  const en = pageOf('en');
  const it = bodyOf(pageOf('it')).replace('href="#main"', 'href="#main-it"');
  return en
    .replace('<script src="/js/site.js" defer></script>', '<script src="/js/not-found.js"></script>\n  <script src="/js/site.js" defer></script>')
    .replace(/(<body[^>]*>)([\s\S]*)(<\/body>)/, (m, open, body, close) => `${open}
  <div class="page-lang page-en">${body}</div>
  <div class="page-lang page-it" lang="it">${it}</div>
${close}`);
}

// ---------------------------------------------------------------------------
// Machine-readable files
// ---------------------------------------------------------------------------
function markdownTwin(page) {
  const md = toMarkdownLinks(localizeMarkdown(stripHeadingIds(page.markdown), page.lang));
  return `# ${page.title}\n\n> ${page.description}\n\n${md.trim()}\n`;
}

function llmsTxt(pages) {
  const home = pages.find((p) => p.path === '/');
  const docs = docPages(pages)
    .map((p) => `- [${p.navTitle}](${p.url}index.md): ${p.description}`).join('\n');
  return `# mestiere

> ${SITE.tagline} mestiere is a command-line tool (macOS, Linux) that installs Agent Skills (folders with a SKILL.md) from git repositories called registries into a project's .agents/skills/ folder, locks each skill to a registry commit and content hash in skills.yaml, and links the skills into .claude/skills/ and .kiro/skills/ for Claude Code and Kiro. Codex, Gemini CLI, Cursor and GitHub Copilot read .agents/skills/ directly.

Key facts for agents:

- Install with \`brew install --cask polarity-dev/tap/mestiere\` (macOS, Linux; upgrade with \`brew upgrade --cask mestiere\`), the latest release archive \`${SITE.repo}/releases/latest/download/mestiere_<os>_<arch>.tar.gz\` (os darwin or linux, arch arm64 or amd64, plus \`checksums.txt\`), or \`go install github.com/polarity-dev/mestiere/cmd/mestiere@latest\`. After a release archive or go install, run \`mestiere add -g mestiere\` once. Requires \`git\` on the PATH. Windows is not supported yet.
- Check the CLI with \`mestiere --version\`. Start with \`mestiere status\`: it lists installed, local and global skills, and in a folder without skills.yaml it says so (then set the project up).
- mestiere's own skills (\`mestiere/project\`, \`mestiere/registry\`) are installed globally by Homebrew or \`mestiere add -g mestiere\`; they cover setting up a repository.
- \`-g\` acts on global skills (\`~/.agents/skills/\`, manifest \`~/.config/mestiere/skills.yaml\`) instead of the project; \`init\`, \`track\` and \`untrack\` don't apply. Prefer project installs for team skills.
- Typical project setup: \`mestiere init --agents claude,kiro\` (only the agents the team uses), \`mestiere registry add <name> <git-url>\`, \`mestiere registry show <name>\` to see what it offers and propose skills to the user, \`mestiere add <namespace>\` or \`mestiere add <namespace>/<skill>\`, then commit \`skills.yaml\` and \`.gitignore\`. Teammates run \`mestiere sync\` after cloning or pulling.
- Updates: \`mestiere outdated\` lists skills whose content changed in the registry; \`mestiere update [targets]\` moves the lock forward; commit \`skills.yaml\` afterwards.
- CI: \`mestiere sync --frozen\` and \`mestiere doctor --strict\` fail on an inconsistent skills.yaml, drifted skills or uncommitted local skills. \`mestiere doctor\` explains every error and its fix.
- Own skills: write them in \`.agents/skills/\` (or an agent folder) and commit them with \`mestiere track <name>\`; installed skills are gitignored by default.
- Always pass targets to \`mestiere add\`: without arguments it opens an interactive picker.
- Never edit the \`lock\` section of skills.yaml or files under .agents/skills/ in place; use \`mestiere eject <name>\` to customize a skill. Do not pass \`--force\` without the user's approval: it discards local edits.
- Installed name of skill \`<namespace>/<skill>\` is \`<namespace>-<skill>\`; namespaces contain no hyphens.
- Every page below is plain Markdown. ${SITE.url}/llms-full.txt holds all of them in one file.

## Docs

- [Overview](${SITE.url}/index.md): ${home.description}
${docs}

## Optional

- [Full documentation in one file](${SITE.url}/llms-full.txt): Every page above, concatenated
- [Documentazione in italiano](${SITE.url}/it/index.md): The same pages in Italian, under ${SITE.url}/it/
- [Source code](${SITE.repo}): Go source, README and releases
- [Agent Skills specification](https://agentskills.io): The SKILL.md format mestiere installs
`;
}

function llmsFullTxt(pages) {
  const parts = pages.map((p) => `<!-- ${p.url} -->\n\n${markdownTwin(p)}`);
  return `# mestiere documentation\n\n> ${SITE.tagline}\n\nSource: ${SITE.url}/llms.txt.\n\n${parts.join('\n---\n\n')}`;
}

function sitemap(pages) {
  const urls = pages.map((p) => `  <url>
    <loc>${p.url}</loc>
    <lastmod>${p.modified}</lastmod>
    <xhtml:link rel="alternate" hreflang="en" href="${SITE.url}${p.key}"/>
    <xhtml:link rel="alternate" hreflang="it" href="${SITE.url}/it${p.key}"/>
    <xhtml:link rel="alternate" hreflang="x-default" href="${SITE.url}${p.key}"/>
  </url>`).join('\n');
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
${urls}
</urlset>
`;
}

// Search engines and AI assistants are all welcome: the site is documentation.
function robotsTxt() {
  return `# mestiere.io: public documentation, open to search engines and AI crawlers.
User-agent: *
Allow: /
Disallow: /404.html

# AI crawlers and assistants, listed explicitly so the intent is unambiguous.
User-agent: GPTBot
User-agent: OAI-SearchBot
User-agent: ChatGPT-User
User-agent: ClaudeBot
User-agent: Claude-SearchBot
User-agent: Claude-User
User-agent: PerplexityBot
User-agent: Perplexity-User
User-agent: Google-Extended
User-agent: Applebot-Extended
User-agent: CCBot
User-agent: Amazonbot
User-agent: meta-externalagent
User-agent: MistralAI-User
Allow: /

Sitemap: ${SITE.url}/sitemap.xml

# LLM-friendly index of this site: ${SITE.url}/llms.txt
`;
}

// ---------------------------------------------------------------------------
// Link check: every internal href/src must exist, and so must its #anchor.
// ---------------------------------------------------------------------------
function checkLinks() {
  const htmlFiles = walk(DIST).filter((f) => f.endsWith('.html'));
  const idsCache = new Map();
  const idsOf = (file) => {
    if (!idsCache.has(file)) {
      idsCache.set(file, new Set([...readFileSync(file, 'utf8').matchAll(/\sid="([^"]+)"/g)].map((m) => m[1])));
    }
    return idsCache.get(file);
  };
  const resolve = (urlPath) => {
    const p = join(DIST, decodeURIComponent(urlPath));
    if (urlPath.endsWith('/')) return join(p, 'index.html');
    return p;
  };
  const errors = [];
  for (const file of htmlFiles) {
    const html = readFileSync(file, 'utf8');
    const pagePath = '/' + relative(DIST, file).replace(/index\.html$/, '');
    for (const [, attr, value] of html.matchAll(/\s(href|src|srcset)="([^"]+)"/g)) {
      for (const ref of attr === 'srcset' ? value.split(',').map((s) => s.trim().split(/\s+/)[0]) : [value]) {
        if (/^(https?:|mailto:|data:)/.test(ref)) continue;
        const [beforeHash, hash] = ref.split('#');
        const pathPart = beforeHash.split('?')[0];
        const target = pathPart === '' ? file : resolve(pathPart.startsWith('/') ? pathPart : pagePath + pathPart);
        if (!existsSync(target)) {
          errors.push(`${relative(DIST, file)}: broken link ${ref}`);
        } else if (hash && target.endsWith('.html') && !idsOf(target).has(hash)) {
          errors.push(`${relative(DIST, file)}: missing anchor ${ref}`);
        }
      }
    }
  }
  // Navigation must stay in the reader's language: an Italian page may link to an English page
  // only through the language switch (the one link with hreflang).
  for (const file of htmlFiles.filter((f) => relative(DIST, f).startsWith('it/'))) {
    for (const [tag, href] of readFileSync(file, 'utf8').matchAll(/<a\b[^>]*\shref="(\/[^"]*)"[^>]*>/g)) {
      const path = href.split(/[#?]/)[0];
      if (PAGE_KEYS.has(path) && !/\shreflang="/.test(tag)) errors.push(`${relative(DIST, file)}: links to the English page ${href}`);
    }
  }
  if (errors.length) {
    throw new Error(`broken internal links:\n  ${errors.join('\n  ')}`);
  }
  return htmlFiles.length;
}

// ---------------------------------------------------------------------------
// Build
// ---------------------------------------------------------------------------
export function build() {
  rmSync(DIST, { recursive: true, force: true });
  cpSync(STATIC, DIST, { recursive: true, filter: (src) => !src.split('/').pop().startsWith('.') });

  const byLang = { en: loadPages('en'), it: loadPages('it') };
  PAGE_KEYS.clear();
  byLang.en.forEach((p) => PAGE_KEYS.add(p.key));
  // Every page must exist in both languages, or the switch and the hreflang links would break
  const missing = [
    ...byLang.en.filter((p) => !byLang.it.some((q) => q.key === p.key)).map((p) => `it${p.key}`),
    ...byLang.it.filter((p) => !PAGE_KEYS.has(p.key)).map((p) => `en${p.key}`),
  ];
  if (missing.length) throw new Error(`pages missing a translation: ${missing.join(', ')}`);
  for (const pages of Object.values(byLang)) {
    for (const page of pages) {
      const html = page.key === '/' ? renderHome(pages, page) : renderDoc(pages, page);
      write(join(DIST, page.path, 'index.html'), html);
      write(join(DIST, page.path, 'index.md'), markdownTwin(page));
    }
  }
  const pages = byLang.en;
  write(join(DIST, '404.html'), render404());
  write(join(DIST, 'sitemap.xml'), sitemap(pages));
  write(join(DIST, 'robots.txt'), robotsTxt());
  write(join(DIST, 'llms.txt'), llmsTxt(pages));
  write(join(DIST, 'llms-full.txt'), llmsFullTxt(pages));

  const checked = checkLinks();
  return { pages: byLang.en.length + byLang.it.length, checked, version: VERSION };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const result = build();
  console.log(`built ${result.pages} pages (v${result.version}), links checked in ${result.checked} HTML files -> ${relative(process.cwd(), DIST) || DIST}`);
}
