import { mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { dirname, extname, join, relative } from "node:path"
import { fileURLToPath } from "node:url"

const ROOT = dirname(dirname(fileURLToPath(import.meta.url)))
const SRC = join(ROOT, "dist")
const OUT = join(ROOT, "lavagna/dist")
const SITE = "https://mestiere.io"
const MIME: Record<string, string> = { ".svg": "image/svg+xml", ".png": "image/png", ".woff2": "font/woff2" }

const read = (url: string): Buffer => readFileSync(join(SRC, url.replace(/^\//, "")))
const dataUri = (url: string): string => {
  const type = MIME[extname(url)]
  if (!type) {
    throw new Error(`no MIME type for ${url}`)
  }

  return `data:${type};base64,${read(url).toString("base64")}`
}

const pageName = (path: string): string => {
  const clean = path
    .replace(/^\//, "")
    .replace(/(^|\/)index\.html$/, "$1")
    .replace(/\.html$/, "")
    .replace(/\/$/, "")
  return clean ? `${clean.replaceAll("/", "-")}.html` : "index.html"
}

const pages = readdirSync(SRC, { recursive: true, encoding: "utf8" })
  .filter(file => file.endsWith(".html"))
  .map(file => `/${file}`)
const pageSet = new Set(pages.map(pageName))

const link = (url: string): string => {
  const [, path = "", suffix = ""] = /^([^?#]*)(.*)$/.exec(url) ?? []
  if ((path.endsWith("/") || path.endsWith(".html")) && pageSet.has(pageName(path))) {
    return pageName(path) + suffix
  }

  return SITE + url
}

const themes = {
  light: { suffix: "", media: "not all", scheme: "light" },
  dark: { suffix: "-dark", media: "all", scheme: "dark" },
}

type Theme = (typeof themes)[keyof typeof themes]
const css = (url: string, theme: Theme): string => {
  const source = read(url).toString("utf8")
  if (!/@media\s*\(prefers-color-scheme:\s*dark\)/.test(source)) {
    throw new Error(`${url}: dark-mode media query not found`)
  }

  return `${source
    .replace(/@media\s*\(prefers-color-scheme:\s*dark\)/g, `@media ${theme.media}`)
    .replace(
      /url\("?(\/[^")]+)"?\)/g,
      (_, asset: string) => `url("${dataUri(asset)}")`,
    )}\n:root { color-scheme: ${theme.scheme}; }`
}

const withSuffix = (name: string, suffix: string): string => name.replace(/\.html$/, `${suffix}.html`)

type State = { suffix: string, apply: (html: string) => string }
const homeStates: State[] = [
  {
    suffix: "-faq-open",
    apply: html => html.replaceAll('<details class="faq-item">', '<details class="faq-item" open>'),
  },
  {
    suffix: "-menu-open",
    apply: html =>
      html.replace("</body>", "<script>document.querySelector('.nav-panel')?.showPopover();</script></body>"),
  },
]

const docsStates: State[] = [
  {
    suffix: "-menu-open",
    apply: html => html.replace(/(<nav class="docs-menu"[^>]*>\s*<details)>/, "$1 open>"),
  },
]

const states: Record<string, State[]> = {
  "index.html": homeStates,
  "it.html": homeStates,
  "docs.html": docsStates,
  "it-docs.html": docsStates,
  "404.html": [
    {
      suffix: "-it",
      apply: html =>
        html
          .replace('<html lang="en">', '<html lang="it" class="is-it">')
          .replace(
            /<title>[^<]*<\/title>/,
            `<title>${/class="page-lang page-it"[^>]*data-title="([^"]+)"/.exec(html)?.[1] ?? ""}</title>`,
          ),
    },
  ],
}

const inline = (html: string, theme: Theme): string => {
  const scripts: string[] = []
  const out = html
    .replaceAll('media="(prefers-color-scheme: dark)"', `media="${theme.media}"`)
    .replace(
      /(<style>)([\s\S]*?)(<\/style>)/g,
      (_, open: string, body: string, close: string) =>
        `${open}${body.replace(/url\("?(\/[^")]+)"?\)/g, (_m: string, asset: string) => `url("${dataUri(asset)}")`)}${close}`,
    )
    .replace(/\s*<link rel="(?:preload|manifest|sitemap|describedby|alternate|apple-touch-icon)"[^>]*>/g, "")
    .replace(/\s*<link rel="icon" type="image\/png"[^>]*>/g, "")
    .replace(
      /<link rel="icon" type="image\/svg\+xml" href="(\/[^"]+)">/,
      (_, url: string) => `<link rel="icon" type="image/svg+xml" href="${dataUri(url)}">`,
    )
    .replace(
      /<link rel="stylesheet" href="(\/[^"]+)">/g,
      (_, url: string) => `<style>\n${css(url, theme)}\n</style>`,
    )
    .replace(/\s*<script src="\/scripts\/not-found\.js"><\/script>/, "")
    .replace(/\s*<script type="module" src="(\/[^"]+)"><\/script>/g, (_, url: string) => {
      scripts.push(read(url).toString("utf8"))
      return ""
    })
    .replace(
      /(<(?:img|source)\b[^>]*\s(?:src|srcset)=")(\/(?!\/)[^"]+)"/g,
      (_, before: string, url: string) => `${before}${dataUri(url)}"`,
    )
    .replace(/(<a\b[^>]*\shref=")(\/(?!\/)[^"]*)"/g, (_, before: string, url: string) => {
      const target = link(url)
      return `${before}${target.startsWith("http") ? target : target.replace(/^([^#?]+)\.html/, `$1${theme.suffix}.html`)}"`
    })
    .replace(
      "</body>",
      () => `${scripts.map(js => `<script type="module">\n${js}\n</script>`).join("\n")}\n</body>`,
    )
  const left = out.match(/\s(?:href|src|srcset)="\/(?!\/)[^"]*"|url\("\/[^"]*"\)/g)
  if (left) {
    throw new Error(`root-relative URLs left: ${left.join(", ")}`)
  }

  return out
}

rmSync(OUT, { recursive: true, force: true })
mkdirSync(OUT, { recursive: true })
const written: string[] = []
for (const page of pages) {
  const name = pageName(page)
  const html = readFileSync(join(SRC, page), "utf8")
  for (const theme of Object.values(themes)) {
    const write = (file: string, content: string): void => {
      writeFileSync(join(OUT, file), inline(content, theme))
      written.push(file)
    }

    write(withSuffix(name, theme.suffix), html)
    for (const state of states[name] ?? []) {
      const stated = state.apply(html)
      if (stated === html) {
        throw new Error(`${name}: state ${state.suffix} changed nothing`)
      }

      write(withSuffix(name, `${state.suffix}${theme.suffix}`), stated)
    }
  }
}

console.log(`${written.length} self-contained pages -> ${relative(process.cwd(), OUT) || OUT}`)
