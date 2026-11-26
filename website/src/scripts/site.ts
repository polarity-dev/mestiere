const IT = document.documentElement.lang === "it"
const T = IT
  ? { copy: "Copia il codice negli appunti", copied: "Copiato", failed: "Copia non riuscita" }
  : { copy: "Copy code to clipboard", copied: "Copied", failed: "Copy failed" }
const SVG_NS = "http://www.w3.org/2000/svg"
const ICONS = {
  copy: ["M9 9h11v11H9z", "M5 15H4V4h11v1"],
  done: ["M5 12.5l4.5 4.5L19 7.5"],
  failed: ["M6 6l12 12", "M18 6L6 18"],
}

type IconName = keyof typeof ICONS

const icon = (name: IconName): SVGSVGElement => {
  const svg = document.createElementNS(SVG_NS, "svg")
  svg.setAttribute("viewBox", "0 0 24 24")
  svg.setAttribute("aria-hidden", "true")
  for (const d of ICONS[name]) {
    const path = document.createElementNS(SVG_NS, "path")
    path.setAttribute("d", d)
    svg.appendChild(path)
  }

  return svg
}

function initCopyButtons(): void {
  if (!("clipboard" in navigator)) {
    return
  }

  document.querySelectorAll<HTMLElement>("pre > code").forEach(code => {
    const pre = code.parentElement
    if (!pre) {
      return
    }

    const wrapper = document.createElement("div")
    wrapper.className = "code-block"
    if (code.textContent.trim().includes("\n")) {
      wrapper.classList.add("is-multiline")
    }

    pre.replaceWith(wrapper)
    wrapper.appendChild(pre)
    const button = document.createElement("button")
    const show = (name: IconName, label: string): void => {
      button.replaceChildren(icon(name))
      button.setAttribute("aria-label", label)
    }

    button.type = "button"
    button.className = "copy-button"
    show("copy", T.copy)
    button.addEventListener("click", async() => {
      const text = code.innerText
        .split("\n")
        .map(line => line.replace(/\s+#\s.*$/, ""))
        .join("\n")
        .trim()
      try {
        await navigator.clipboard.writeText(text)
        show("done", T.copied)
      } catch {
        show("failed", T.failed)
      }

      setTimeout(() => show("copy", T.copy), 1600)
    })

    pre.classList.add("has-copy")
    wrapper.appendChild(button)
  })
}

const CASK_URL = "https://raw.githubusercontent.com/polarity-dev/homebrew-tap/main/Casks/mestiere.rb"
const RELEASE_URL = "https://api.github.com/repos/polarity-dev/mestiere/releases/latest"
const CACHE_KEY = "mestiere-version-2"
const CACHE_MS = 10 * 60_000
const SEMVER = /^\d+\.\d+\.\d+$/

const semver = (version: string | undefined, source: string): string => {
  if (!version || !SEMVER.test(version)) {
    throw new Error(`${source}: no version`)
  }

  return version
}

const fromCask = async(signal: AbortSignal): Promise<string> => {
  const response = await fetch(CASK_URL, { signal, cache: "no-cache" })
  if (!response.ok) {
    throw new Error(`cask: ${response.status}`)
  }

  return semver((await response.text()).match(/^\s*version\s+"([^"]+)"/m)?.[1], "cask")
}

const fromApi = async(signal: AbortSignal): Promise<string> => {
  const response = await fetch(RELEASE_URL, { signal, headers: { Accept: "application/vnd.github+json" } })
  if (!response.ok) {
    throw new Error(`api: ${response.status}`)
  }

  const release = (await response.json()) as { tag_name?: string }
  return semver(release.tag_name?.replace(/^v/, ""), "api")
}

const latestVersion = async(): Promise<string> => {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), 6000)
  try {
    return await fromCask(controller.signal).catch(() => fromApi(controller.signal))
  } finally {
    clearTimeout(timeout)
  }
}

const readCache = (): string | null => {
  try {
    const cached = JSON.parse(sessionStorage.getItem(CACHE_KEY) ?? "null") as {
      version: string
      at: number
    } | null
    return cached && Date.now() - cached.at < CACHE_MS ? cached.version : null
  } catch {
    return null
  }
}

const writeCache = (version: string): void => {
  try {
    sessionStorage.setItem(CACHE_KEY, JSON.stringify({ version, at: Date.now() }))
  } catch {}
}

function initVersions(): void {
  const slots = document.querySelectorAll<HTMLElement>(".live-version[data-fallback]")
  if (!slots.length) {
    return
  }

  const show = (version: string, instant: boolean): void =>
    slots.forEach(slot => {
      slot.textContent = `${slot.dataset.prefix ?? ""}${version}`
      slot.classList.add("is-loaded")
      if (instant) {
        slot.classList.add("is-instant")
      }
    })
  const cached = readCache()
  if (cached) {
    show(cached, true)
    return
  }

  const fallback = slots[0]?.dataset.fallback ?? ""
  void latestVersion()
    .then(
      version => {
        writeCache(version)
        return version
      },
      () => fallback,
    )
    .then(version => show(version, false))
}

document.addEventListener("astro:page-load", () => {
  initCopyButtons()
  initVersions()
})
