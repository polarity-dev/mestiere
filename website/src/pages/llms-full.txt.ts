import type { APIRoute } from "astro"
import { llmsFullTxt } from "../lib/agent-files.ts"
import { getPages } from "../lib/pages.ts"

export const GET: APIRoute = async() => new Response(llmsFullTxt((await getPages()).en))
