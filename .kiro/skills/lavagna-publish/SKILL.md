---
name: lavagna-publish
description: Publish the mestiere.io website (or any static page) on Lavagna for review. Use when asked to upload, push, update or show the landing/site/pages on Lavagna (project polarity/mestiere, rigger-viewer.polarity.dev), or when a page on a Lavagna frame shows without styles, fonts or images.
---
# Publishing pages on Lavagna

Lavagna (ex Rigger) shows pages in frames: a frame's url is a file under the project's `dist/`.
**Lavagna always gets a dedicated build where every page is one self-contained HTML file.**
Never push the production build as is: root-relative URLs (`/css/site.css`), folder URLs
(`docs/`) and separate asset files do not work reliably inside frames.

## Rules of the Lavagna build
- One `.html` file per page, flat names: `docs/ci/index.html` -> `docs-ci.html`, `/` -> `index.html`.
- Inline everything the page needs:
  - CSS in a `<style>`, with fonts as `data:font/woff2;base64,…` inside `url()`;
  - `<img src>` and `<source srcset>` as `data:` URIs (SVG, PNG);
  - JS in a `<script>` at the end of `<body>` (an inline script cannot be `defer`);
  - the SVG favicon as a `data:` URI.
- Drop head links that need another file or only matter on the real site: preload, manifest,
  sitemap, alternate/describedby (markdown, llms.txt), PNG and apple-touch icons.
- Links between pages point at the flat names. Links to non-page files (`llms.txt`, `*.md`) go
  to the live site (`https://mestiere.io/...`). External links stay as they are.
- The build fails if any root-relative `href`/`src`/`srcset`/`url()` is left.
- Each file stays under 10 MiB (the push limit). The two Parmigiano fonts make about 400 KiB per page.

## mestiere.io
- Script: `website/lavagna.mjs` (`npm run lavagna`), reads `website/dist`, writes `website/dist-lavagna` (gitignored).
- Config: `website/lavagna.config.json` -> project `polarity/mestiere`, dist `dist-lavagna`.
- Every screen is on the board, and every page is built twice so frames never depend on the
  reviewer's OS theme: `<name>.html` forces light, `<name>-dark.html` forces dark (the
  `prefers-color-scheme: dark` query in CSS and `<source media>` become `not all` / `all`).
  Dark pages link to dark pages.
- States not reachable by URL are extra files from the `states` table in `lavagna.mjs`
  (today `index-faq-open[-dark].html`, all FAQ `<details>` open). Add new states there.
- Italian pages flatten with an `it` prefix: `/it/` -> `it.html`, `/it/docs/ci/` -> `it-docs-ci.html`,
  states `it-faq-open.html`, `it-docs-menu-open.html`. On the board the Italian column starts at
  x 6420 with the same layout, ids `it-<id>` and groups `section-it-<id>`. The single 404.html
  holds both languages (js/not-found.js picks one by path); on Lavagna the Italian one is the
  state file `404-it.html`.
- Board `main` layout, one group `section-<id>` per page, stacked at x=0 with 400 between groups,
  width 4520, frames 160 from the group top:
  - per row: light desktop 1440 (x 120), light smartphone 390 (x 1720), dark desktop (x 2270),
    dark smartphone (x 3870); frame ids `<id>`, `<id>-mobile`, `<id>-dark`, `<id>-dark-mobile`;
  - frames are full-page high (measured scrollHeight at 1440 and 390 wide, e.g. with Playwright);
  - groups: landing (plus a `landing-faq-open` row), install, docs, not-found;
  - docs: only the Getting started page (`docs`), light and dark. The other doc pages share its
    layout, so Giovanni doesn't want them as frames. Their files stay published because the docs
    nav inside the frame links to them;
  - note `llms-txt` (markdown, from `website/dist/llms.txt`) at x 4920, y 0.
- After pages change height: `node lavagna-layout.mjs` (in website/) measures every page and resizes and restacks all frames and groups of the board. It is idempotent: rerun it after a network error.
- When the llms.txt changes: `lavagna board update llms-txt --text-file dist/llms.txt --board main --yes`.

```sh
cd website
npm run build && npm run lavagna
lavagna push --yes                 # uploads dist-lavagna/* to the project's dist/
lavagna board --board main --yes   # check frames
```

New page on the board: `lavagna board create frame --id <id> --title "<Title>" --url dist/<flat-name>.html --x … --y … --width 1440 --height 900 --board main --yes`.
Read the current geometry first (`lavagna board --board main`): Giovanni moves frames by hand.

## Cleanup
`lavagna push` never deletes remote files. When pages are renamed or the layout of `dist-lavagna`
changes, compare `lavagna files --json` with the local folder and delete only the stale files of
this project with `lavagna files delete <path>... --yes` (no undo; dist/ is shared by every board
of the project, so check no other board's frame points at them).

## CLI
`lavagna` from the Homebrew cask `lavagna-ai/tap/lavagna`, logged in with `lavagna login`
(check with `lavagna whoami`). Pass `--board <slug>` explicitly: the default is the first board.

## Review comments
- Read them with `lavagna comments --board main --json` (filter `--mention ai` for the ones addressed to the agent).
- A comment's `x`/`y` are board coordinates. On a frame, the point in the page is
  `(x - frame.x, y - frame.y)` at the frame's width. Find the element there with a headless browser
  (`document.elementFromPoint` after `scrollTo({ behavior: 'instant' })`: the site uses smooth scroll).
- Comments stay at their board coordinates when a frame moves: if you moved frames after the
  comment was written, use the frame's old position.
- The page may have changed since the comment was written (`createdAt` older than the last push):
  then match it by its meaning, not only by position.
- Fix in the site sources (`website/templates`, `static`, `content`), never in `dist-lavagna`.
  Rebuild, push, update the frame heights if pages got taller or shorter, then reply to each
  thread with what changed (in Italian) and resolve it.
- Re-read the comments before finishing: reviewers keep adding them while you work.
- The smartphone frames (390 wide) show the phone version of the content (`.mobile-only`):
  check those frames separately when a comment touches one of them.
