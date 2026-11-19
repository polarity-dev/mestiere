---
title: skills.yaml reference
seoTitle: skills.yaml reference for mestiere projects
nav: skills.yaml
description: Reference for skills.yaml, the mestiere project file that lists agents, registries, installed namespaces and skills, exclusions and the lock.
order: 3
---

`skills.yaml` sits at the root of the project. One file holds both what you asked for and what got resolved. [Global skills](/docs/global-skills/) use the same format in `~/.config/mestiere/skills.yaml`.

## Example

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
        path: skills/frontend/forms
        commit: 3f2a9c1e...
        tree: 9be1d07a...
```

## Keys

| Key | Edited by | Meaning |
|---|---|---|
| `version` | mestiere | File format version. Always `1`. |
| `agents` | you | Harnesses that get links: `claude`, `kiro`. `.agents/skills/` is always used. |
| `registries.<name>.url` | you | Git URL of the registry: SSH, HTTPS or a local path. |
| `registries.<name>.ref` | you | Branch, tag or commit to follow. Defaults to the registry's default branch. |
| `registries.<name>.install` | you | Namespaces (`frontend`) and skills (`backend/db-migrations`) to install. |
| `registries.<name>.exclude` | you | Skills of an installed namespace to skip. |
| `registries.<name>.lock` | mestiere | Resolved skills: registry path, commit and content hash. Never edit it. |
| `link` | you | Optional. `copy` gives agent folders copies instead of symlinks. |
| `strict` | you | Optional. `true` makes `doctor` fail on local skills that aren't committed. |

mestiere preserves your comments and key order when it rewrites the file. After editing `install`, `exclude` or `agents` by hand, run `mestiere sync`.

## The lock

Each entry under `lock` is keyed by the installed name, `<namespace>-<skill>`:

- `path`: where the skill lives in the registry.
- `commit`: the registry commit it was resolved at.
- `tree`: the git tree ID of the installed folder. It changes only when that skill's content changes, which is how `outdated` ignores commits that don't touch a skill.

`add` and `update` write the lock. `sync` installs exactly what it says.

## Pinning a project

Follow a tag or a commit instead of a branch:

```yaml
registries:
  polarity:
    url: git@github.com:polarity-dev/skills.git
    ref: v2026.10
```

Registry authors don't need to tag versions: every install is already locked to a commit.

## Installed layout

```
.agents/skills/frontend-forms/                  # real files
.claude/skills/frontend-forms -> ../../.agents/skills/frontend-forms
.kiro/skills/frontend-forms   -> ../../.agents/skills/frontend-forms
```

Links are relative, so they keep working when the repository moves. With `link: copy`, agent folders get copies instead.

## .gitignore

`mestiere init` adds one rule, and only if your existing rules don't already cover skill folders:

```gitignore
**/.*/skills/*
```

It ignores every skill folder inside any hidden directory, nested packages included, but leaves files like `.claude/settings.json` alone. A single exception line commits a whole skill:

```gitignore
!.claude/skills/my-skill
```

`mestiere track <skill>` and `mestiere untrack <skill>` add and remove these lines for every folder the skill appears in. Writing them by hand works just as well.
