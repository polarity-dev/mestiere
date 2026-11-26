import markSvg from "@brand/mestiere-mark-black.svg?raw"
import wordmarkSvg from "@brand/mestiere-wordmark-black.svg?raw"

const viewBoxOf = (svg: string): [number, number, number, number] => {
  const box = /viewBox="([^"]+)"/.exec(svg)?.[1]?.split(/\s+/).map(Number)
  if (box?.length !== 4) {
    throw new Error("brand SVG without a viewBox")
  }

  return box as [number, number, number, number]
}

export const MARK_PATH = (() => {
  const paths = [...markSvg.matchAll(/<path[^>]*\sd="([^"]+)"/g)].map(m => m[1])
  const [path] = paths
  if (paths.length !== 1 || !path) {
    throw new Error("docs/assets/mestiere-mark-black.svg: expected exactly one path")
  }

  return path
})()

export const WORDMARK_INNER = wordmarkSvg.replace(/^[\s\S]*?<svg[^>]*>/, "").replace(/<\/svg>\s*$/, "")
export const WORDMARK_VIEWBOX = viewBoxOf(wordmarkSvg)

export const MARK_CENTER = { x: 43.68, y: 44.5 }
