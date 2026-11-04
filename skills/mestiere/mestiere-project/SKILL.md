---
name: mestiere-project
description: Set up mestiere in a repository and manage the agent skills it installs, in a project or globally. Use when the user asks to set up or start using mestiere, or to add, remove, update, sync, inspect, customize or commit skills, when skills.yaml or .agents/skills is involved, or when mestiere reports drift or local changes.
---

# Managing installed skills with mestiere

`mestiere` installs Agent Skills from git registries. A project's configuration is
`skills.yaml` at its root. Read `references/concepts.md` for vocabulary, file layout and the
safety rules before acting on an unfamiliar request.

## Before you start

1. Check the CLI is available: `mestiere --version`. If it isn't, tell the user to install
   it (`brew install --cask polarity-dev/tap/mestiere`, or
   `go install github.com/polarity-dev/mestiere/cmd/mestiere@latest` followed by
   `mestiere add -g mestiere`) instead of editing skill folders by hand.
2. Run `mestiere status` to see installed skills, their link state, local skills and global
   skills. If it reports `no skills.yaml found`, set the project up first (below).
3. Never pass `--force` or delete skill folders without the user's explicit approval:
   both can destroy their local edits.

## Set up a project

1. At the repository root, run `mestiere init --agents claude,kiro`. Pass only the agents the
   team uses; existing `.claude/` or `.kiro/` folders are a good hint. Ask when it's unclear.
2. Add the team's registry with `mestiere registry add <name> <git-url>`. If the user didn't
   give one, ask for it.
3. Run `mestiere registry show <name>`, propose the skills that fit the codebase, and install
   the ones the user agrees to.
4. Commit `skills.yaml` and `.gitignore`. Tell the user teammates run `mestiere sync` after
   cloning, and that CI can run `mestiere sync --frozen` and `mestiere doctor --strict`.

This skill is installed globally, not per project. If a teammate's agent lacks it, they run
`mestiere add -g mestiere` once; the Homebrew install already does.

## Tasks

**Discover skills.** `mestiere registry list` shows configured registries.
`mestiere registry show [registry]` lists every namespace and skill with its description
and whether it's installed. Use it to pick skills that match the user's request.

**Add skills.** `mestiere add <namespace>` installs a whole namespace and follows it on
update. `mestiere add <namespace>/<skill>` installs one skill. Add `-r <registry>` when
several registries are configured. Always pass targets: without them `add` opens an
interactive picker you can't use.

**Remove skills.** `mestiere remove <target>`. Removing one skill of an installed namespace
adds it to `exclude` instead. Modified skills are kept and reported as local skills.

**Update.** `mestiere outdated` lists skills whose content changed upstream.
`mestiere update` moves everything; `mestiere update <target>` moves only that target.
Summarize what changed for the user before and after updating.

**Sync.** After a pull, a branch switch or a hand edit of `skills.yaml`, run
`mestiere sync`. Use `mestiere sync --dry-run` to preview. In CI use `mestiere sync --frozen`.

**Customize a registry skill.** Don't edit files under `.agents/skills/<name>` in place:
the next sync refuses to continue. Run `mestiere eject <name>` first, then edit. Add
`--track` if the customized skill should be committed.

**Commit a skill.** Installed skills are ignored by git. `mestiere track <name>` adds the
`.gitignore` exceptions; `mestiere untrack <name>` removes them. New hand-written skills in
`.agents/skills/`, `.claude/skills/` or `.kiro/skills/` also need `track` before teammates get them.

**Agents.** Edit `agents` in `skills.yaml` (supported: `claude`, `kiro`), then run
`mestiere sync` to add or remove the links.

**Add a registry.** `mestiere registry add <name> <git-url> [--ref <branch|tag|commit>]`.
Access uses the user's git credentials, so first check `git ls-remote <url>` works.

**Global skills.** Add `-g` to `add`, `remove`, `update`, `outdated`, `sync`, `status`,
`doctor`, `eject` and the `registry` subcommands to act on skills for every project on the
machine. They live in `~/.agents/skills/`, linked into the agents' folders under `~`, with the
manifest at `~/.config/mestiere/skills.yaml`. `track`, `untrack` and `init` don't apply. Suggest
global installs only for personal skills; skills the team relies on belong in the project.

## When something fails

Run `mestiere doctor` and act on each `error:` line:

| Message | Action |
|---|---|
| `has local changes` | Ask the user: keep them (`mestiere eject <name>`) or discard them (`mestiere sync --force`) |
| `install and lock disagree` (with `--frozen`) | Run `mestiere sync` and commit `skills.yaml` |
| `already exists and isn't a published version` | A local skill has the same name; rename it, or ask before `--force` |
| `comes from both <a> and <b>` | Add the skill to `exclude` in one of the registries |
| `isn't reachable` | A git access problem; check the user's credentials for that URL |
| `is committed but ignored` | Suggest `git rm -r --cached <path>`, or `mestiere track <name>` to keep committing it |

After any change, commit `skills.yaml` (and `.gitignore` if it changed). Skill folders stay
out of git unless tracked.
