import { execFileSync } from "node:child_process"
import { readdirSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"
import { chromium } from "playwright"

interface Element {
  id: string
  kind: string
  url: string
  x: number
  y: number
  width: number
  height: number
}

const ROOT = dirname(fileURLToPath(import.meta.url))
const DIR = join(ROOT, "dist")

const heights: Record<string, Record<number, number>> = {}
const browser = await chromium.launch()
for (const [w, h] of [
  [1440, 900],
  [390, 844],
] as const) {
  const page = await browser.newPage({ viewport: { width: w, height: h } })
  for (const file of readdirSync(DIR)) {
    await page.goto(`file://${DIR}/${file}`)
    await page.evaluate(() => document.fonts.ready)
    const scrollHeight: number = await page.evaluate(() => document.documentElement.scrollHeight)
    ;(heights[file] ??= {})[w] = Math.max(h, Math.ceil(scrollHeight))
  }

  await page.close()
}

await browser.close()

const cli = (...args: (string | number)[]): string =>
  execFileSync("lavagna", [...args.map(String), "--board", "main", "--yes"], { cwd: ROOT, encoding: "utf8" })
const elements = (JSON.parse(cli("board", "--json")) as { elements: Element[] }).elements
const groups = elements.filter(e => e.kind === "section" || e.kind === "group")
const frames = elements.filter(e => e.kind === "frame")
const inside = (f: Element, g: Element): boolean =>
  f.x >= g.x && f.x < g.x + g.width && f.y >= g.y && f.y < g.y + g.height

let updates = 0
for (const x0 of new Set(groups.map(g => g.x))) {
  const column = groups.filter(g => g.x === x0).sort((a, b) => a.y - b.y)
  let y = column[0]?.y ?? 0
  for (const group of column) {
    const mine = frames.filter(f => inside(f, group))
    let rowY = y + 160
    for (const rowTop of [...new Set(mine.map(f => f.y))].sort((a, b) => a - b)) {
      let tallest = 0
      for (const f of mine.filter(m => m.y === rowTop)) {
        const h = heights[f.url.replace(/^dist\//, "")]?.[f.width === 390 ? 390 : 1440] ?? f.height
        tallest = Math.max(tallest, h)
        if (f.y !== rowY || f.height !== h) {
          cli("board", "update", f.id, "--y", rowY, "--height", h)
          updates++
        }
      }

      rowY += tallest + 160
    }

    const height = rowY - 160 + 120 - y
    if (group.y !== y || group.height !== height) {
      cli("board", "update", group.id, "--y", y, "--height", height)
      updates++
    }

    y += height + 400
  }
}

console.log(`${updates} board elements updated`)
