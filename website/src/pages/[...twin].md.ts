import type { APIRoute, GetStaticPaths } from "astro"
import { markdownTwin } from "../lib/markdown.ts"
import { getAllPages, type Page } from "../lib/pages.ts"

export const getStaticPaths = (async() =>
  (await getAllPages()).map(page => ({
    params: { twin: `${page.path.slice(1)}index` },
    props: { page },
  }))) satisfies GetStaticPaths

export const GET: APIRoute<{ page: Page }> = ({ props }) => new Response(markdownTwin(props.page))
