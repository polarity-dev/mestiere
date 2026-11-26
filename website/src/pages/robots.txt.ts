import type { APIRoute } from "astro"
import { robotsTxt } from "../lib/agent-files.ts"

export const GET: APIRoute = () => new Response(robotsTxt())
