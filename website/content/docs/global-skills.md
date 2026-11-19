---
title: Global skills
seoTitle: Install Agent Skills for every project with mestiere -g
nav: Global skills
description: Install Agent Skills for every project on your machine with mestiere -g: where files and the manifest go, which commands apply, and when to prefer a project.
order: 5.5
---

Global skills are installed once for your whole machine instead of one repository. Agents load them in every project. Add `-g` to a command to act on them:

```sh
mestiere add -g mestiere                  # mestiere's own skills; the Homebrew install runs this
mestiere registry add -g polarity git@github.com:polarity-dev/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

## Where things go

| What | Where |
|---|---|
| Manifest | `~/.config/mestiere/skills.yaml`, or `$XDG_CONFIG_HOME/mestiere/skills.yaml` |
| Skill files | `~/.agents/skills/<namespace>-<skill>/` |
| Links | `~/.claude/skills/`, `~/.kiro/skills/`, for the agents that have a folder in your home directory |

The manifest has the same format as a project's [skills.yaml](/docs/skills-yaml/), with its `install`, `exclude` and lock. A new global manifest comes with the `mestiere` registry already configured, so `mestiere add -g mestiere` works without `registry add`.

If none of the agents has a folder in your home directory yet, mestiere links all of them. Otherwise it creates no folders for agents you don't use.

## Commands

These commands accept `-g`: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject` and the `registry` subcommands. They work exactly as in a project, with the same [safety model](/docs/how-it-works/#safety-model): a global skill you edited is never overwritten.

`init`, `track`, `untrack` and `eject --track` don't apply, because nothing global is in git.

## mestiere's own skills

`mestiere add -g mestiere` installs `mestiere-project` and `mestiere-registry`, which teach your agent to set up and run mestiere in any repository. The Homebrew install does it for you and keeps them updated on `brew upgrade`. See [Agents](/docs/agents/#teach-your-agent-to-use-mestiere).

To remove them, for example before uninstalling:

```sh
mestiere remove -g mestiere
```

## Seeing global skills from a project

`mestiere status` in a project also lists the skills in `~/.agents/skills`, `~/.claude/skills` and `~/.kiro/skills`, global or not, because agents load them there too. It works in folders without a `skills.yaml` as well.

## Global or project?

Prefer project installs for skills a team relies on. They're locked in a committed `skills.yaml`, so teammates and CI get the same versions. Global skills are per machine: use them for personal skills and for tools like mestiere's own skills.
