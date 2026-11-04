# Using mestiere in a project

This guide covers everything a project that installs skills needs: the files mestiere writes, the commands, how updates work, and how to keep your own skills next to installed ones. To publish skills, see [Authoring a registry](registry.md).

## Concepts

| Term | Meaning |
|---|---|
| Registry | A git repository of skills. Each project lists the registries it uses. |
| Namespace | A folder of related skills in a registry, such as `frontend` or `backend`. |
| Skill | A folder with a `SKILL.md`, plus any scripts, references and assets it needs. |
| Installed name | `<namespace>-<skill>`, e.g. `frontend-forms`. Commands also accept it. When a hyphenated name could mean more than one namespace or skill, write `ns/skill`, or `ns/` for a namespace. |
| Agent | A harness that reads skills. Codex, Gemini CLI, Cursor and GitHub Copilot read `.agents/skills/` directly; Claude Code and Kiro read their own folder (`.claude/skills/`, `.kiro/skills/`), which mestiere links. |
| Lock | The exact commit and content hash of each installed skill, stored in `skills.yaml`. |

## `skills.yaml`

One file holds both what you asked for and what got resolved:

```yaml
version: 1
agents:
  - claude
  - kiro
registries:
  polarity:
    url: git@github.com:polarity-dev/skills.git
    ref: main                     # branch, tag or commit; defaults to the registry's default branch
    install:
      - frontend                  # whole namespace
      - backend/db-migrations     # single skill
    exclude:
      - frontend/legacy-forms     # skip one skill of an installed namespace
    lock: # written by mestiere
      frontend-forms:
        path: skills/frontend/frontend-forms
        commit: 3f2a9c1e...
        tree: 9be1d07a...
```

- **`install`, `exclude` and `agents`** are yours to edit by hand. mestiere preserves your comments and key order when it rewrites the file.
- **`lock`** is written by mestiere. `tree` is the git tree ID of the installed folder, so it changes only when that skill's content changes.
- **Optional settings:**
  - `link: copy` gives agent folders copies instead of symlinks.
  - `strict: true` makes `doctor` fail on local skills that aren't committed.

## Installed layout

```
.agents/skills/frontend-forms/                  # real files
.claude/skills/frontend-forms -> ../../.agents/skills/frontend-forms
.kiro/skills/frontend-forms   -> ../../.agents/skills/frontend-forms
```

Links are relative, so they keep working when the repository moves.

| Agent | ID | Skills folder |
|---|---|---|
| All (canonical store) | `agents` (always on) | `.agents/skills/` |
| Claude Code | `claude` | `.claude/skills/` |
| Kiro | `kiro` | `.kiro/skills/` |

Harnesses that read `.agents/skills/` directly see installed skills without any extra setting.

## `.gitignore`

`mestiere init` adds one rule, and only if your existing rules don't already cover skill folders:

```gitignore
**/.*/skills/*
```

This ignores every skill folder inside any hidden directory, nested packages included, but leaves files like `.claude/settings.json` alone. A single exception line commits a whole skill:

```gitignore
!.claude/skills/my-skill
```

`mestiere track <skill>` and `mestiere untrack <skill>` add and remove these lines for every folder the skill appears in. Writing them by hand works just as well.

## Commands

| Command | What it does |
|---|---|
| `init [--agents claude,kiro] [--link copy]` | Create `skills.yaml` and the `.gitignore` rule |
| `registry add <name> <url> [--ref]` | Add a registry to the project |
| `registry remove <name>` | Uninstall the registry's skills and drop it |
| `registry list` | List the project's registries |
| `registry show [name]` | List a registry's namespaces with their `namespace.yaml` description, each followed by its skills, indented; a terminal cuts each description to 72 characters, a pipe shows the whole text |
| `add <ns \| ns/skill>... [-r registry]` | Install namespaces or skills; without arguments, a picker lists namespaces with their skills nested under them. `ns/ns-skill`, the folder name, also works |
| `remove <ns \| ns/skill>...` | Uninstall them (one skill of an installed namespace is excluded instead) |
| `sync [--frozen] [--force] [--dry-run]` | Make the skill folders match the lock |
| `update [targets...]` | Move skills to the latest version of their ref |
| `outdated` | Show skills whose content changed in the registry |
| `status` | Show installed skills, their links, local skills and global skills |
| `track` / `untrack <skill>...` | Commit a skill, or stop committing it |
| `eject <skill>... [--track]` | Keep an installed skill as your own; mestiere stops managing it |
| `doctor [--strict] [--offline]` | Check everything without changing anything |
| `cache dir \| info \| clean` | Inspect or clear the registry cache |
| `-g`, `--global` | Act on [global skills](#global-skills) instead of the project |
| `registry init \| new-namespace \| new-skill \| lint` | Author a registry |

`update` and `remove` targets accept `frontend`, `frontend/forms`, `frontend-forms` or a registry-qualified form like `polarity:frontend`. Every command accepts `-C <dir>` and `--quiet`. Shell completions come from `mestiere completion bash|zsh|fish`.

## How updates work

mestiere locks each skill to a registry commit, the same way package lockfiles do. Registry authors don't need to tag versions.

- `add` resolves the registry's `ref` and locks the new skills at that commit.
- `outdated` fetches the registry and lists skills whose content changed. A new commit that doesn't touch a skill isn't reported.
- `update` moves the lock forward, either for everything or only for the targets you name. Updating a namespace also picks up skills added to it and drops skills deleted from it.
- `sync` installs exactly what the lock says. `sync --frozen` fails if `install` and `lock` disagree, which is what CI should run.

Pin a whole project with `ref: v2026.10` (a tag) or a commit ID.

## Your own skills

mestiere manages only the skills listed in the lock. Anything else in a skills folder is yours: it's never modified, moved or deleted. See [how mestiere decides](architecture.md#safety-model).

- **Write a skill** directly in `.agents/skills/`, `.claude/skills/` or `.kiro/skills/`, then `mestiere track <name>` to commit it.
- **Customize a registry skill** with `mestiere eject <name>`. The files stay where they are and mestiere stops updating them. The skill is excluded from future installs of its namespace.
- **See uncommitted local skills** with `mestiere status`. They're ignored by default, so this catches a skill that teammates would otherwise never get. `status` also lists global skills from `~/.agents/skills`, `~/.claude/skills` and `~/.kiro/skills`, and works in folders without a `skills.yaml`.

## Global skills

Add `-g` to install skills for every project on your machine instead of one repository:

```sh
mestiere add -g mestiere                  # mestiere's own skills; the Homebrew install runs this
mestiere registry add -g polarity git@github.com:polarity-dev/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

Global skills use the same commands and safety rules as a project. The manifest is `~/.config/mestiere/skills.yaml` (or `$XDG_CONFIG_HOME/mestiere/skills.yaml`), files go to `~/.agents/skills/`, and the agents that have a folder in your home directory get links. A new global manifest comes with the `mestiere` registry configured. `track`, `untrack` and `init` don't apply, because nothing global is in git.

Prefer project installs for skills a team relies on: they're locked in a committed `skills.yaml`, so teammates and CI get the same versions. Global skills are per machine.

## CI

```yaml
- run: mestiere sync --frozen      # fail if skills.yaml is inconsistent
- run: mestiere doctor --strict    # fail on drift, broken links or uncommitted local skills
```

For private registries, give the CI job git credentials that can read the registry, such as a deploy key or a token. mestiere uses whatever git is configured with.

## Configuration

| Variable | Effect |
|---|---|
| `MESTIERE_CACHE_DIR` | Cache location (highest precedence) |
| `XDG_CACHE_HOME` | Cache goes to `$XDG_CACHE_HOME/mestiere`, honored on macOS too |
| `MESTIERE_NO_UPDATE_CHECK`, `NO_UPDATE_NOTIFIER` | Turn off the new-version notice |

By default the cache is `~/Library/Caches/mestiere` on macOS and `~/.cache/mestiere` on Linux. It holds one bare mirror per registry and is safe to delete: `mestiere cache clean` removes it, and the next command re-fetches what it needs.

Once a day mestiere asks GitHub for the latest release, in the background, and prints a notice on stderr when a newer version exists. The request sends no project data. It is skipped for development builds, when stderr isn't a terminal and when `CI` is set.
