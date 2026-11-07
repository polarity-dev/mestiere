---
title: Command reference
seoTitle: "mestiere command reference: every command and flag"
nav: Commands
description: Every mestiere command with its arguments, flags and examples: init, registry, add, remove, sync, update, outdated, status, track, eject, doctor and cache.
order: 6
---

Run `mestiere <command> --help` for the same information in your terminal.

## Global flags

| Flag | Effect |
|---|---|
| `-C, --dir <dir>` | Run as if started in `<dir>` |
| `-q, --quiet` | Only print warnings and errors |
| `-g, --global` | Act on the skills installed for every project, in your home directory. See [Global skills](/docs/global-skills/). |
| `--version` | Print the mestiere version |
| `-h, --help` | Help for any command |

## Targets

Commands that take skills accept several forms:

| Form | Means |
|---|---|
| `frontend` | The `frontend` namespace |
| `frontend/forms` | The `forms` skill of `frontend` |
| `frontend-forms` | The same skill, by installed name |
| `polarity:frontend` | `frontend` from the `polarity` registry |

## Exit status

Commands exit non-zero on any `error:` line. `warning:` and `hint:` lines don't fail the command.

## Project setup

### mestiere init

Create `skills.yaml` and the `.gitignore` rule in the current project.

```sh
mestiere init --agents claude,kiro
```

| Flag | Effect |
|---|---|
| `--agents <list>` | Comma-separated agents: `claude`, `kiro`. Asks when omitted in a terminal. |
| `--link <mode>` | How agents get skills: `symlink` (default) or `copy` |

## Registries

### mestiere registry add

Add a registry to the project.

```sh
mestiere registry add <name> <url> [--ref <branch|tag|commit>]
mestiere registry add polarity git@github.com:polarity-dev/skills.git --ref main
```

`--ref` defaults to the registry's default branch.

### mestiere registry remove

Uninstall the registry's skills and drop it from `skills.yaml`.

```sh
mestiere registry remove <name> [--force]
```

`--force` also deletes skills with local changes.

### mestiere registry list

List the project's registries.

### mestiere registry show

List the namespaces and skills a registry offers, with descriptions and install state.

```sh
mestiere registry show [name]
```

## Installing and removing

### mestiere add

Install namespaces or skills.

```sh
mestiere add <namespace | namespace/skill>... [-r <registry>] [--force]
mestiere add frontend
mestiere add backend/db-migrations -r polarity
```

| Flag | Effect |
|---|---|
| `-r, --registry <name>` | Registry to install from; required when several are configured |
| `--force` | Overwrite existing folders with the same name |

Without arguments, `add` opens an interactive picker.

### mestiere remove

Uninstall namespaces or skills.

```sh
mestiere remove <namespace | namespace/skill>... [--force]
```

Removing one skill of an installed namespace adds it to `exclude` instead. Modified skills are kept and reported as local skills unless you pass `--force`.

## Syncing and updating

### mestiere sync

Make the skill folders match the lock in `skills.yaml`. Run it after a clone, a pull, a branch switch or a hand edit.

```sh
mestiere sync [--frozen] [--force] [--dry-run]
```

| Flag | Effect |
|---|---|
| `--frozen` | Fail if `install` and `lock` disagree instead of re-resolving. Use in CI. |
| `--force` | Overwrite local changes |
| `--dry-run` | Print what would change |

### mestiere update

Move locked skills to the latest version of their ref. Without targets, everything is updated. Updating a namespace also picks up skills added to it and drops skills deleted from it.

```sh
mestiere update [target]... [--force]
mestiere update frontend polarity:backend
```

### mestiere outdated

Show skills whose content changed in their registry. A commit that doesn't touch a skill isn't reported.

## Inspecting

### mestiere status

Show installed skills, their links, local skills (including the ones that aren't committed) and global skills from `~/.agents/skills`, `~/.claude/skills` and `~/.kiro/skills`. In a folder without `skills.yaml` it lists the skills already there.

### mestiere doctor

Check the project without changing anything: lock consistency, local changes, links, `.gitignore` and registry access.

```sh
mestiere doctor [--strict] [--offline]
```

| Flag | Effect |
|---|---|
| `--strict` | Also fail on local skills that aren't committed |
| `--offline` | Skip network checks |

Common errors and their fixes:

| Message | Fix |
|---|---|
| `has local changes` | Keep them with `mestiere eject <name>`, or discard them with `mestiere sync --force` |
| `install and lock disagree` | Run `mestiere sync` and commit `skills.yaml` |
| `already exists and isn't a published version` | A local skill has the same name: rename it, or overwrite it with `--force` |
| `comes from both <a> and <b>` | Add the skill to `exclude` in one of the registries |
| `isn't reachable` | A git access problem: check your credentials for that URL |
| `is committed but ignored` | `git rm -r --cached <path>`, or `mestiere track <name>` to keep committing it |

## Your own skills

### mestiere track

Commit a skill by adding `.gitignore` exceptions for every folder it appears in.

```sh
mestiere track <skill>...
```

### mestiere untrack

Remove a skill's `.gitignore` exceptions.

```sh
mestiere untrack <skill>...
```

### mestiere eject

Keep an installed skill as your own. The files stay; mestiere stops managing them and excludes the skill from future installs of its namespace.

```sh
mestiere eject <skill>... [--track]
```

`--track` also commits the ejected skills. It doesn't apply with `-g`.

## Global skills

With `-g`, these commands act on the skills for every project on your machine instead of the current repository: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject` and the `registry` subcommands.

```sh
mestiere add -g mestiere
mestiere registry add -g polarity git@github.com:polarity-dev/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

`init`, `track` and `untrack` don't apply: nothing global is in git. Where files and the manifest go: [Global skills](/docs/global-skills/).

## Cache

### mestiere cache

Inspect or clear the registry cache: one bare git mirror per registry.

```sh
mestiere cache dir      # print the cache directory
mestiere cache info     # show the directory and its size
mestiere cache clean    # delete all mirrors; they are re-fetched when needed
```

The location is configurable: see [Configuration](/docs/ci/#configuration).

## Registry authoring

### mestiere registry init

Scaffold a new registry repository.

```sh
mestiere registry init [dir] [--name <name>] [--description <text>]
```

`--name` defaults to the directory name.

### mestiere registry new-namespace

```sh
mestiere registry new-namespace <namespace> [--description <text>] [--owners <list>]
```

### mestiere registry new-skill

Create a skill with a `SKILL.md` template.

```sh
mestiere registry new-skill <namespace>/<skill> [--description <text>]
```

### mestiere registry lint

Check a registry against the naming and `SKILL.md` rules. Run it in the registry's CI.

```sh
mestiere registry lint [dir]
```

See [what lint checks](/docs/registries/#what-lint-checks).

## Shell completions

```sh
mestiere completion bash|zsh|fish
```

Setup instructions: [Install](/install/#shell-completions).
