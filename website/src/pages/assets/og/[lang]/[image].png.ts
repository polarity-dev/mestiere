import type { APIRoute, GetStaticPaths } from "astro"
import { UI } from "../../../../i18n/ui.ts"
import { homeOgSvg, OG_WIDTH, pageOgSvg, renderPng } from "../../../../lib/images.ts"
import { getAllPages, ogEyebrow, pageSlug, type Page } from "../../../../lib/pages.ts"

export const getStaticPaths = (async() =>
  (await getAllPages()).map(page => ({
    params: { lang: page.lang, image: pageSlug(page.key) },
    props: { page },
  }))) satisfies GetStaticPaths

export const GET: APIRoute<{ page: Page }> = ({ props: { page } }) => {
  const svg =
    page.key === "/"
      ? homeOgSvg(UI[page.lang].ogTagline)
      : pageOgSvg({
        eyebrow: ogEyebrow(page),
        title: page.title,
        description: page.description,
        url: page.url.replace(/^https:\/\//, ""),
      })
  return new Response(renderPng(svg, OG_WIDTH), { headers: { "Content-Type": "image/png" } })
}
