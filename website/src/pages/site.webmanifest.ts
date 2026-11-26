import type { APIRoute } from "astro"
import { SITE } from "../lib/site.ts"

export const GET: APIRoute = () =>
  new Response(
    JSON.stringify(
      {
        name: SITE.name,
        short_name: SITE.name,
        description: SITE.tagline,
        start_url: "/",
        display: "browser",
        background_color: "#000000",
        theme_color: "#000000",
        icons: [
          { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
          { src: "/apple-touch-icon.png", sizes: "180x180", type: "image/png" },
          { src: "/favicon.svg", sizes: "any", type: "image/svg+xml" },
        ],
      },
      null,
      2,
    ),
  )
