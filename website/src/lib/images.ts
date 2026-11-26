import { mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { join } from "node:path"
import { Resvg } from "@resvg/resvg-js"
import opentype from "opentype.js"
import wawoff2 from "wawoff2"
import { MARK_CENTER, MARK_PATH, WORDMARK_INNER, WORDMARK_VIEWBOX } from "./brand.ts"
import { escapeXml } from "./graphics.ts"

export const OG_WIDTH = 1200
export const OG_HEIGHT = 630

const FAMILY = "Parmigiano Text L"
const FONTS_DIR = join(process.cwd(), "src/assets/fonts")

const FONT_CACHE = join(process.cwd(), "node_modules/.cache/mestiere-fonts")
mkdirSync(FONT_CACHE, { recursive: true })

const fontFiles: string[] = []

const loadFont = async(name: string): Promise<opentype.Font> => {
  // Copied out: wawoff2 returns a view on WebAssembly memory that its next call overwrites
  const font = Buffer.from(
    await wawoff2.decompress(readFileSync(join(FONTS_DIR, `parmigiano-text-${name}.woff2`))),
  )

  const file = join(FONT_CACHE, `parmigiano-text-${name}.otf`)
  writeFileSync(file, font)
  fontFiles.push(file)
  return opentype.parse(font.buffer.slice(font.byteOffset, font.byteOffset + font.byteLength))
}

const metrics = {
  light: await loadFont("light"),
  italic: await loadFont("light-italic"),
  bold: await loadFont("bold"),
}

interface TextStyle {
  size: number
  weight?: number
  italic?: boolean
}

const textAttrs = ({ size, weight = 300, italic = false }: TextStyle): string =>
  `font-family="${FAMILY}" font-weight="${weight}"${italic ? ' font-style="italic"' : ""} font-size="${size}" fill="#000"`

const measure = (text: string, { size, weight = 300, italic = false }: TextStyle): number =>
  metrics[weight >= 600 ? "bold" : italic ? "italic" : "light"].getAdvanceWidth(text, size, { kerning: true })

function wrap(text: string, style: TextStyle, maxWidth: number, maxLines: number): string[] {
  const lines: string[] = []
  let line = ""
  for (const word of text.split(/\s+/)) {
    const candidate = line ? `${line} ${word}` : word
    if (!line || measure(candidate, style) <= maxWidth) {
      line = candidate
      continue
    }

    lines.push(line)
    line = word
    if (lines.length === maxLines) {
      let last = lines.pop() ?? ""
      while (last.includes(" ") && measure(`${last}…`, style) > maxWidth) {
        last = last.replace(/\s+\S+$/, "")
      }

      lines.push(`${last.replace(/[\s,.;:]+$/, "")}…`)
      return lines
    }
  }

  if (line) {
    lines.push(line)
  }

  return lines
}

const lines = (texts: string[], x: number, y: number, lineHeight: number, anchor = "start"): string =>
  texts
    .map((t, i) => `<text x="${x}" y="${y + i * lineHeight}" text-anchor="${anchor}">${escapeXml(t)}</text>`)
    .join("")

const wordmark = (x: number, y: number, width: number): string => {
  const [, , w, h] = WORDMARK_VIEWBOX
  return `<svg x="${x}" y="${y}" width="${width}" height="${((width * h) / w).toFixed(2)}" viewBox="${WORDMARK_VIEWBOX.join(" ")}">${WORDMARK_INNER}</svg>`
}

const frame = (
  content: string,
  label: string,
): string => `<svg xmlns="http://www.w3.org/2000/svg" width="${OG_WIDTH}" height="${OG_HEIGHT}" viewBox="0 0 ${OG_WIDTH} ${OG_HEIGHT}" role="img" aria-label="${escapeXml(label)}">
<rect width="${OG_WIDTH}" height="${OG_HEIGHT}" fill="#fff"/>
${content}
</svg>`

export function homeOgSvg(tagline: string): string {
  const width = 620
  const style = { size: 44, italic: true }
  return frame(
    `${wordmark((OG_WIDTH - width) / 2, 170, width)}
<g ${textAttrs(style)}>${lines(wrap(tagline, style, 1000, 2), OG_WIDTH / 2, 430, 58, "middle")}</g>`,
    tagline,
  )
}

export function pageOgSvg({
  eyebrow,
  title,
  description,
  url,
}: {
  eyebrow: string
  title: string
  description: string
  url: string
}): string {
  const M = 80
  const maxWidth = OG_WIDTH - 2 * M
  const titleStyle = { size: 92 }
  const descStyle = { size: 34, italic: true }
  const titleLines = wrap(title, titleStyle, maxWidth, 2)
  const titleY = 225
  const descY = titleY + (titleLines.length - 1) * 100 + 78
  const descLines = wrap(description, descStyle, maxWidth, titleLines.length > 1 ? 2 : 3)
  return frame(
    `<g ${textAttrs({ size: 28, italic: true })}><text x="${M}" y="118">${escapeXml(eyebrow)}</text></g>
<g ${textAttrs(titleStyle)}>${lines(titleLines, M, titleY, 100)}</g>
<g ${textAttrs(descStyle)}>${lines(descLines, M, descY, 48)}</g>
<rect x="${M}" y="508" width="${maxWidth}" height="1.5" fill="#000"/>
${wordmark(M, 532, 170)}
<g ${textAttrs({ size: 26, italic: true })}><text x="${OG_WIDTH - M}" y="566" text-anchor="end">${escapeXml(url)}</text></g>`,
    `${title}: ${description}`,
  )
}

export function iconSvg(): string {
  const side = 112
  const x = MARK_CENTER.x - side / 2
  const y = MARK_CENTER.y - side / 2
  return `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="${x} ${y} ${side} ${side}">
<rect x="${x}" y="${y}" width="${side}" height="${side}" fill="#000"/>
<path d="${MARK_PATH}" fill="#fff"/>
</svg>
`
}

export const renderPng = (svg: string, width: number): Uint8Array<ArrayBuffer> =>
  new Uint8Array(
    new Resvg(svg, {
      font: { fontFiles, loadSystemFonts: false, defaultFontFamily: FAMILY },
      fitTo: { mode: "width", value: width },
    })
      .render()
      .asPng(),
  )
