---
title: Global skills
seoTitle: Install Agent Skills for every project with mestiere -g
nav: Global skills
description: "Install Agent Skills for every project on your machine with mestiere -g: where files go, which commands apply, how to clean up, and when to prefer a project."
order: 5.5
---

Global skills are installed once for your whole machine instead of one repository. Agents load them in every project. Add `-g` to a command to act on them:

```sh
mestiere add -g mestiere                  # mestiere's own skills; the Homebrew install runs this
mestiere registry add -g polarity git@github.com:your-org/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

## Where things go

| What | Where |
|---|---|
| Manifest | `~/.config/mestiere/skills.yaml`, or `$XDG_CONFIG_HOME/mestiere/skills.yaml` |
| Skill files | `~/.agents/skills/<namespace>-<skill>/` |
| Links | `~/.claude/skills/`, `~/.kiro/skills/`, `~/.gemini/skills/`, `~/.copilot/skills/`, `~/.config/opencode/skills/`, for the agents installed on your machine |
| Trash | `~/.local/share/mestiere/trash/`, or `$XDG_DATA_HOME/mestiere/trash/`, for skills moved by `prune` |

The manifest has the same format as a project's [skills.yaml](/docs/skills-yaml/), with its `install`, `exclude` and lock. A new global manifest comes with the `mestiere` registry already configured, so `mestiere add -g mestiere` works without `registry add`.

A new global manifest links the agents that have a configuration folder in your home directory, such as `~/.claude`, and Claude Code and Kiro when none is found. It creates no folders for agents you don't use. Cursor and Codex read `~/.agents/skills/` and need no links. The full table is in [Agents](/docs/agents/#supported-agents).

## Commands

These commands accept `-g`: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject`, `adopt`, `prune` and `registry add`, `remove`, `list` and `show`. They work exactly as in a project, with the same [safety model](/docs/how-it-works/#safety-model): a global skill you edited is never overwritten.

`init`, `track`, `untrack` and `eject --track` don't apply, because nothing global is in git. Other commands reject `-g` with a message.

## mestiere's own skills

`mestiere add -g mestiere` installs `mestiere-project` and `mestiere-registry`, which teach your agent to set up and run mestiere in any repository. The Homebrew install does it for you and keeps them updated on `brew upgrade`. See [Agents](/docs/agents/#teach-your-agent-to-use-mestiere).

To remove them, for example before uninstalling:

```sh
mestiere remove -g mestiere
```

## Cleaning up global skills

Global skills load in every session of every project, so leftovers cost context everywhere. `mestiere doctor -g` reports:

- skills mestiere doesn't manage;
- broken links, and skills with several separate copies that can drift apart;
- for each agent installed on the machine, how many global skills it loads and how many characters their names and descriptions take.

`mestiere status -g` shows each unmanaged skill as a copy, a link with its target, or a broken link. Then act on each one:

```sh
mestiere adopt -g shonin          # manage it, if a configured registry publishes the same content
mestiere prune -g ponytail-audit  # move every copy and link of it to the trash
```

`adopt` works when a configured registry publishes a skill with the same name and content, such as one you copied by hand or installed with another tool. The folder becomes the managed copy, and links replace other copies.

`prune` moves skills to a dated folder under `~/.local/share/mestiere/trash` (`$XDG_DATA_HOME/mestiere/trash` when set). Nothing is deleted: move a folder back to restore it. It refuses skills mestiere manages; remove those with `mestiere remove -g`. Without names, people get a picker; agents and scripts must name the skills.

## Seeing global skills from a project

`mestiere status` in a project also lists the skills in every folder an agent reads in your home directory, global or not, because agents load them there too. It works in folders without a `skills.yaml` as well.

A skill that is both in the project and in your home folder is listed under "also installed globally", with the copy each agent loads: Claude Code loads the global one, Kiro the project one. `doctor` warns when an agent loads the global copy. See [Skills installed in both places](/docs/agents/#skills-installed-in-both-places).

## Global or project?

Prefer project installs for skills a team relies on. They're locked in a committed `skills.yaml`, so teammates and CI get the same versions. Global skills are per machine: use them for personal skills and for tools like mestiere's own skills.
