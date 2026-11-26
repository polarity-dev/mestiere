import { createSatteriMarkdownProcessor } from "@astrojs/markdown-satteri"
import { markdownOptions } from "./markdown-plugin.ts"
import type { Page } from "./pages.ts"
import { localizeHref, localizeLinks, SITE, type Lang } from "./site.ts"

let processor: ReturnType<typeof createSatteriMarkdownProcessor> | undefined

export async function renderMarkdown(markdown: string, lang: Lang): Promise<string> {
  processor ??= createSatteriMarkdownProcessor(markdownOptions)
  return localizeLinks((await (await processor).render(markdown)).code, lang)
}

const versionless = (md: string): string =>
  md
    .split("\n")
    .filter(line => !line.includes("<!-- html-only -->"))
    .join("\n")
    .replaceAll("{{version}}", "<version>")

const stripHeadingIds = (md: string): string => md.replace(/[ \t]*\{#[a-z0-9-]+\}[ \t]*$/gm, "")

const toTwinLinks = (md: string, page: Page): string =>
  md.replace(/\]\((\/[^)#\s]*)(#[^)\s]*)?\)/g, (_, path: string, hash = "") => {
    const localized = localizeHref(path, page.lang)
    return `](${SITE.url}${localized}${localized.endsWith("/") ? "index.md" : ""}${hash})`
  })

export const markdownTwin = (page: Page): string =>
  `# ${page.title}\n\n> ${page.description}\n\n${toTwinLinks(stripHeadingIds(versionless(page.body)), page).trim()}\n`

export const htmlToText = (html: string): string =>
  html
    .replace(/<[^>]+>/g, "")
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">")
    .replaceAll("&quot;", '"')
    .replaceAll("&#39;", "'")
    .replaceAll("&amp;", "&")
    .trim()

export interface Faq {
  question: string
  answer: string
}

export function extractFaq(markdown: string): Faq[] {
  const section = markdown.split(
    /^## (?:Frequently asked questions|.*\{#frequently-asked-questions\})\s*$/m,
  )[1]
  if (!section) {
    return []
  }

  return section
    .split(/^### /m)
    .slice(1)
    .map(chunk => {
      const [question = "", ...rest] = chunk.split("\n")
      return { question: question.trim(), answer: (rest.join("\n").split(/^## /m)[0] ?? "").trim() }
    })
}
