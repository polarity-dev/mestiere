export const SITE = {
  url: "https://mestiere.io",
  name: "mestiere",
  tagline: "Your team's agent skills, versioned in git and installed into every repository.",
  repo: "https://github.com/polarity-dev/mestiere",
  org: { name: "Polarity", url: "https://polarity.dev" },
} as const

export const LANGS = ["en", "it"] as const
export type Lang = (typeof LANGS)[number]
export const DEFAULT_LANG: Lang = "en"

export const LOCALES: Record<Lang, { prefix: string, locale: string, short: string }> = {
  en: { prefix: "", locale: "en_US", short: "EN" },
  it: { prefix: "/it", locale: "it_IT", short: "IT" },
}

export const otherLang = (lang: Lang): Lang => (lang === "en" ? "it" : "en")

export const isLang = (value: string): value is Lang => (LANGS as readonly string[]).includes(value)

export const pagePath = (lang: Lang, key: string): string => LOCALES[lang].prefix + key

export const isPagePath = (path: string): boolean =>
  path.endsWith("/") || !(path.split("/").pop() ?? "").includes(".")

export function localizeHref(href: string, lang: Lang): string {
  const { prefix } = LOCALES[lang]
  if (!prefix || !href.startsWith("/") || href.startsWith("//")) {
    return href
  }

  const path = href.split(/[?#]/)[0] ?? ""
  if (path === prefix || path.startsWith(`${prefix}/`) || !isPagePath(path)) {
    return href
  }

  return prefix + href
}

export const localizeLinks = (html: string, lang: Lang): string =>
  html.replace(
    /(\shref=")([^"]*)"/g,
    (_, attr: string, href: string) => `${attr}${localizeHref(href, lang)}"`,
  )
