import { existsSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { satteri } from "@astrojs/markdown-satteri"
import sitemap from "@astrojs/sitemap"
import { defineConfig, fontProviders } from "astro/config"
import { checkSiteIntegration } from "./src/integrations/check-site.ts"
import { lastModified } from "./src/lib/git.ts"
import { markdownOptions } from "./src/lib/markdown-plugin.ts"
import { DEFAULT_LANG, LANGS, LOCALES, SITE } from "./src/lib/site.ts"

const brand = fileURLToPath(new URL("../docs/assets", import.meta.url))

function contentFile(url: string): string {
  const path = new URL(url).pathname
  const lang = LANGS.find(l => LOCALES[l].prefix && path.startsWith(`${LOCALES[l].prefix}/`)) ?? DEFAULT_LANG
  const rest = path.slice(LOCALES[lang].prefix.length)
  const base = `src/content/${lang}${rest.replace(/\/$/, "") || "/index"}`
  return existsSync(`${base}.md`) ? `${base}.md` : `${base}/index.md`
}

const fontVariant = (file: string, weight: number, style: "normal" | "italic") => ({
  src: [`./src/assets/fonts/parmigiano-text-${file}.woff2`] as [string],
  weight,
  style,
})

export default defineConfig({
  site: SITE.url,
  trailingSlash: "always",
  compressHTML: true,
  devToolbar: { enabled: false },
  fonts: [
    {
      provider: fontProviders.local(),
      name: "Parmigiano Text",
      cssVariable: "--font-parmigiano",
      fallbacks: ["Georgia", "serif"],
      display: "block",
      options: {
        variants: [
          fontVariant("light", 300, "normal"),
          fontVariant("light-italic", 300, "italic"),
          fontVariant("bold", 700, "normal"),
        ],
      },
    },
  ],
  markdown: {
    syntaxHighlight: false,
    processor: satteri(markdownOptions),
  },
  integrations: [
    sitemap({
      i18n: { defaultLocale: DEFAULT_LANG, locales: Object.fromEntries(LANGS.map(lang => [lang, lang])) },
      serialize: item => {
        const links = item.links ?? []
        const fallback = links.find(link => link.lang === DEFAULT_LANG)
        return {
          ...item,
          lastmod: lastModified(contentFile(item.url)),
          links: fallback ? [...links, { lang: "x-default", url: fallback.url }] : links,
        }
      },
    }),
    checkSiteIntegration(),
  ],
  vite: {
    resolve: { alias: { "@brand": brand } },
    build: { assetsInlineLimit: 0 },
    server: { fs: { allow: [".", brand] } },
  },
})
