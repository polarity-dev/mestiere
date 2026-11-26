import { markdownTwin } from "./markdown.ts"
import { docPages, type Page } from "./pages.ts"
import { SITE } from "./site.ts"

export function llmsTxt(pages: Page[]): string {
  const home = pages.find(p => p.key === "/")
  const docs = docPages(pages)
    .map(p => `- [${p.navTitle}](${p.url}index.md): ${p.description}`)
    .join("\n")
  return `# mestiere

> ${SITE.tagline} mestiere is a command-line tool (macOS, Linux) that installs Agent Skills (folders with a SKILL.md) from git repositories called registries into a project's .agents/skills/ folder, locks each skill to a registry commit and content hash in skills.yaml, and links the skills into .claude/skills/ and .kiro/skills/ for Claude Code and Kiro. Cursor, Codex, Gemini CLI, GitHub Copilot and OpenCode read .agents/skills/ directly. It also installs skills from repositories published for \`npx skills\`.

Key facts for agents:

- Install with \`brew install --cask polarity-dev/tap/mestiere\` (macOS, Linux; upgrade with \`brew upgrade --cask mestiere\`), the latest release archive \`${SITE.repo}/releases/latest/download/mestiere_<os>_<arch>.tar.gz\` (os darwin or linux, arch arm64 or amd64, plus \`checksums.txt\`), or \`go install github.com/polarity-dev/mestiere/cmd/mestiere@latest\`. After a release archive or go install, run \`mestiere add -g mestiere\` once. Requires \`git\` on the PATH. Windows is not supported yet.
- Pass \`--no-input\` on every command: your shell may look like a terminal, and a prompt would block you. With it, a missing argument fails at once with a hint. \`status\`, \`registry show\`, \`outdated\` and \`doctor\` accept \`--json\`. A \`did you mean …?\` error means a name was mistyped; check the suggestion matches the user's request before retrying.
- Check the CLI with \`mestiere --version\`. Start with \`mestiere status\`: it lists installed, local and global skills, and in a folder without skills.yaml it says so (then set the project up).
- mestiere's own skills (\`mestiere-project\`, \`mestiere-registry\`) are installed globally by Homebrew or \`mestiere add -g mestiere\`; they cover setting up a repository.
- \`-g\` acts on global skills (\`~/.agents/skills/\`, manifest \`~/.config/mestiere/skills.yaml\`) instead of the project; \`init\`, \`track\` and \`untrack\` don't apply. Prefer project installs for team skills.
- Supported agent IDs: \`claude\`, \`kiro\`, \`cursor\`, \`codex\`, \`gemini\`, \`copilot\`, \`opencode\`. A skill both in the project and in ~ is shadowed for some agents (Claude Code loads the global copy); \`status\` and \`doctor\` report it. Fix with \`mestiere remove -g <name>\` or, if unmanaged, \`mestiere prune -g <name>\`.
- Clean up global skills: \`mestiere doctor -g\` lists unmanaged skills and how much context each agent loads; \`mestiere adopt -g <name>\` manages one a configured registry publishes; \`mestiere prune -g <name>\` moves one to ~/.local/share/mestiere/trash (reversible). Only prune with the user's approval.
- Skills published for \`npx skills\` (e.g. Next.js): \`mestiere registry add next https://github.com/vercel/next.js\`, then \`mestiere add -r next <skill-name>\`. Names have no namespace. Don't run \`npx skills\` in a project managed by mestiere.
- Typical project setup: \`mestiere init --agents claude,kiro\` (only the agents the team uses), \`mestiere registry add <name> <git-url>\`, \`mestiere registry show <name>\` to see what it offers and propose skills to the user, \`mestiere add <namespace>\` or \`mestiere add <namespace>/<skill>\`, then commit \`skills.yaml\` and \`.gitignore\`. Teammates run \`mestiere sync\` after cloning or pulling.
- Updates: \`mestiere outdated\` lists skills whose content changed in the registry; \`mestiere update [targets]\` moves the lock forward; commit \`skills.yaml\` afterwards.
- CI: \`mestiere sync --frozen\` and \`mestiere doctor --strict\` fail on an inconsistent skills.yaml, drifted skills or uncommitted local skills. \`mestiere doctor\` explains every error and its fix.
- Own skills: write them in \`.agents/skills/\` (or an agent folder) and commit them with \`mestiere track <name>\`; installed skills are gitignored by default.
- Always pass targets to \`mestiere add\`: without arguments it opens an interactive picker.
- Never edit the \`lock\` section of skills.yaml or files under .agents/skills/ in place; use \`mestiere eject <name>\` to customize a skill. Do not pass \`--force\` without the user's approval: it discards local edits.
- Installed name of skill \`<namespace>/<skill>\` is \`<namespace>-<skill>\`, which is also its folder in the registry (\`skills/<namespace>/<namespace>-<skill>/\`). Namespaces and skills may contain hyphens: when a name is ambiguous, mestiere lists the readings; write \`ns/skill\` or \`ns/\`.
- A skill's top-level README.md is for people browsing the registry and is never installed.
- Every page below is plain Markdown. ${SITE.url}/llms-full.txt holds all of them in one file.

## Docs

- [Overview](${SITE.url}/index.md): ${home?.description ?? ""}
${docs}

## Optional

- [Full documentation in one file](${SITE.url}/llms-full.txt): Every page above, concatenated
- [Documentazione in italiano](${SITE.url}/it/index.md): The same pages in Italian, under ${SITE.url}/it/
- [Source code](${SITE.repo}): Go source, README and releases
- [Agent Skills specification](https://agentskills.io): The SKILL.md format mestiere installs
`
}

export function llmsFullTxt(pages: Page[]): string {
  const parts = pages.map(p => `<!-- ${p.url} -->\n\n${markdownTwin(p)}`)
  return `# mestiere documentation\n\n> ${SITE.tagline}\n\nSource: ${SITE.url}/llms.txt.\n\n${parts.join("\n---\n\n")}`
}

export const robotsTxt =
  (): string => `# mestiere.io: public documentation, open to search engines and AI crawlers.
User-agent: *
Allow: /
Disallow: /404.html

# AI crawlers and assistants, listed explicitly so the intent is unambiguous.
User-agent: GPTBot
User-agent: OAI-SearchBot
User-agent: ChatGPT-User
User-agent: ClaudeBot
User-agent: Claude-SearchBot
User-agent: Claude-User
User-agent: PerplexityBot
User-agent: Perplexity-User
User-agent: Google-Extended
User-agent: Applebot-Extended
User-agent: CCBot
User-agent: Amazonbot
User-agent: meta-externalagent
User-agent: MistralAI-User
Allow: /

Sitemap: ${SITE.url}/sitemap-index.xml

# LLM-friendly index of this site: ${SITE.url}/llms.txt
`
