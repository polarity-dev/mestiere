# mestiere.io

Website of mestiere: a static [Astro](https://astro.build) site in TypeScript, served from a
private S3 bucket behind CloudFront. No server runs at request time.

```
website/
├── src/
│   ├── content/en/, content/it/   # one Markdown file per page and language
│   ├── i18n/                      # UI strings (ui.ts) and the home page text (home.ts)
│   ├── components/, layouts/      # page chrome, home page, docs page, <head>
│   ├── pages/                     # routes: pages, Markdown twins, OG images, icons, llms.txt
│   ├── lib/                       # content, Markdown plugin, graphics, images, structured data
│   ├── integrations/check-site.ts # post-build checks
│   ├── scripts/site.ts            # the only client script
│   ├── styles/site.css
│   └── assets/                    # fonts and icons bundled by the build
├── public/scripts/not-found.js    # 404 language pick, runs before the first paint
├── infra/site.yaml                # CloudFormation: bucket, OAC, CloudFront, function, headers, deploy role
└── lavagna/                       # review build for Lavagna
```

## Develop

Requires Node 24 or later and pnpm (the version is pinned in `packageManager`).

```sh
cd website
pnpm install
pnpm run dev       # http://localhost:4321
pnpm run lint      # ESLint (eslint.config.ts): TypeScript, Astro components, style
pnpm run check     # astro check: TypeScript and templates
pnpm run build     # dist/; fails on broken links, anchors, language leaks and inline scripts
pnpm run preview   # serve dist/
```

`pnpm run dev` and `pnpm run preview` don't run the CloudFront function: there are no language
redirects locally, and `/it/` pages are reached through the language switch.

Add a page by adding a Markdown file in both languages: `src/content/en/docs/foo.md` and
`src/content/it/docs/foo.md` become `/docs/foo/` and `/it/docs/foo/`, join the docs sidebar (by
`order`), the sitemap, `llms.txt` and `llms-full.txt`. Front matter: `title`, `description`
(160 characters at most, the build fails otherwise), optional `seoTitle` (around 60 characters),
`nav` and `order`. `{{version}}` in content becomes the live version.

## Single sources

- Brand artwork is read from [`docs/assets/`](../docs/assets) through the `@brand` alias: the
  wordmark in the header, the mark in the hero, 404 and icons, `mestiere-social.png` in the
  JSON-LD. Favicons, app icons and link previews are rendered from it at build time.
- Each page is written once, in Markdown. The HTML page, its Markdown twin (`<page>/index.md`),
  `llms-full.txt` and the home page FAQ (HTML and `FAQPage` JSON-LD) all come from it.
- The home page markup is one component, `src/components/HomePage.astro`; its text in both
  languages is in `src/i18n/home.ts`.
- The CloudFront function lives only in `infra/site.yaml`.

## What the build generates

| Output | For |
|---|---|
| `<page>/index.html` | People and search engines: canonical URL, meta description, Open Graph, JSON-LD (`SoftwareApplication`, `FAQPage`, `TechArticle`, `BreadcrumbList`) |
| `assets/og/<lang>/<page>.png` | Link previews (Open Graph / X card, 1200x630), drawn by `src/lib/images.ts` with the page title and description |
| `<page>/index.md` | Agents: the same page as Markdown ([llms.txt](https://llmstxt.org/) convention), linked with `rel="alternate"` |
| `llms.txt`, `llms-full.txt` | Agents: index of the Markdown pages with key facts about the CLI, and every page in one file |
| `sitemap-index.xml`, `sitemap-0.xml` | Search engines ([`@astrojs/sitemap`](https://docs.astro.build/en/guides/integrations-guide/sitemap/)): every page in both languages with its `hreflang` alternates, `lastmod` from each file's last commit |
| `robots.txt` | Everyone allowed, AI crawlers listed explicitly |
| `404.html` | `noindex` error page in both languages |
| `favicon.svg`, `*.png` icons, `site.webmanifest` | Browsers and home screens |

## Deploy

Infrastructure is one CloudFormation stack, `mestiere-website` (`infra/site.yaml`), in eu-west-1:
private bucket, CloudFront, the viewer-request function, response headers and the GitHub deploy
role. The certificate lives in us-east-1, as CloudFront requires, and the public DNS zone lives in
another AWS account. Both are set up once, by hand: request the certificate for `mestiere.io` and
`www.mestiere.io` with DNS validation, then create the stack and point A/AAAA alias records for the
apex and `www` at the distribution.

```sh
aws cloudformation deploy --region eu-west-1 --stack-name mestiere-website \
  --template-file website/infra/site.yaml --capabilities CAPABILITY_NAMED_IAM \
  --parameter-overrides CertificateArn=<arn of the us-east-1 certificate>
```

Every push to `main` that touches the site, or a manual run, builds and publishes it in one job
(`.github/workflows/website.yml`). The job assumes the role in the stack output
`GitHubDeployRoleArn` through GitHub OIDC, set as the repository variable
`WEBSITE_DEPLOY_ROLE_ARN`. The role trusts only `refs/heads/main` of this repository and can only
upload to the bucket and invalidate the distribution. New files are uploaded first with their
`Cache-Control` and content type, stale ones are deleted last, then the whole cache is invalidated.

CloudFront gives every page one URL: `www.mestiere.io` → `mestiere.io`, `/docs` → `/docs/`,
`/docs/index.html` → `/docs/`, all 301. Markdown and `llms*.txt` files are served with
`X-Robots-Tag: noindex` and open CORS, so agents can read them without competing with the HTML
pages in search results. Files under `/_astro/` have a content hash in their name and are cached
for a year.

## Review on Lavagna

`lavagna/build.ts` turns `dist/` into `lavagna/dist/`: one self-contained HTML file per page and
state, in light and dark (CSS, fonts, images and script inlined, pages flattened to names like
`docs-ci.html`). `lavagna/layout.ts` resizes the board frames to the page heights in
Playwright's Chromium.

```sh
cd website && pnpm run build && pnpm run lavagna
(cd lavagna && lavagna push --yes)
pnpm exec playwright install chromium   # once
pnpm run lavagna:layout
```

## Rules for the pages

- Everything is self-hosted: no CDNs. The Content-Security-Policy in `infra/site.yaml` allows
  scripts only from the site's own files: no inline `<script>` (JSON-LD data is fine), the build
  checks it. Inline styles are allowed (the fonts API writes its `@font-face` rules inline).
  Astro's hash-based CSP (`security.csp`) isn't used: it doesn't support `<ClientRouter />`.
- Brand: black and white only. Headings and lead text use Parmigiano Text Light, h3 Parmigiano
  Text Bold, body text the system sans. Parmigiano is a licensed Typotheque font: only the woff2
  web files ship, in `src/assets/fonts/`, converted from the OTFs in `docs/assets/fonts/`, which
  git ignores. The Astro fonts API (`fonts` in `astro.config.ts`) declares them, with
  metric-matched fallbacks; the three files are preloaded and use `font-display: block`.
- Phones get different content (`.mobile-only` / `.desktop-only`, below 760px): there is no
  terminal on a phone, so the landing there only promotes. Commands, Quick start and "Teach
  your agent" are desktop only.
- The site is in English and Italian; the build fails if a page has no translation. Italian
  content:
  - keeps English link paths (`/docs/ci/`): the build adds `/it`, and fails if an Italian page
    still links to an English one (only the language switch may);
  - ends every heading with the English id, `## Configurazione {#configuration}`, so anchors are
    the same in both languages.
- Language choice happens in the CloudFront function, before the page loads: a reader who never
  chose goes to `/it/` when the browser's first language is Italian. The switch links carry
  `?lang=en|it`, which sets the `mestiere_lang` cookie for a year and always wins. Crawlers and
  link-preview fetchers are never redirected, so a shared link previews the page that was shared.
- Navigation doesn't reload the page: Astro's `<ClientRouter />` (in `Document.astro`) swaps
  the page in place with a view transition, prefetches links on hover, and keeps history,
  scroll and focus. The language switch is a full load (`data-astro-reload`) and is never
  prefetched, since `?lang=` sets the cookie. Code that sets up page content runs on the
  `astro:page-load` event (`src/scripts/site.ts`).
- The phone menu is a popover (`popover` / `popovertarget`): the browser opens and closes it,
  handles Escape and taps outside, and sets `aria-expanded`. Accordions are plain `<details>`,
  animated by CSS where `interpolate-size` is supported.
- CloudFront serves one `404.html` for every missing path. It holds the English and the Italian
  page; `public/scripts/not-found.js`, loaded in `<head>` without `defer`, shows the Italian one
  under `/it/` before the first paint.
- Every version shown in the HTML pages is live (`liveVersion()` in templates, `{{version}}` in
  content). `site.ts` fills it at page load from the Homebrew cask that every release writes
  (CORS-enabled, no API rate limit), then the GitHub API `releases/latest`, then the build-time
  version, the highest `vX.Y.Z` tag on GitHub. The CSP allows both hosts in `connect-src`.
- Markdown twins, `llms.txt` and `llms-full.txt` never state a version: lines marked
  `<!-- html-only -->` are dropped and other `{{version}}` become `<version>`. Release archives
  are versionless, so instructions use `releases/latest/download/`.
- The hero and 404 graphics (`src/lib/graphics.ts`) are SVG built at build time, animated with
  SMIL (no script) and still under `prefers-reduced-motion`.
- From the design review:
  - copying is always an icon button on the code block, never a text "Copy" button;
  - in a group of buttons the primary one is the last, on the right;
  - "read more" links go on their own line (`<p class="more">`, with →); external links in
    running text get ↗ automatically;
  - code blocks are dark grey (`--code-bg`), not black, with a scrollbar styled to match;
  - on the grey bands (`.section-alt`) body text is full black/white, not muted.
