import { defineCollection } from "astro:content"
import { glob } from "astro/loaders"
import { z } from "astro/zod"

const pages = defineCollection({
  loader: glob({
    pattern: "**/*.md",
    base: "./src/content",
    generateId: ({ entry }) => entry.replace(/\.md$/, ""),
  }),
  schema: z.object({
    title: z.string(),
    seoTitle: z.string().optional(),
    nav: z.string().optional(),
    description: z.string().max(160),
    order: z.number().default(999),
  }),
})

export const collections = { pages }
