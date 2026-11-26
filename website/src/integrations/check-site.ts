import { existsSync, readdirSync, readFileSync } from "node:fs"
import { join, relative } from "node:path"
import { fileURLToPath } from "node:url"
import type { AstroIntegration } from "astro"
import { isPagePath } from "../lib/site.ts"

function checkSite(dist: string): string[] {
  const htmlFiles = readdirSync(dist, { recursive: true, encoding: "utf8" })
    .filter(f => f.endsWith(".html"))
    .map(f => join(dist, f))
  const ids = new Map<string, Set<string>>()
  const idsOf = (file: string): Set<string> => {
    let set = ids.get(file)
    if (!set) {
      set = new Set([...readFileSync(file, "utf8").matchAll(/\sid="([^"]+)"/g)].map(m => m[1] ?? ""))
      ids.set(file, set)
    }

    return set
  }

  const target = (path: string): string => {
    const file = join(dist, decodeURIComponent(path))
    return path.endsWith("/") ? join(file, "index.html") : file
  }

  const errors: string[] = []
  for (const file of htmlFiles) {
    const html = readFileSync(file, "utf8")
    const name = relative(dist, file)
    const pagePath = `/${name.replace(/index\.html$/, "")}`
    const italian = name.startsWith("it/")

    for (const [, attr, value = ""] of html.matchAll(/\s(href|src|srcset)="([^"]+)"/g)) {
      const refs = attr === "srcset" ? value.split(",").map(s => s.trim().split(/\s+/)[0] ?? "") : [value]
      for (const ref of refs) {
        if (/^(https?:|mailto:|data:)/.test(ref)) {
          continue
        }

        const [beforeHash = "", hash] = ref.split("#")
        const path = beforeHash.split("?")[0] ?? ""
        const resolved = path === "" ? file : target(path.startsWith("/") ? path : pagePath + path)
        if (!existsSync(resolved)) {
          errors.push(`${name}: broken link ${ref}`)
        } else if (hash && resolved.endsWith(".html") && !idsOf(resolved).has(hash)) {
          errors.push(`${name}: missing anchor ${ref}`)
        }
      }
    }

    if (italian) {
      for (const [tag, href = ""] of html.matchAll(/<a\b[^>]*\shref="(\/[^"]*)"[^>]*>/g)) {
        const path = href.split(/[#?]/)[0] ?? ""
        if (!path.startsWith("/it/") && isPagePath(path) && !/\shreflang="/.test(tag)) {
          errors.push(`${name}: links to the English page ${href}`)
        }
      }
    }

    for (const [tag] of html.matchAll(/<script\b[^>]*>/g)) {
      if (!/\ssrc="/.test(tag) && !/type="application\/ld\+json"/.test(tag)) {
        errors.push(`${name}: inline script ${tag}`)
      }
    }
  }

  return errors
}

export const checkSiteIntegration = (): AstroIntegration => ({
  name: "mestiere:check-site",
  hooks: {
    "astro:build:done": ({ dir, logger }) => {
      const errors = checkSite(fileURLToPath(dir))
      if (errors.length) {
        throw new Error(`the site breaks its rules:\n  ${errors.join("\n  ")}`)
      }

      logger.info("links, anchors, languages and inline scripts checked")
    },
  },
})
