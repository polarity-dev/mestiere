---
title: Agents
seoTitle: Agent Skills for Claude Code, Kiro, Codex and Cursor
nav: Agents
description: How mestiere installs Agent Skills for Claude Code, Kiro, Codex, Cursor and other agents, how agents run mestiere, and how to let your agent set it up.
order: 5
---

An agent, or harness, is a coding tool that reads skills. mestiere stores every installed skill once in `.agents/skills/`, the shared folder that Codex, Cursor, Gemini CLI, GitHub Copilot and other agents read directly. Agents with their own folder get links to it. `.agents/skills/` is the source of truth; every other folder only holds links.

## Supported agents

| Agent | ID | Links in a project | Links for global skills | Same name in project and `~` |
|---|---|---|---|---|
| All (store) | `agents` (always on) | `.agents/skills/` | `~/.agents/skills/` | |
| Claude Code | `claude` (`claude-code`) | `.claude/skills/` | `~/.claude/skills/` | global copy loads |
| Kiro | `kiro` (`kiro-cli`) | `.kiro/skills/` | `~/.kiro/skills/` | project copy loads |
| Cursor | `cursor` | none, reads `.agents/skills/` | none, reads `~/.agents/skills/` | not documented |
| Codex | `codex` | none | none | not documented |
| Gemini CLI | `gemini` (`gemini-cli`) | none | `~/.gemini/skills/` | not documented |
| GitHub Copilot | `copilot` (`github-copilot`) | none | `~/.copilot/skills/` | not documented |
| OpenCode | `opencode` | none | `~/.config/opencode/skills/` | not documented |

List in `agents` the ones that need links. Agents that read `.agents/skills/` see project skills with no setting. The IDs in parentheses are accepted too.

## Change the agents of a project

Edit `agents` in `skills.yaml`, then sync:

```yaml
agents:
  - claude
  - kiro
```

```sh
mestiere sync
```

`sync` adds links for new agents and removes them for agents you dropped. To choose at setup time, pass `mestiere init --agents claude,kiro`.

## Symlinks or copies

By default each agent folder gets a relative symlink to `.agents/skills/<name>`. If a tool doesn't follow symlinks, give agent folders real copies:

```yaml
link: copy
```

Or choose it at setup with `mestiere init --link copy`.

## Skills installed in both places

A skill can exist in the project and in your home folder under the same name, for example a team skill locked in `skills.yaml` and an older personal copy. Which one runs depends on the agent: Claude Code loads the personal copy, so the team's locked version never reaches it.

`status` lists these skills under "also installed globally", with the copy each agent loads. `doctor` warns when an agent loads the global copy, and `add` warns right after installing such a skill. Neither fails, even with `--strict`, because the global copy is on one machine only.

Fix it by removing the global copy, or by renaming one of them:

```sh
mestiere remove -g frontend-forms   # installed globally with mestiere
mestiere prune -g frontend-forms    # installed another way
```

## Running mestiere from an agent

mestiere prompts only when a person is at a terminal. Every prompt has a flag or argument that replaces it, so nothing requires a person:

```sh
mestiere init --no-input --agents claude,kiro
mestiere registry add --no-input polarity git@github.com:your-org/skills.git
mestiere add --no-input frontend backend/db-migrations
```

Prompts are off, and a missing argument fails with a hint that names what to pass, when any of these holds:

- the command has `--no-input`;
- `MESTIERE_NO_INPUT` or `CI` is set;
- stdin or stdout isn't a terminal;
- the shell belongs to a coding agent, recognized from `AI_AGENT`, `AGENT`, `CLAUDECODE`, `CODEX_SANDBOX`, `CURSOR_AGENT`, `GEMINI_CLI` or the variables Kiro's agent shell sets.

Without prompts, mestiere also sets `GIT_TERMINAL_PROMPT=0` and, unless you configured `core.sshCommand` or `GIT_SSH_COMMAND`, `ssh -o BatchMode=yes`, so git fails instead of waiting for a password.

Agent shells often have a terminal attached, so mestiere's own skills tell agents to pass `--no-input` on every command rather than rely on detection. Output for agents is plain text, with whole descriptions in `registry show`. `status`, `registry show`, `outdated` and `doctor` print JSON with `--json`. A mistyped name fails with a `did you mean …?` suggestion the agent can check before retrying.

## Teach your agent to use mestiere

The mestiere repository is also a registry. It ships two skills that teach coding agents to run mestiere correctly, including setting it up in a new repository:

| Skill | For |
|---|---|
| `mestiere-project` | Repositories that install skills: adding, updating, syncing, ejecting, committing, cleaning up global skills, fixing `doctor` errors |
| `mestiere-registry` | Registry repositories: scaffolding, writing `SKILL.md`, sharing files through `_shared/`, fixing lint errors, CI |

They're installed globally, for every project on your machine. The Homebrew install already does it; otherwise run once:

```sh
mestiere add -g mestiere
```

Teammates whose agent lacks them run the same command. Then ask your agent, from any repository:

```text
Set up mestiere for this repo with our registry git@github.com:your-org/skills.git
```

The agent runs `mestiere init`, shows you what the registry offers, installs the skills that fit the codebase and commits `skills.yaml`. Later prompts work the same way:

- "Are any of our skills outdated? Update them and summarize what changed."
- "I want to tweak frontend-forms for this repo, keep it as our own and commit it."
- "Why does mestiere doctor fail?"
- "Clean up my global skills."

## Reading these docs from an agent

Every page of this site has a Markdown version at the same URL plus `index.md`, for example [/docs/commands/index.md](/docs/commands/index.md). [llms.txt](/llms.txt) lists them all, and [llms-full.txt](/llms-full.txt) holds the whole documentation in one file.
