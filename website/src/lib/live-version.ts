import { VERSION } from "./git.ts"

export const liveVersion = (prefix = ""): string =>
  `<span class="live-version" data-prefix="${prefix}" data-fallback="${VERSION}">${prefix}${VERSION}</span>`
