---
title: mestiere
seoTitle: "mestiere: share Agent Skills across your team's repositories"
description: mestiere installs your team's Agent Skills from a git registry into every repository, pinned and verified, for Claude Code, Kiro and other coding agents.
order: 0
---

mestiere is a command-line tool for macOS and Linux. Your team keeps its [Agent Skills](https://agentskills.io) in one git repository, a *registry*, organized by namespace. Each project installs the skills it needs, pinned to an exact registry commit, and every agent harness in the repository sees them.

*Mestiere* is Italian for craft: the know-how a workshop hands from one person to the next.

## Install

```sh
brew install --cask polarity-dev/tap/mestiere
```

Homebrew works on macOS and Linux. Prebuilt archives (amd64, arm64) are on every [GitHub release](https://github.com/polarity-dev/mestiere/releases), and `go install` works too. Full instructions: [Install mestiere](/install/).

## Quick start with your agent

The Homebrew install also gives Claude Code and Kiro mestiere's own skills globally. If you installed another way, run `mestiere add -g mestiere` once. Then ask your agent, from any repository:

```text
Set up mestiere for this repo with our registry git@github.com:polarity-dev/skills.git
```

The agent runs `mestiere init`, shows you what the registry offers, installs the skills that fit the codebase and commits `skills.yaml`. Later prompts work the same way: "Are any of our skills outdated? Update them and summarize what changed", or "I want to tweak frontend-forms for this repo, keep it as our own and commit it".

## Quick start

```sh
mestiere init --agents claude,kiro
mestiere registry add polarity git@github.com:polarity-dev/skills.git
mestiere add frontend backend/db-migrations
git add skills.yaml .gitignore && git commit -m "Add team skills"
```

A teammate who clones the repository runs `mestiere sync`. Step by step: [Getting started](/docs/).

## How it works

1. A registry is a git repository of skills grouped in namespaces, such as `frontend` or `backend`.
2. `mestiere add` copies skills into `.agents/skills/<namespace>-<skill>/` and locks each one to a registry commit and content hash in `skills.yaml`.
3. Codex, Gemini CLI, Cursor and GitHub Copilot read `.agents/skills/` directly. Claude Code and Kiro get relative symlinks in `.claude/skills/` and `.kiro/skills/`.
4. `mestiere sync` makes any checkout match the lock; `mestiere update` moves skills to the latest registry version.

## Why mestiere

- **One source of truth.** Skills live in a git registry, public or private, reviewed like any other code.
- **Organized by namespace.** Install a whole namespace or a single skill. New skills in an installed namespace arrive on the next update.
- **Reproducible.** Every install is locked to a registry commit and a content hash. Teammates and CI get byte-identical skills.
- **Every agent at once.** Files live once in `.agents/skills/`, which Codex, Gemini CLI, Cursor and others read directly. Agents with their own folder, like Claude Code and Kiro, get symlinks to it.
- **Out of your way.** Installed skills are ignored by git by default. Your own skills sit next to them untouched, and committing any skill is one command.
- **Safe by construction.** mestiere replaces or deletes a folder only when it can prove the folder is an unmodified copy of a published skill. Edited skills are never overwritten silently.
- **No lock-in.** Registries are plain git repositories. Installed skills are plain folders that follow the Agent Skills spec.

## Frequently asked questions

### What is mestiere?

mestiere is a command-line tool that installs Agent Skills from git repositories into your projects. It works like a package manager for skills: you declare what you want in `skills.yaml`, mestiere resolves it to exact registry commits, and everyone who runs `mestiere sync` gets the same files.

### What are Agent Skills?

An Agent Skill is a folder with a `SKILL.md` file, plus any scripts, references and assets it needs. Coding agents such as Claude Code, Kiro, Codex and Cursor read the skill's description and load it when a task matches. The format is described at [agentskills.io](https://agentskills.io).

### Which coding agents does mestiere support?

Every agent that reads Agent Skills. Installed files live in `.agents/skills/`, the shared folder that Codex, Gemini CLI, Cursor, GitHub Copilot and others read directly. Claude Code (`.claude/skills/`) and Kiro (`.kiro/skills/`) get links to it. See [Agents](/docs/agents/).

### Can I use a private skills repository?

Yes. mestiere runs your system `git`, so a registry works wherever `git clone` works: SSH keys, credential helpers and `gh auth` included. In CI, give the job credentials that can read the registry.

### How do I share skills with my whole team?

Commit `skills.yaml` and `.gitignore`. Teammates and CI run `mestiere sync`, which installs exactly the locked versions. In CI, `mestiere sync --frozen` fails if the file is inconsistent. See [CI](/docs/ci/).

### Do I have to commit the installed skills?

No. Installed skills are ignored by git by default, because `skills.yaml` is enough to reproduce them. To commit a skill anyway, run `mestiere track <skill>`.

### Will mestiere overwrite a skill I edited?

No. Before replacing or deleting a folder, mestiere checks that its content matches a version published in the registry. Edited skills make `sync` stop; keep your changes with `mestiere eject <skill>`. See the [safety model](/docs/how-it-works/#safety-model).

### Can my agent set up mestiere for me?

Yes. mestiere ships its own skills, and the Homebrew install gives them to Claude Code and Kiro for every project. Ask "Set up mestiere for this repo with our registry <url>" and the agent runs `init`, proposes skills and commits `skills.yaml`. See [Agents](/docs/agents/#teach-your-agent-to-use-mestiere).

### Can I install skills for every project on my machine?

Yes, with `-g`: `mestiere add -g -r polarity frontend` installs into `~/.agents/skills/`, linked for your agents, with the manifest in `~/.config/mestiere/skills.yaml`. Prefer project installs for skills the team relies on, because only a committed `skills.yaml` gives teammates and CI the same versions. See [Global skills](/docs/global-skills/).

### How do I update skills?

`mestiere outdated` lists skills whose content changed in the registry. `mestiere update` moves all of them to the latest version, or only the targets you name. See [How updates work](/docs/how-it-works/#how-updates-work).

### Does mestiere work on Windows?

Not yet. macOS and Linux on amd64 and arm64 are supported; Windows support is on the roadmap.
