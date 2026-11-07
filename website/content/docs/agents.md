---
title: Agents
seoTitle: Agent Skills for Claude Code, Kiro, Codex and Cursor
nav: Agents
description: How mestiere installs Agent Skills for Claude Code, Kiro, Codex, Cursor and other agents, how to use copies, and how to let your agent set it up.
order: 5
---

An agent, or harness, is a coding tool that reads skills. mestiere stores every installed skill once in `.agents/skills/`, the shared folder that Codex, Gemini CLI, Cursor, GitHub Copilot and other agents read directly. Agents with their own folder, like Claude Code and Kiro, get links to it.

## Supported agents

| Agent | ID | Skills folder |
|---|---|---|
| All (canonical store) | `agents` (always on) | `.agents/skills/` |
| Claude Code | `claude` | `.claude/skills/` |
| Kiro | `kiro` | `.kiro/skills/` |

Codex, Gemini CLI, Cursor, GitHub Copilot and any other harness that reads `.agents/skills/` see installed skills without any extra setting, so they need no entry in `agents`.

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

## Teach your agent to use mestiere

The mestiere repository is also a registry. It ships two skills that teach coding agents to run mestiere correctly, including setting it up in a new repository:

| Skill | For |
|---|---|
| `mestiere/project` | Repositories that install skills: adding, updating, syncing, ejecting, committing, fixing `doctor` errors |
| `mestiere/registry` | Registry repositories: scaffolding, writing `SKILL.md`, sharing files through `_shared/`, fixing lint errors, CI |

They're installed globally, for every project on your machine. The Homebrew install already does it; otherwise run once:

```sh
mestiere add -g mestiere
```

Teammates whose agent lacks them run the same command. Then ask your agent, from any repository:

```text
Set up mestiere for this repo with our registry git@github.com:polarity-dev/skills.git
```

The agent runs `mestiere init`, shows you what the registry offers, installs the skills that fit the codebase and commits `skills.yaml`. Later prompts work the same way:

- "Are any of our skills outdated? Update them and summarize what changed."
- "I want to tweak frontend-forms for this repo, keep it as our own and commit it."
- "Why does mestiere doctor fail?"

## Reading these docs from an agent

Every page of this site has a Markdown version at the same URL plus `index.md`, for example [/docs/commands/index.md](/docs/commands/index.md). [llms.txt](/llms.txt) lists them all, and [llms-full.txt](/llms-full.txt) holds the whole documentation in one file.
