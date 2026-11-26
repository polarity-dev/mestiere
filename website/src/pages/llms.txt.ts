import type { APIRoute } from "astro"
import { llmsTxt } from "../lib/agent-files.ts"
import { getPages } from "../lib/pages.ts"

export const GET: APIRoute = async() => new Response(llmsTxt((await getPages()).en))
