---
title: Command reference
seoTitle: "mestiere command reference: every command and flag"
nav: Commands
description: "Every mestiere command with its arguments, flags and examples: init, registry, add, remove, sync, update, status, doctor, eject, adopt, prune and cache."
order: 6
---

Run `mestiere <command> --help` for the same information in your terminal. A bare `mestiere` in a terminal shows a banner with the main commands.

## Global flags

| Flag | Effect |
|---|---|
| `-C, --dir <dir>` | Run as if started in `<dir>` |
| `-q, --quiet` | Only print warnings and errors |
| `--no-input` | Never prompt: a missing argument fails with a hint instead. For agents and scripts. See [Agents](/docs/agents/#running-mestiere-from-an-agent). |
| `-v, --version` | Print the mestiere version |
| `-h, --help` | Help for any command |

`-g, --global` acts on the skills installed for every project, in your home directory. It works with `add`, `remove`, `sync`, `update`, `outdated`, `status`, `doctor`, `eject`, `adopt`, `prune` and `registry add`, `remove`, `list` and `show`. Other commands reject it with a message. See [Global skills](/docs/global-skills/).

## Aliases

| Alias | Command |
|---|---|
| `install` | `sync` |
| `rm`, `uninstall` | `remove` |
| `upgrade` | `update` |
| `ls`, `list` | `status` |

## Targets

Commands that take skills accept several forms:

| Form | Means |
|---|---|
| `frontend` | The `frontend` namespace |
| `frontend/forms` | The `forms` skill of `frontend` |
| `frontend-forms` | The same skill, by installed name |
| `frontend/frontend-forms` | The same skill, by folder name |
| `polarity:frontend` | `frontend` from the `polarity` registry |

Namespaces and skills may contain hyphens, so a name like `ai-tools-review` can mean more than one thing. When it does, mestiere stops and lists the possible readings: write `ns/skill` for a skill, or `ns/` for a namespace.

A mistyped command, flag, namespace, skill or installed name fails with a `did you mean …?` suggestion.

## Output and exit status

Commands exit non-zero on any `error:` line. `warning:` and `hint:` lines don't fail the command.

In a terminal, people get styled output: check marks, colors and a spinner while git fetches. Agents, scripts and pipes get plain text with the same content. `NO_COLOR` turns colors off. `status`, `registry show`, `outdated` and `doctor` print JSON with `--json`.

## Project setup

### mestiere init

Create `skills.yaml` and the `.gitignore` rule in the current project.

```sh
mestiere init --agents claude,kiro
```

| Flag | Effect |
|---|---|
| `--agents <list>` | Comma-separated agents: `claude`, `kiro`, `cursor`, `codex`, `gemini`, `copilot`, `opencode`. See [Agents](/docs/agents/#supported-agents). |
| `--link <mode>` | How agents get skills: `symlink` (default) or `copy` |

In a terminal without `--agents`, `init` runs a wizard:

1. It asks which agents the team uses, with the ones found in the project or your home folder preselected.
2. It asks for a registry git URL and checks git can reach it. Leave it empty to skip.
3. It opens a filterable skill picker, then asks you to confirm the install.
4. It prints the next steps.

## Registries

### mestiere registry add

Add a registry to the project.

```sh
mestiere registry add <name> <url> [--ref <branch|tag|commit>]
mestiere registry add polarity git@github.com:your-org/skills.git --ref main
```

`--ref` defaults to the registry's default branch.

`registry add` fetches the registry and detects its layout. A repository with no `registry.yaml` at the root and at least one `skills/<name>/SKILL.md` is a repository published for `npx skills`, and `skills.yaml` gets `layout: skills`. See [Skills from vercel/skills repositories](/docs/registries/#skills-from-vercelskills-repositories).

### mestiere registry remove

Uninstall the registry's skills and drop it from `skills.yaml`.

```sh
mestiere registry remove <name> [--force]
```

`--force` also deletes skills with local changes.

### mestiere registry list

List the project's registries.

### mestiere registry show

List a registry's namespaces with their description, each followed by its skills, indented and marked when installed.

```sh
mestiere registry show [name] [--json]
```

In a person's terminal each description is cut to 72 characters; agents and pipes get the whole text.

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

Without arguments, `add` opens a filterable picker that lists namespaces with their skills nested under them. Without prompts, a missing target is an error. A skill the registry doesn't offer is an error that says so, and suggests a skill with the same installed name when one exists.

`add` warns right after installing a skill that also exists in your home folder under the same name, when an agent would load that copy instead. See [Skills installed in both places](/docs/agents/#skills-installed-in-both-places).

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

```sh
mestiere outdated [--json]
```

## Inspecting

### mestiere status

Show installed skills, their links, local skills (including the ones that aren't committed) and global skills from every folder an agent reads in your home directory. In a folder without `skills.yaml` it lists the skills already there.

```sh
mestiere status [--json]
```

A skill that is both in the project and in your home folder is listed under "also installed globally", with the copy each agent loads. With `-g`, each global skill mestiere doesn't manage is shown as a copy, a link with its target, or a broken link.

### mestiere doctor

Check the project without changing anything: lock consistency, local changes, links, `.gitignore`, registry access and skills shadowed by a global copy.

```sh
mestiere doctor [--strict] [--offline] [--json]
```

| Flag | Effect |
|---|---|
| `--strict` | Also fail on local skills that aren't committed |
| `--offline` | Skip network checks |
| `--json` | Print the findings as JSON; the exit status still fails on problems |

With `-g`, `doctor` also reports broken links, skills with several separate copies, skills mestiere doesn't manage, and for each agent installed on the machine how many global skills it loads and how many characters their names and descriptions take. See [Cleaning up global skills](/docs/global-skills/#cleaning-up-global-skills).

Common errors and their fixes:

| Message | Fix |
|---|---|
| `has local changes` | Keep them with `mestiere eject <name>`, or discard them with `mestiere sync --force` |
| `install and lock disagree` | Run `mestiere sync` and commit `skills.yaml` |
| `already exists and isn't a published version` | A local skill has the same name: rename it, or overwrite it with `--force` |
| `comes from both <a> and <b>` | Add the skill to `exclude` in one of the registries |
| `isn't reachable` | A git access problem: check your credentials for that URL |
| `is committed but ignored` | `git rm -r --cached <path>`, or `mestiere track <name>` to keep committing it |
| `<agent> loads ~/… instead of this project's <name>` | A warning: a global copy hides the project's version from that agent. Remove it with `mestiere remove -g <name>`, or `mestiere prune -g <name>` if mestiere didn't install it, or rename one. It never fails, even with `--strict`. |
| `has N separate copies` (with `-g`) | `mestiere adopt -g <name>` if a registry publishes it, otherwise keep one copy |

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

Keep an installed skill as your own. The files stay; mestiere stops managing them and excludes the skill from future installs of its namespace. Registries with `layout: skills` have no namespaces, so nothing is written to `exclude`.

```sh
mestiere eject <skill>... [--track]
```

`--track` also commits the ejected skills. It doesn't apply with `-g`.

### mestiere adopt

Manage a skill you installed another way, such as a copy made by hand or with another tool.

```sh
mestiere adopt [-g] <skill>...
mestiere adopt -g shonin
```

`adopt` looks for a configured registry that publishes a skill with the same name and content as the folder on disk, then installs it from there. The folder becomes the managed copy, and links replace other copies. If no registry publishes it, the skill stays yours.

### mestiere prune

Move global skills mestiere doesn't manage to the trash.

```sh
mestiere prune -g [skill]...
mestiere prune -g ponytail-audit
```

`prune` moves every copy and link of the named skills to a dated folder under `~/.local/share/mestiere/trash` (`$XDG_DATA_HOME/mestiere/trash` when set). Nothing is deleted: move a folder back to restore it. It refuses skills mestiere manages; remove those with `mestiere remove -g`. Without names, people get a picker; agents and scripts must name the skills.

## Global skills

With `-g`, these commands act on the skills for every project on your machine instead of the current repository: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject`, `adopt`, `prune` and `registry add`, `remove`, `list` and `show`.

```sh
mestiere add -g mestiere
mestiere registry add -g polarity git@github.com:your-org/skills.git
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

Completions are dynamic and offline. They offer the skills of the cached registries for `add`, installed skills for `remove`, `update`, `eject`, `track` and `untrack`, unmanaged skills for `adopt` and `prune`, registry names, and the values of `--agents` and `--link`. Completion never touches the network: run `mestiere outdated` or `registry show` to refresh the cache.

Setup instructions: [Install](/install/#shell-completions).
