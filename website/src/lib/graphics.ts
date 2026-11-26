import { MARK_CENTER, MARK_PATH } from "./brand.ts"

const DURATION = 3.2
const START = 0.9
const MARK_SCALE = 0.62

const round = (n: number): number => Math.round(n * 10) / 10
export const escapeXml = (s: string): string =>
  s.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;")

interface Point {
  x: number
  y: number
}

const mark = (source: Point): string =>
  `<path class="loom-mark" transform="translate(${round(source.x - MARK_CENTER.x * MARK_SCALE)} ${round(source.y - MARK_CENTER.y * MARK_SCALE)}) scale(${MARK_SCALE})" d="${MARK_PATH}"/>`

const pulse = (
  d: string,
  begin: number,
  values = "0;1;1;0",
  keyTimes = "0;0.08;0.9;1",
): string => `<circle class="loom-pulse" r="3">
        <animateMotion dur="${DURATION}s" begin="${begin}s" repeatCount="indefinite" path="${d}" calcMode="spline" keyPoints="0;1" keyTimes="0;1" keySplines="0.45 0 0.2 1"/>
        <animate attributeName="opacity" dur="${DURATION}s" begin="${begin}s" repeatCount="indefinite" values="${values}" keyTimes="${keyTimes}"/>
      </circle>`

export function heroGraphic(title: string): string {
  const W = 640
  const H = 460
  const source = { x: 110, y: H / 2, r: 30 }
  const names = ["web", "api", "mobile", "billing", "search", "auth", "infra", "data", "docs"]
  const stagger = 0.41
  const top = 36
  const step = (H - 2 * top) / (names.length - 1)
  const start = { x: source.x + source.r + 6, y: source.y }
  const repos = names.map((name, i) => {
    const y = top + i * step
    const t = (y - H / 2) / (H / 2 - top)
    const x = 470 - 56 * t * t
    const d = `M${start.x} ${start.y} C${start.x + 150} ${start.y} ${round(x - 170)} ${round(y)} ${round(x - 9)} ${round(y)}`
    return {
      name,
      x: round(x),
      y: round(y),
      d,
      begin: round(START + ((i * stagger * 2) % (names.length * stagger))),
    }
  })

  const threads = repos.map(r => `<path class="loom-thread" d="${r.d}"/>`).join("")
  const pulses = repos.map(r => pulse(r.d, r.begin)).join("")
  const nodes = repos
    .map(
      r => `<g class="loom-repo">
        <circle class="loom-node" cx="${r.x}" cy="${r.y}" r="4.5">
          <animate attributeName="r" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="4.5;4.5;8;4.5" keyTimes="0;0.9;0.95;1"/>
        </circle>
        <circle class="loom-ring" cx="${r.x}" cy="${r.y}" r="4.5">
          <animate attributeName="r" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="4.5;4.5;18" keyTimes="0;0.92;1"/>
          <animate attributeName="opacity" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="0;0;0.6;0" keyTimes="0;0.92;0.93;1"/>
        </circle>
        <text class="loom-label" x="${r.x + 16}" y="${r.y}">${r.name}/</text>
      </g>`,
    )
    .join("")

  return `<svg class="loom" viewBox="0 0 ${W} ${H}" role="img" aria-labelledby="loom-title">
        <title id="loom-title">${escapeXml(title)}</title>
        <g class="loom-threads">${threads}</g>
        <g class="loom-pulses">${pulses}</g>
        <circle class="loom-halo" cx="${source.x}" cy="${source.y}" r="${source.r}">
          <animate attributeName="r" dur="${DURATION / 2}s" begin="${START}s" repeatCount="indefinite" values="${source.r};${source.r + 16}"/>
          <animate attributeName="opacity" dur="${DURATION / 2}s" begin="${START}s" repeatCount="indefinite" values="0.25;0"/>
        </circle>
        <circle class="loom-source" cx="${source.x}" cy="${source.y}" r="${source.r}"/>
        ${mark(source)}
        <text class="loom-caption" x="${source.x}" y="${source.y + source.r + 26}">registry</text>
        ${nodes}
      </svg>`
}

export function notFoundGraphic(title: string, idSuffix = ""): string {
  const W = 560
  const H = 420
  const source = { x: 96, y: H / 2, r: 30 }
  const names = ["web", "api", "404", "docs", "infra"]
  const start = { x: source.x + source.r + 6, y: source.y }
  const snap = { x: 300, y: source.y + 3 }
  const titleId = `loom-404-title${idSuffix}`
  const rows = names.map((name, i) => {
    const y = 70 + i * 70
    const t = (y - source.y) / 140
    const x = round(450 - 34 * t * t)
    const d = `M${start.x} ${start.y} C${start.x + 120} ${start.y} ${round(x - 150)} ${y} ${round(x - 9)} ${y}`
    return { name, x, y, d, begin: round(START + i * 0.7), missing: name === "404" }
  })

  const parts = rows
    .map(r => {
      if (r.missing) {
        const head = `M${start.x} ${start.y} C${start.x + 90} ${start.y} ${snap.x - 70} ${snap.y - 2} ${snap.x} ${snap.y}`
        const pivot = `${snap.x + 12} ${snap.y + 4}`
        return `<path class="loom-thread" d="${head}"/>
        <g class="loom-loose">
          <animateTransform attributeName="transform" type="rotate" dur="3.6s" repeatCount="indefinite" calcMode="spline" keyTimes="0;0.5;1" keySplines="0.45 0 0.55 1;0.45 0 0.55 1" values="-7 ${pivot};7 ${pivot};-7 ${pivot}"/>
          <path class="loom-thread loom-thread-loose" d="M${pivot} C${snap.x + 26} ${snap.y + 10} ${snap.x + 34} ${snap.y + 26} ${snap.x + 36} ${snap.y + 46}"/>
        </g>
        ${pulse(head, r.begin, "0;1;1;0", "0;0.1;0.75;1")}
        <circle class="loom-missing" cx="${r.x}" cy="${r.y}" r="6"/>
        <text class="loom-label loom-label-missing" x="${r.x + 16}" y="${r.y}">${r.name}/</text>`
      }

      return `<path class="loom-thread" d="${r.d}"/>
        ${pulse(r.d, r.begin)}
        <circle class="loom-node" cx="${r.x}" cy="${r.y}" r="4.5"/>
        <text class="loom-label" x="${r.x + 16}" y="${r.y}">${r.name}/</text>`
    })
    .join("\n        ")

  return `<svg class="loom loom-404" viewBox="0 0 ${W} ${H}" role="img" aria-labelledby="${titleId}">
        <title id="${titleId}">${escapeXml(title)}</title>
        ${parts}
        <circle class="loom-source" cx="${source.x}" cy="${source.y}" r="${source.r}"/>
        ${mark(source)}
      </svg>`
}
