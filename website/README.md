# mestiere.io

Static website of mestiere: Markdown content built into HTML by a small Node script, served
from a private S3 bucket behind CloudFront.

```
website/
├── content/           # one Markdown file per page (front matter: title, seoTitle, nav, description, order)
│   ├── index.md       # home: Markdown twin, plus the FAQ shown on the home page
│   ├── install.md
│   └── docs/          # getting started, skills.yaml, registries, agents, commands, how it works, CI
├── templates/         # layout.html (every page), home.html (body of /)
├── static/            # copied as is: css, js, fonts, logos, favicons, manifest
├── build.mjs          # content + templates + static -> dist/
├── dev.mjs            # local preview with live rebuild
├── infra/site.yaml    # CloudFormation: bucket, OAC, CloudFront, function, headers
└── ops.sh             # cert, stack, dns, deploy, outputs
```

## Develop

```sh
cd website
npm install
npm run dev        # http://127.0.0.1:8080, rebuilds on every change
npm run build      # dist/ only; fails on broken internal links or anchors
```

Add a page by adding a Markdown file under `content/`: `content/docs/foo.md` becomes
`/docs/foo/`, joins the docs sidebar (by `order`), the sitemap, `llms.txt` and `llms-full.txt`.
`{{version}}` in content is replaced with the latest `v*` git tag.

## What the build generates

| Output | For |
|---|---|
| `<page>/index.html` | People and search engines: canonical URL, meta description, Open Graph, JSON-LD (`SoftwareApplication`, `FAQPage`, `TechArticle`, `BreadcrumbList`) |
| `assets/og/<lang>/<page>.png` | Link previews (Open Graph / X card, 1200x630): one per page and language, drawn by `og-image.mjs` with the page title and description, cached in `node_modules/.cache/mestiere-og` |
| `<page>/index.md` | Agents: the same page as Markdown ([llms.txt v2](https://llmstxt.org/) convention), linked with `rel="alternate"` |
| `llms.txt` | Agents: index of the Markdown pages with key facts about the CLI |
| `llms-full.txt` | Agents: every page in one file |
| `sitemap.xml` | Search engines: every page in both languages with its `hreflang` alternates, `lastmod` from each file's last commit |
| `robots.txt` | Everyone allowed, AI crawlers listed explicitly |
| `404.html` | `noindex` error page |

## Where things live

| What | Where |
|---|---|
| Bucket, CloudFront, stack `mestiere-website` | polarity-tools (404420994311), eu-west-1 |
| ACM certificate (`mestiere.io`, `www.mestiere.io`) | polarity-tools, us-east-1 |
| Domain and Route 53 zone `mestiere.io` | Polarity main account (477398036046) |

CloudFront gives every page one URL: `www.mestiere.io` → `mestiere.io`, `/docs` → `/docs/`,
`/docs/index.html` → `/docs/`, all 301. Markdown and `llms*.txt` files are served with
`X-Robots-Tag: noindex` and open CORS, so agents can read them without competing with the HTML
pages in search results.

## Deploy

Credentials come from Shonin (`shonin login polarity`).

```sh
website/ops.sh cert      # once: request and DNS-validate the certificate
website/ops.sh stack     # create or update the stack
website/ops.sh dns       # once: A/AAAA alias records for apex and www
website/ops.sh deploy    # build, upload dist/ and invalidate the cache
```

Every push to `main` that touches `website/` publishes the site on its own
(`.github/workflows/website.yml`, also runnable by hand). It runs `ops.sh deploy` with the
`mestiere-website-github-deploy` role (stack output `GitHubDeployRoleArn`), assumed through
GitHub OIDC and trusted only for `refs/heads/main` of `polarity-dev/mestiere`: it can only
upload to the bucket and invalidate the distribution.

## Review on Lavagna
The site is also shown on the Lavagna project `polarity/mestiere` (board `main`) from a dedicated
build: `lavagna.mjs` turns `dist/` into `dist-lavagna/`, one self-contained HTML file per page
(CSS, fonts, images and script inlined, pages flattened to names like `docs-ci.html`). See the
`lavagna-publish` skill.
```sh
cd website && npm run build && npm run lavagna && lavagna push --yes
```
## Rules for the pages

- Everything is self-hosted. The Content-Security-Policy in `infra/site.yaml` allows only
  `'self'`: no CDNs, web fonts from other hosts, inline `<script>` or `style=""` attributes.
  JSON-LD blocks are data, not scripts, and are fine.
- Keep `seoTitle` around 60 characters and `description` under 160 (the build warns).
- Brand: black and white only. Headings and lead text use Parmigiano Text Light, h3 Parmigiano
  Text Bold (a licensed Typotheque font: only the woff2 web files ship here, converted from the
  OTFs in `docs/assets/fonts/`, which git ignores), body text the system sans.
- `static/assets/`: `logo-*.svg` are copies of `docs/assets/mestiere-wordmark-*.svg`, `mark-*.svg` of `docs/assets/mestiere-mark-*.svg`, `social.png` of `docs/assets/mestiere-social.png` (only the JSON-LD `SoftwareApplication` image: link previews use the generated `assets/og/` images). See `docs/brand.md`.
- Phones get different content (`.mobile-only` / `.desktop-only`, below 760px): there is no
  terminal on a phone, so the landing there only promotes. Commands, Quick start and "Teach
  your agent" are desktop only, and the calls to action send the link to a computer.
- The site is in English and Italian. Every page in `content/` has its Italian twin in
  `content/it/` (same path) and `templates/home.it.html` mirrors `home.html`; the build fails
  if a translation is missing. Italian files:
  - keep English link paths (`/docs/ci/`): the build adds `/it`, and fails if an Italian page
    still links to an English one (only the language switch may);
  - end every heading with the English id, `## Configurazione {#configuration}`, so anchors
    are the same in both languages;
  - UI strings (nav, footer, docs menu, pager) are in `STRINGS` in `build.mjs`, script strings
    in `site.js`.
- Language choice happens in the CloudFront function, before the page loads: a reader who never
  chose goes to `/it/` when the browser's first language (`Accept-Language`) is Italian. The
  switch links carry `?lang=en|it`, which sets the `mestiere_lang` cookie for a year and always
  wins. Crawlers and link-preview fetchers (matched by user agent) are never redirected, so a
  shared link previews exactly the page that was shared; `hreflang` and the sitemap list both.
  `dev.mjs` applies the same rules locally.
- Navigation doesn't reload the page: `site.js` intercepts links to pages of the same language,
  fetches the page (prefetched on hover), swaps `<main>`, the nav state, the title and the head
  links, and keeps history and scroll positions. Other languages, the 404 page, files and any
  error fall back to a normal load. Code that sets up page content goes in `initPage()`.
- Accordions (`<details>`: FAQ, docs menu on phones) open and close with a height animation
  (`initAccordions` in `site.js`); without JavaScript or with reduced motion they snap.
- The three Parmigiano files are preloaded and use `font-display: block`: text never shows in the
  fallback font first.
- CloudFront serves one `404.html` for every missing path. It contains the English and the
  Italian page; `js/not-found.js`, loaded in `<head>` without `defer`, shows the Italian one
  under `/it/` before the first paint, so a reader only ever sees one language.
- Every version shown in the HTML pages is live: write `{{version}}` (content) or
  `{{liveVersionV}}` (templates). The build turns it into a `.live-version` element, which
  `site.js` fills at page load (invisible until then, then a fade-in). Source, in order:
  the Homebrew cask that every release writes (`raw.githubusercontent.com/polarity-dev/homebrew-tap/main/Casks/mestiere.rb`:
  CORS-enabled, 5-minute CDN cache, no API rate limit), then the GitHub API `releases/latest`
  (60 calls an hour per IP), then the build-time version, which is the highest `vX.Y.Z` tag on
  GitHub (`git ls-remote`, so stale local tags don't matter). `github.com/.../releases/latest`
  has no CORS, so browsers can't read it. The CSP allows both hosts in `connect-src`.
- Markdown twins, llms.txt and llms-full.txt never state a version: lines marked
  `<!-- html-only -->` are dropped and other `{{version}}` become `<version>`. Release archives
  are versionless (`mestiere_<os>_<arch>.tar.gz`), so instructions use `releases/latest/download/`.
- The GitHub buttons use the Octicons `mark-github` icon (MIT), served from
  `static/assets/icons/github-mark.svg` and inlined by the build.
- The hero graphic (`hero-graphic.mjs`) is SVG generated at build time, animated with SMIL (no
  script) and still under `prefers-reduced-motion`.
- From the design review (Marco, on Lavagna):
  - copying is always an icon button on the code block, never a text "Copy" button;
  - in a group of buttons the primary one is the last, on the right;
  - "read more" links go on their own line (`<p class="more">`, with →); external links in
    running text get ↗ automatically;
  - code blocks are dark grey (`--code-bg`), not black, with a scrollbar styled to match;
  - on the grey bands (`.section-alt`) body text is full black/white, not muted.
