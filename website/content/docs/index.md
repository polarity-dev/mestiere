---
title: Getting started
seoTitle: "Getting started with mestiere: install Agent Skills"
nav: Getting started
description: Set up mestiere in a repository, add a skills registry, install namespaces or single skills, and share them with teammates and CI.
order: 2
---

This guide installs skills from a registry into a project and shares them with your team. It assumes mestiere is [installed](/install/) and you can `git clone` the registry.

> Prefer to let your agent do it? With mestiere's skills installed (the Homebrew install adds them), ask "Set up mestiere for this repo with our registry <git-url>". The agent runs the same steps below and asks you which skills to install. See [Agents](/docs/agents/#teach-your-agent-to-use-mestiere).

## 1. Initialize the project

In the root of your repository:

```sh
mestiere init --agents claude,kiro
```

This creates `skills.yaml` and adds one rule to `.gitignore` so installed skills stay out of git. `--agents` picks the harnesses that receive the skills: `claude`, `kiro`, or both. Without the flag, `init` asks.

## 2. Add a registry

A registry is a git repository of skills. Give it a short name and its git URL:

```sh
mestiere registry add polarity git@github.com:polarity-dev/skills.git
```

mestiere follows the registry's default branch. Use `--ref` for a branch, tag or commit, e.g. `--ref v2026.10`. Check access first with `git ls-remote <url>` if you're unsure.

## 3. See what it offers

```sh
mestiere registry show polarity
```

This lists every namespace and skill with its description, and marks what's already installed.

## 4. Install skills

```sh
mestiere add frontend                 # a whole namespace
mestiere add backend/db-migrations    # a single skill
```

- A **namespace** install follows the namespace: skills added to it later arrive with the next `mestiere update`.
- A **skill** install brings just that skill.
- With several registries configured, pick one with `-r <registry>`.
- Without arguments, `add` opens an interactive picker.

Each skill lands in `.agents/skills/<namespace>-<skill>/`, for example `.agents/skills/frontend-forms/`, and each agent gets a relative link to it:

```
.agents/skills/frontend-forms/                  # real files
.claude/skills/frontend-forms -> ../../.agents/skills/frontend-forms
.kiro/skills/frontend-forms   -> ../../.agents/skills/frontend-forms
```

## 5. Commit and share

```sh
git add skills.yaml .gitignore
git commit -m "Add team skills"
```

`skills.yaml` now holds both what you asked for and the exact registry commit of every skill. A teammate who clones the repository runs:

```sh
mestiere sync
```

Run `mestiere sync` again after a pull or a branch switch. In CI, use `mestiere sync --frozen`: see [CI](/docs/ci/).

## Keep skills up to date

```sh
mestiere outdated              # skills whose content changed in the registry
mestiere update                # move everything to the latest version
mestiere update frontend       # or only some targets
```

Commit `skills.yaml` afterwards. A new registry commit that doesn't touch a skill isn't reported. Details: [How updates work](/docs/how-it-works/#how-updates-work).

## Remove skills

```sh
mestiere remove backend/db-migrations
```

Removing one skill of an installed namespace adds it to `exclude` instead, so the rest of the namespace stays. Skills with local changes are kept and reported.

## Your own skills

mestiere manages only the skills listed in the lock. Anything else in a skills folder is yours: it's never modified, moved or deleted.

- **Write a skill** directly in `.agents/skills/`, `.claude/skills/` or `.kiro/skills/`, then run `mestiere track <name>` to commit it.
- **Customize a registry skill** with `mestiere eject <name>`. The files stay where they are and mestiere stops updating them. Add `--track` to commit it.
- **Find uncommitted local skills** with `mestiere status`. They're ignored by default, so this catches a skill your teammates would never get.

## Skills for every project

Add `-g` to install skills for your whole machine instead of one repository, for example `mestiere add -g -r polarity frontend`. Prefer project installs for skills a team relies on. See [Global skills](/docs/global-skills/).

## Check the project

```sh
mestiere status     # installed skills, their links, local and global skills
mestiere doctor     # everything, without changing anything
```

`status` also works in a folder without `skills.yaml`: it lists the skills already there and the global ones.

`doctor` exits with an error on drift, broken links or inconsistent files and says how to fix each problem.

## Next steps

- [skills.yaml reference](/docs/skills-yaml/): every setting, the lock, and `.gitignore`.
- [Command reference](/docs/commands/): every command and flag.
- [Registries](/docs/registries/): publish your own skills.
- [Global skills](/docs/global-skills/): skills for every project on your machine.
