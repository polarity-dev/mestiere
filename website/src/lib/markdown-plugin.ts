import { satteriHeadingIdsPlugin } from "@astrojs/markdown-satteri"
import { defineHastPlugin, type HastPluginEntry } from "satteri"
import { liveVersion } from "./live-version.ts"
import { DEFAULT_LANG, isLang, localizeHref, type Lang } from "./site.ts"

const VERSION_TOKEN = "{{version}}"

function langOf(fileURL: URL | undefined): Lang {
  const folder = fileURL?.pathname.match(/\/src\/content\/([^/]+)\//)?.[1] ?? ""
  return isLang(folder) ? folder : DEFAULT_LANG
}

const contentPlugin: HastPluginEntry = ({ fileURL }) => {
  const lang = langOf(fileURL)

  return defineHastPlugin({
    name: "mestiere-content",
    element: [
      {
        filter: ["h2", "h3", "h4", "h5", "h6"],
        visit(node, ctx) {
          const id = node.properties["id"]
          if (typeof id !== "string") {
            return
          }

          ctx.prependChild(node, {
            type: "element",
            tagName: "a",
            properties: { className: ["anchor"], href: `#${id}`, ariaHidden: "true", tabIndex: -1 },
            children: [],
          })
        },
      },
      {
        filter: ["table"],
        visit(node, ctx) {
          ctx.wrapNode(node, {
            type: "element",
            tagName: "div",
            properties: { className: ["table-wrap"] },
            children: [],
          })
        },
      },
      {
        filter: ["a"],
        visit(node, ctx) {
          const href = node.properties["href"]
          if (typeof href === "string") {
            ctx.setProperty(node, "href", localizeHref(href, lang))
          }
        },
      },
    ],
    text(node, ctx) {
      if (!node.value.includes(VERSION_TOKEN)) {
        return
      }

      ctx.replaceNode(
        node,
        node.value
          .split(VERSION_TOKEN)
          .flatMap((value, i) => [
            ...(i > 0 ? [{ type: "raw" as const, value: liveVersion() }] : []),
            ...(value ? [{ type: "text" as const, value }] : []),
          ]),
      )
    },
  })
}

export const markdownOptions = {
  hastPlugins: [satteriHeadingIdsPlugin(), contentPlugin],
  features: { smartPunctuation: false, headingAttributes: true },
}
