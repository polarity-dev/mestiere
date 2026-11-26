import { execFileSync } from "node:child_process"
import { SITE } from "./site.ts"

function git(...args: string[]): string {
  try {
    return execFileSync("git", args, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim()
  } catch {
    return ""
  }
}

export const BUILD_DATE = new Date().toISOString()

const modified = new Map<string, string>()

export function lastModified(file: string): string {
  let date = modified.get(file)
  if (date === undefined) {
    date = git("log", "-1", "--format=%cI", "--", file) || BUILD_DATE
    modified.set(file, date)
  }

  return date
}

const newestTag = (tags: string[]): string | undefined =>
  tags
    .filter(tag => /^v\d+\.\d+\.\d+$/.test(tag))
    .sort((a, b) => a.localeCompare(b, "en", { numeric: true }))
    .at(-1)

const remoteTags = git("ls-remote", "--tags", "--refs", `${SITE.repo}.git`)
  .split("\n")
  .map(line => line.split("refs/tags/")[1] ?? "")
  .filter(Boolean)

export const VERSION = (
  newestTag(remoteTags) ||
  git("describe", "--tags", "--abbrev=0", "--match", "v[0-9]*") ||
  "v0.1.0"
).replace(/^v/, "")
