import social from "@brand/mestiere-social.png"
import { UI } from "../i18n/ui.ts"
import { VERSION } from "./git.ts"
import type { Page } from "./pages.ts"
import { LOCALES, SITE } from "./site.ts"

type Node = Record<string, unknown>

const ORG = {
  "@type": "Organization",
  "@id": `${SITE.url}/#organization`,
  name: SITE.org.name,
  url: SITE.org.url,
}

const WEBSITE = {
  "@type": "WebSite",
  "@id": `${SITE.url}/#website`,
  url: `${SITE.url}/`,
  name: SITE.name,
  description: SITE.tagline,
  inLanguage: ["en", "it"],
  publisher: { "@id": ORG["@id"] },
}

const SOFTWARE = {
  "@type": "SoftwareApplication",
  "@id": `${SITE.url}/#software`,
  name: "mestiere",
  alternateName: "mestiere CLI",
  description:
    "Command-line tool that installs Agent Skills from git registries into every repository, pinned to an exact version, for Claude Code, Kiro and other coding agents.",
  applicationCategory: "DeveloperApplication",
  applicationSubCategory: "Command-line tool",
  operatingSystem: "macOS, Linux",
  softwareVersion: VERSION,
  url: `${SITE.url}/`,
  downloadUrl: `${SITE.repo}/releases/latest`,
  installUrl: `${SITE.url}/install/`,
  codeRepository: SITE.repo,
  programmingLanguage: "Go",
  softwareRequirements: "git",
  image: SITE.url + social.src,
  publisher: { "@id": ORG["@id"] },
}

function breadcrumbs(page: Page): Node {
  const prefix = LOCALES[page.lang].prefix
  const items = [{ name: "mestiere", url: `${SITE.url}${prefix}/` }]
  if (page.key.startsWith("/docs/") && page.key !== "/docs/") {
    items.push({ name: UI[page.lang].crumbDocs, url: `${SITE.url}${prefix}/docs/` })
  }

  items.push({ name: page.navTitle, url: page.url })
  return {
    "@type": "BreadcrumbList",
    itemListElement: items.map((it, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: it.name,
      item: it.url,
    })),
  }
}

export const homeGraph = (page: Page, faq: { question: string, text: string }[]): Node[] => [
  ORG,
  WEBSITE,
  SOFTWARE,
  {
    "@type": "WebPage",
    "@id": `${page.url}#webpage`,
    url: page.url,
    name: page.seoTitle,
    description: page.description,
    isPartOf: { "@id": WEBSITE["@id"] },
    about: { "@id": SOFTWARE["@id"] },
    primaryImageOfPage: SITE.url + page.ogImage,
    inLanguage: page.lang,
  },
  {
    "@type": "FAQPage",
    "@id": `${page.url}#faq`,
    mainEntity: faq.map(({ question, text }) => ({
      "@type": "Question",
      name: question,
      acceptedAnswer: { "@type": "Answer", text },
    })),
  },
]

export const docGraph = (page: Page): Node[] => [
  ORG,
  WEBSITE,
  {
    "@type": "TechArticle",
    "@id": `${page.url}#article`,
    headline: page.title,
    description: page.description,
    url: page.url,
    inLanguage: page.lang,
    dateModified: page.modified,
    isPartOf: { "@id": WEBSITE["@id"] },
    about: { "@id": SOFTWARE["@id"] },
    author: { "@id": ORG["@id"] },
    publisher: { "@id": ORG["@id"] },
    image: SITE.url + page.ogImage,
  },
  breadcrumbs(page),
]

export const jsonLd = (graph: Node[]): string =>
  JSON.stringify({ "@context": "https://schema.org", "@graph": graph }).replaceAll("<", "\\u003c")
