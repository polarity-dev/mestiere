import { getCollection, type CollectionEntry } from "astro:content"
import { UI } from "../i18n/ui.ts"
import { lastModified } from "./git.ts"
import { LANGS, pagePath, SITE, type Lang } from "./site.ts"

export interface Page {
  lang: Lang
  key: string
  path: string
  url: string
  title: string
  seoTitle: string
  navTitle: string
  description: string
  order: number
  body: string
  modified: string
  ogImage: string
  entry: CollectionEntry<"pages">
}

export const pageSlug = (key: string): string =>
  key === "/" ? "home" : key.slice(1, -1).replaceAll("/", "-")

export const ogImagePath = (lang: Lang, key: string): string => `/assets/og/${lang}/${pageSlug(key)}.png`

export const ogEyebrow = (page: Page): string => {
  const t = UI[page.lang]
  return page.key.startsWith("/docs/") ? t.docs : page.key === "/install/" ? t.installEyebrow : "mestiere"
}

export const ogImageAlt = (page: Page): string =>
  page.key === "/" ? `mestiere: ${UI[page.lang].ogTagline}` : `mestiere, ${ogEyebrow(page)}: ${page.title}`

function toPage(entry: CollectionEntry<"pages">): Page {
  const [lang, ...rest] = entry.id.split("/") as [Lang, ...string[]]
  const rel = rest.join("/")
  const key = rel === "index" ? "/" : `/${rel.replace(/(^|\/)index$/, "")}/`.replace("//", "/")
  return {
    lang,
    key,
    path: pagePath(lang, key),
    url: SITE.url + pagePath(lang, key),
    title: entry.data.title,
    seoTitle: entry.data.seoTitle ?? entry.data.title,
    navTitle: entry.data.nav ?? entry.data.title,
    description: entry.data.description,
    order: entry.data.order,
    body: entry.body ?? "",
    modified: lastModified(entry.filePath ?? ""),
    ogImage: ogImagePath(lang, key),
    entry,
  }
}

export async function getPages(): Promise<Record<Lang, Page[]>> {
  const all = (await getCollection("pages")).map(toPage).sort((a, b) => a.order - b.order)
  const byLang = Object.fromEntries(LANGS.map(lang => [lang, all.filter(p => p.lang === lang)])) as Record<
    Lang,
    Page[]
  >
  const keys = new Set(all.map(p => p.key))
  const missing = LANGS.flatMap(lang =>
    [...keys].filter(key => !byLang[lang].some(p => p.key === key)).map(key => lang + key))
  if (missing.length) {
    throw new Error(`pages missing a translation: ${missing.join(", ")}`)
  }

  return byLang
}

export const getAllPages = async(): Promise<Page[]> => Object.values(await getPages()).flat()

export const docPages = (pages: Page[]): Page[] => pages.filter(p => p.key !== "/")
