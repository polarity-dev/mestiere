import type { APIRoute, GetStaticPaths } from "astro"
import { iconSvg, renderPng } from "../lib/images.ts"

const ICON_SIZES = { "favicon-32": 32, "apple-touch-icon": 180, "icon-512": 512 } as const

export const getStaticPaths = (() =>
  Object.entries(ICON_SIZES).map(([icon, size]) => ({
    params: { icon },
    props: { size },
  }))) satisfies GetStaticPaths

export const GET: APIRoute<{ size: number }> = ({ props }) =>
  new Response(renderPng(iconSvg(), props.size), { headers: { "Content-Type": "image/png" } })
