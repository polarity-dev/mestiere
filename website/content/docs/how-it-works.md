---
title: How it works
seoTitle: "How mestiere works: locks, updates and the safety model"
nav: How it works
description: How mestiere locks skills to registry commits, detects updates per skill with git tree hashes, and never overwrites a skill you edited.
order: 7
---

## How updates work

mestiere locks each skill to a registry commit, the same way package lockfiles do. Registry authors don't need to tag versions.

- `add` resolves the registry's `ref` and locks the new skills at that commit.
- `outdated` fetches the registry and lists skills whose content changed. A new commit that doesn't touch a skill isn't reported.
- `update` moves the lock forward, for everything or only for the targets you name. Updating a namespace also picks up skills added to it and drops skills deleted from it.
- `sync` installs exactly what the lock says. `sync --frozen` fails if `install` and `lock` disagree, which is what CI should run.

Pin a whole project with `ref: v2026.10` (a tag) or a commit ID.

## Safety model

The lock is the only state mestiere keeps. There's no hidden database or marker file to get out of sync. Before replacing or deleting a folder, mestiere checks the folder itself:

1. It computes the git tree hash of the folder on disk, skipping hidden files such as `.DS_Store` and editor swap files.
2. It looks up the registry skill that matches the folder's name, and every version that skill has had in the registry's history.
3. Only if the hash matches one of those versions is the folder an untouched install, and only then is it safe to replace or remove.

For each folder in a skills directory:

- **Listed in the lock, hash matches the locked version:** nothing to do.
- **Listed in the lock, hash matches an older published version:** upgraded to the locked version.
- **Listed in the lock, hash matches no published version:** `sync` stops. The folder has local changes; use `--force` or `eject`.
- **Not in the lock, name and content match a published skill:** removed, because it was dropped from `skills.yaml`.
- **Not in the lock, anything else:** left alone. It's your skill.

`--force` is always an explicit choice. The same check makes routine team workflows work without extra steps:

| Situation | Result |
|---|---|
| A teammate updates a skill; you pull and run `sync` | Your older, unmodified copy is upgraded |
| A teammate removes a skill; you pull and run `sync` | The leftover folder is removed |
| You edited an installed skill | `sync` stops and suggests `--force` or `eject` |
| You copied a skill to make your own | The copy is never touched |
| Offline, with a cached registry | `sync` installs from the cache |

## Architecture

- **Registry (git):** `skills/<namespace>/<skill>/SKILL.md` and `skills/<namespace>/_shared/`.
- **Local cache:** one bare git mirror per registry, filled with `git fetch`.
- **Project:** `skills.yaml` (install and lock), `.agents/skills/<namespace>-<skill>/` with the real files, and symlinks from `.claude/skills/` and `.kiro/skills/`.

`add` and `update` resolve `install` against the registry mirror and write `lock`. `sync` compares the lock with the skill folders on disk and applies only the steps it can prove are safe: install, upgrade, link, unlink, remove.

## Design decisions

- **The system `git` binary, not a Go git library,** so authentication works exactly as it does for your own `git`.
- **Git tree hashes as content identity.** A skill's identity is the tree ID of its installed folder, so changes are detected per skill. For skills without shared links it's the registry's own tree ID.
- **History as proof of ownership.** Ownership is never recorded in a state file. It's decided by whether a folder's content was ever published.
- **Flat installs, namespaced registries.** Harnesses discover skills one level deep, so namespaces exist in the registry and in the installed names, not as folders.
- **Writes go through a temporary sibling folder,** which is then renamed into place, so an interrupted run never leaves a half-written skill.
