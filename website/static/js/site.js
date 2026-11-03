// Progressive enhancement for mestiere.io: every page works without this script.
//   - copy icon on code blocks, "send to my computer", hero tilt, live version from GitHub
//   - in-page navigation: links between pages swap <main> instead of reloading the page
const IT = document.documentElement.lang === 'it';
const T = IT
  ? { copy: 'Copia il codice negli appunti', copied: 'Copiato', failed: 'Copia non riuscita', share: 'Le skill degli agenti per tutto il team', home: 'https://mestiere.io/it/' }
  : { copy: 'Copy code to clipboard', copied: 'Copied', failed: 'Copy failed', share: 'Agent skills for the whole team', home: 'https://mestiere.io/' };
const SVG_NS = 'http://www.w3.org/2000/svg';
const ICONS = {
  copy: ['M9 9h11v11H9z', 'M5 15H4V4h11v1'],
  done: ['M5 12.5l4.5 4.5L19 7.5'],
  failed: ['M6 6l12 12', 'M18 6L6 18'],
};
const icon = (name) => {
  const svg = document.createElementNS(SVG_NS, 'svg');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('aria-hidden', 'true');
  for (const d of ICONS[name]) {
    const path = document.createElementNS(SVG_NS, 'path');
    path.setAttribute('d', d);
    svg.appendChild(path);
  }
  return svg;
};
const reducedMotion = () => matchMedia('(prefers-reduced-motion: reduce)').matches;

// Copy icon on code blocks. The button lives on a wrapper, not in the <pre>: it must not scroll
// with the code.
function initCopyButtons(root) {
  if (!navigator.clipboard) return;
  root.querySelectorAll('pre > code').forEach((code) => {
    const pre = code.parentElement;
    if (pre.parentElement.classList.contains('code-block')) return;
    const wrapper = document.createElement('div');
    wrapper.className = 'code-block';
    if (code.textContent.trim().includes('\n')) wrapper.classList.add('is-multiline');
    pre.replaceWith(wrapper);
    wrapper.appendChild(pre);
    const button = document.createElement('button');
    const show = (name, label) => {
      button.replaceChildren(icon(name));
      button.setAttribute('aria-label', label);
    };
    button.type = 'button';
    button.className = 'copy-button';
    show('copy', T.copy);
    button.addEventListener('click', async () => {
      // Drop trailing "# comments" so pasted commands run as is.
      const text = code.innerText.split('\n').map((line) => line.replace(/\s+#\s.*$/, '')).join('\n').trim();
      try {
        await navigator.clipboard.writeText(text);
        show('done', T.copied);
      } catch {
        show('failed', T.failed);
      }
      setTimeout(() => show('copy', T.copy), 1600);
    });
    pre.classList.add('has-copy');
    wrapper.appendChild(button);
  });
}

// "Send to my computer": the native share sheet where there is one, else the mailto link.
function initShare(root) {
  if (!navigator.share) return;
  root.querySelectorAll('[data-share]').forEach((link) => {
    link.addEventListener('click', (event) => {
      event.preventDefault();
      navigator.share({ title: 'mestiere', text: T.share, url: T.home }).catch(() => {});
    });
  });
}

// Hero graphic: a slight tilt toward the pointer.
function initTilt(root) {
  const loom = root.querySelector('.hero .loom');
  const hero = loom?.closest('.hero');
  if (!hero || !matchMedia('(pointer: fine)').matches || reducedMotion()) return;
  hero.addEventListener('pointermove', (event) => {
    const box = loom.getBoundingClientRect();
    const dx = (event.clientX - (box.left + box.width / 2)) / box.width;
    const dy = (event.clientY - (box.top + box.height / 2)) / box.height;
    loom.style.setProperty('--loom-ry', `${(dx * 8).toFixed(2)}deg`);
    loom.style.setProperty('--loom-rx', `${(-dy * 6).toFixed(2)}deg`);
  });
  hero.addEventListener('pointerleave', () => {
    loom.style.setProperty('--loom-ry', '0deg');
    loom.style.setProperty('--loom-rx', '0deg');
  });
}

// Version: the latest published release, read live once per tab. Until it arrives every
// .live-version is invisible; then it fades in (CSS). On any failure the build-time version fades
// in instead. Pages reached by in-page navigation show the known version at once.
//
// Where the number comes from, in order:
// 1. The Homebrew cask that every release writes to polarity-dev/homebrew-tap, read from
//    raw.githubusercontent.com: public, CORS-enabled, CDN-cached for 5 minutes and not subject to
//    the GitHub API rate limit. It is exactly the version `brew install` gets.
// 2. The GitHub REST API (releases/latest): authoritative, but limited to 60 calls an hour per IP
//    without a token, so a shared office IP can run out.
// github.com/.../releases/latest itself sends no CORS headers, so a browser can't read it.
const CASK_URL = 'https://raw.githubusercontent.com/polarity-dev/homebrew-tap/main/Casks/mestiere.rb';
const RELEASE_URL = 'https://api.github.com/repos/polarity-dev/mestiere/releases/latest';
const CACHE_KEY = 'mestiere-version-2'; // renamed when the source changed, to drop old cached values
const CACHE_MS = 10 * 60_000;
const SEMVER = /^\d+\.\d+\.\d+$/;
let knownVersion = null;
let versionRequest = null;
const fromCask = async (signal) => {
  const response = await fetch(CASK_URL, { signal, cache: 'no-cache' });
  if (!response.ok) throw new Error(`cask: ${response.status}`);
  const version = (await response.text()).match(/^\s*version\s+"([^"]+)"/m)?.[1];
  if (!SEMVER.test(version ?? '')) throw new Error('cask: no version');
  return version;
};
const fromApi = async (signal) => {
  const response = await fetch(RELEASE_URL, { signal, headers: { Accept: 'application/vnd.github+json' } });
  if (!response.ok) throw new Error(`api: ${response.status}`);
  const version = (await response.json()).tag_name?.replace(/^v/, '');
  if (!SEMVER.test(version ?? '')) throw new Error('api: no version');
  return version;
};
const latestVersion = async () => {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 6000);
  try {
    return await fromCask(controller.signal).catch(() => fromApi(controller.signal));
  } finally {
    clearTimeout(timeout);
  }
};
function initVersions(root, instant) {
  const slots = root.querySelectorAll('.live-version[data-fallback]:not(.is-loaded)');
  if (!slots.length) return;
  const show = (version, now) => slots.forEach((slot) => {
    slot.textContent = `${slot.dataset.prefix}${version}`;
    slot.classList.add('is-loaded');
    if (now) slot.classList.add('is-instant');
  });
  if (knownVersion) {
    show(knownVersion, instant);
    return;
  }
  if (!versionRequest) {
    let cached = null;
    try { cached = JSON.parse(sessionStorage.getItem(CACHE_KEY)); } catch { /* storage unavailable */ }
    versionRequest = cached && Date.now() - cached.at < CACHE_MS
      ? Promise.resolve(cached.version)
      : latestVersion().then((version) => {
        try { sessionStorage.setItem(CACHE_KEY, JSON.stringify({ version, at: Date.now() })); } catch { /* ignore */ }
        return version;
      }).catch(() => slots[0].dataset.fallback);
    versionRequest.then((version) => { knownVersion = version; });
  }
  versionRequest.then((version) => show(version, false));
}

// Accordions (<details>: FAQ, docs menu on phones) open and close with a height animation
// instead of snapping. The element animates between its summary's height and its full height;
// a click during an animation reverses it from where it is.
function initAccordions(root) {
  root.querySelectorAll('details').forEach((details) => {
    const summary = details.querySelector(':scope > summary');
    if (!summary || details.dataset.accordion) return;
    details.dataset.accordion = 'on';
    let animation = null;
    const content = () => [...details.children].filter((el) => el !== summary);
    summary.addEventListener('click', (event) => {
      if (reducedMotion() || !details.animate) return;
      event.preventDefault();
      const opening = !details.open || details.classList.contains('is-closing');
      const from = details.offsetHeight;
      animation?.cancel();
      details.classList.remove('is-closing');
      details.style.overflow = 'hidden';
      if (opening) details.open = true;
      // Height of the target state: full when opening, the summary alone when closing
      const borders = details.offsetHeight - details.clientHeight;
      const to = (opening ? details.scrollHeight : summary.offsetHeight) + borders;
      if (!opening) details.classList.add('is-closing');
      animation = details.animate({ height: [`${from}px`, `${to}px`] }, { duration: 320, easing: 'cubic-bezier(0.25, 0.8, 0.25, 1)' });
      content().forEach((el) => el.animate({ opacity: opening ? [0, 1] : [1, 0] }, { duration: opening ? 320 : 200, easing: 'ease-out', fill: 'both' }));
      animation.onfinish = () => {
        if (!opening) details.open = false;
        details.classList.remove('is-closing');
        details.style.overflow = '';
        content().forEach((el) => el.getAnimations().forEach((a) => a.cancel()));
        animation = null;
      };
    });
  });
}

function initPage(root, instant = false) {
  initAccordions(root);
  initCopyButtons(root);
  initShare(root);
  initTilt(root);
  initVersions(root, instant);
}
initPage(document);

// ---------------------------------------------------------------------------
// In-page navigation. A click on a link to another page of the same language fetches it and
// swaps <main>, the nav state, the title and the head links, then updates the history. Header
// and footer stay, so nothing reloads or flashes. Anything unusual (other site, other language,
// modifier keys, files, errors) falls back to a normal page load.
// ---------------------------------------------------------------------------
const cache = new Map();
let currentPage = location.pathname + location.search;
const isPagePath = (path) => path.endsWith('/') || path.endsWith('.html');

function pageUrl(link) {
  if (!(link instanceof HTMLAnchorElement) || !link.href) return null;
  if (link.target && link.target !== '_self') return null;
  if (link.hasAttribute('download') || link.hasAttribute('hreflang') || link.dataset.share !== undefined) return null;
  const url = new URL(link.href, location.href);
  if (url.origin !== location.origin || !isPagePath(url.pathname)) return null;
  return url;
}

function load(url) {
  const key = url.pathname + url.search;
  if (!cache.has(key)) {
    const request = fetch(key, { headers: { Accept: 'text/html' } }).then(async (response) => {
      const type = response.headers.get('content-type') || '';
      if (!response.ok || !type.includes('text/html')) throw new Error(`${response.status}`);
      return { html: await response.text(), url: response.url };
    });
    request.catch(() => cache.delete(key));
    cache.set(key, request);
  }
  return cache.get(key);
}

// Head elements that change from page to page
const HEAD_SELECTORS = ['meta[name="description"]', 'link[rel="canonical"]', 'link[rel="alternate"]', 'meta[property^="og:"]', 'meta[name^="twitter:"]', 'script[type="application/ld+json"]'];

function swap(next) {
  document.title = next.title;
  for (const selector of HEAD_SELECTORS) {
    document.head.querySelectorAll(selector).forEach((el) => el.remove());
    next.head.querySelectorAll(selector).forEach((el) => document.head.appendChild(el));
  }
  document.body.className = next.body.className;
  document.querySelector('.site-header .nav').replaceWith(next.body.querySelector('.site-header .nav'));
  const nextSwitch = next.body.querySelector('.lang-switch');
  if (nextSwitch) document.querySelector('.lang-switch')?.replaceWith(nextSwitch);
  const main = next.body.querySelector('main');
  document.querySelector('main').replaceWith(main);
  initPage(main, true);
  return main;
}

function scrollAfter(url, scrollY) {
  if (scrollY !== undefined) {
    window.scrollTo({ top: scrollY, behavior: 'instant' });
    return;
  }
  const target = url.hash && document.getElementById(decodeURIComponent(url.hash.slice(1)));
  if (target) target.scrollIntoView({ behavior: 'instant' });
  else window.scrollTo({ top: 0, behavior: 'instant' });
}

let navigation = 0;
async function navigate(url, { push = true, scrollY } = {}) {
  const id = ++navigation;
  let page;
  try {
    page = await load(url);
  } catch {
    location.assign(url.href);
    return;
  }
  if (id !== navigation) return; // a newer click won
  const next = new DOMParser().parseFromString(page.html, 'text/html');
  // A page in the other language (e.g. after a server redirect) needs a full load
  if (next.documentElement.lang !== document.documentElement.lang || !next.querySelector('main')) {
    location.assign(page.url || url.href);
    return;
  }
  const finalUrl = new URL(page.url || url.href);
  finalUrl.hash = url.hash;
  if (push) {
    history.replaceState({ scrollY: window.scrollY }, '');
    history.pushState({}, '', finalUrl.href);
  }
  const run = () => {
    const main = swap(next);
    currentPage = finalUrl.pathname + finalUrl.search;
    scrollAfter(finalUrl, scrollY);
    // Move focus to the new content for keyboard and screen reader users
    main.setAttribute('tabindex', '-1');
    main.focus({ preventScroll: true });
  };
  if (document.startViewTransition && !reducedMotion()) document.startViewTransition(run);
  else run();
}

if (window.history?.pushState && window.DOMParser && window.fetch) {
  history.scrollRestoration = 'manual';
  document.addEventListener('click', (event) => {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    // The 404 page holds both languages in one file: leave it with a normal page load
    if (document.body.classList.contains('page-404')) return;
    const link = event.target.closest('a');
    const url = link && pageUrl(link);
    if (!url) return;
    // Same page, only the anchor changes: the browser does that already
    if (url.pathname === location.pathname && url.search === location.search && url.hash) return;
    event.preventDefault();
    navigate(url);
  });
  // Prefetch on hover or focus, so the click lands on a page that is already here
  const prefetch = (event) => {
    if (navigator.connection?.saveData) return;
    const link = event.target.closest?.('a');
    const url = link && pageUrl(link);
    if (url && url.pathname !== location.pathname) load(url).catch(() => {});
  };
  document.addEventListener('pointerover', prefetch, { passive: true });
  document.addEventListener('focusin', prefetch);
  // Every history entry remembers its scroll position, so back and forward land where you were
  let saveTimer;
  window.addEventListener('scroll', () => {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => history.replaceState({ ...history.state, scrollY: window.scrollY }, ''), 120);
  }, { passive: true });
  window.addEventListener('popstate', (event) => {
    // Back or forward between anchors of the same page: no fetch, only the scroll position
    if (location.pathname + location.search === currentPage) {
      scrollAfter(new URL(location.href), event.state?.scrollY);
      return;
    }
    navigate(new URL(location.href), { push: false, scrollY: event.state?.scrollY });
  });
}
