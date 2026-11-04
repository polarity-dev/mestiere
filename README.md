<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/mestiere-wordmark-white.svg">
    <img src="docs/assets/mestiere-wordmark-black.svg" alt="mestiere" width="480">
  </picture>
</p>

<p align="center">
  Your team's agent skills, versioned in git and installed into every repository.
</p>

<p align="center">
  <a href="https://github.com/polarity-dev/mestiere/releases"><img src="https://img.shields.io/github/v/release/polarity-dev/mestiere?color=000000" alt="Latest release"></a>
  <a href="https://github.com/polarity-dev/mestiere/actions/workflows/release.yml"><img src="https://github.com/polarity-dev/mestiere/actions/workflows/release.yml/badge.svg" alt="Release status"></a>
</p>

---

*Mestiere* is Italian for craft: the know-how a workshop hands from one person to the next. mestiere does that for coding agents. Your team keeps its [Agent Skills](https://agentskills.io) in one git repository, organized by namespace. Each project then installs the skills it needs, pinned to an exact version, and every agent harness in the repo sees them.

```sh
mestiere init --agents claude,kiro
mestiere registry add polarity git@github.com:polarity-dev/skills.git
mestiere add frontend backend/db-migrations
```

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/mestiere-hero-dark.png">
    <img src="docs/assets/mestiere-hero.png" alt="Skills flow from a git registry with frontend, backend and infra namespaces into .agents/skills, which links them to .claude and .kiro." width="820">
  </picture>
</p>

## Quick start with your agent

```sh
brew install --cask polarity-dev/tap/mestiere
```

The install also gives Claude Code and Kiro mestiere's own skills globally. If you installed another way, run `mestiere add -g mestiere` once. Then ask your agent, from any repository:

```text
Set up mestiere for this repo with our registry git@github.com:polarity-dev/skills.git
```

The agent runs `mestiere init`, shows you what the registry offers, installs the skills that fit the codebase and commits `skills.yaml`. Later prompts work the same way: "Are any of our skills outdated? Update them and summarize what changed", or "I want to tweak frontend-forms for this repo, keep it as our own and commit it".

| Skill | For |
|---|---|
| `mestiere-project` | Repositories that install skills: adding, updating, syncing, ejecting, committing, fixing `doctor` errors |
| `mestiere-registry` | Registry repositories: scaffolding, writing `SKILL.md`, sharing files through `_shared/`, fixing lint errors, CI |

## Why mestiere

- **One source of truth.** Skills live in a git registry, public or private, reviewed like any other code.
- **Organized by namespace.** Install a whole namespace (`frontend`) or a single skill (`backend/db-migrations`). New skills in an installed namespace arrive on the next update.
- **Reproducible.** Every install is locked to a registry commit, and each skill gets a content hash. Teammates and CI get byte-identical skills.
- **Every agent at once.** Files live once in `.agents/skills/`, which Codex, Gemini CLI, Cursor and others read directly. Agents with their own folder, like Claude Code and Kiro, get symlinks to it.
- **Out of your way.** Installed skills are ignored by git by default. Your own skills sit next to them untouched, and committing any skill is one command.
- **Safe by construction.** mestiere only replaces or deletes a folder when it can prove the folder is an unmodified copy of a published skill. Edited skills are never overwritten silently.
- **No lock-in.** Registries are plain git repositories. Installed skills are plain folders that follow the Agent Skills spec.

## Install

**With Homebrew** (macOS and Linux):

```sh
brew install --cask polarity-dev/tap/mestiere
```

**From a release.** Prebuilt archives for macOS and Linux (amd64 and arm64) are attached to every [GitHub release](https://github.com/polarity-dev/mestiere/releases):

```sh
# Pick darwin or linux, and arm64 or amd64
OS=darwin ARCH=arm64
curl -fsSL "https://github.com/polarity-dev/mestiere/releases/latest/download/mestiere_${OS}_${ARCH}.tar.gz" | tar -xz mestiere
sudo mv mestiere /usr/local/bin/
```

Binaries aren't notarized by Apple yet. A download made with `curl` runs as is. If you downloaded the archive with a browser, macOS blocks the binary; clear the flag with `xattr -d com.apple.quarantine mestiere`.

**With Go 1.27.3 or later:**

```sh
go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
mestiere add -g mestiere    # give your agents mestiere's own skills
```

**Requirements:** `git` on your `PATH`. mestiere uses your existing git setup (SSH agent, credential helpers, `gh auth`), so private registries work wherever `git clone` does. macOS and Linux are supported; Windows isn't yet. mestiere checks for a newer version once a day and tells you when one exists; set `MESTIERE_NO_UPDATE_CHECK=1` to turn it off.

## Quick start

### In a project

```sh
mestiere init --agents claude,kiro        # creates skills.yaml and the .gitignore rule
mestiere registry add polarity git@github.com:polarity-dev/skills.git
mestiere add frontend                     # whole namespace
mestiere add backend/db-migrations        # single skill
git add skills.yaml .gitignore && git commit -m "Add team skills"
```

A teammate who clones the repository runs:

```sh
mestiere sync
```

In CI, `mestiere sync --frozen` and `mestiere doctor --strict` fail on an inconsistent `skills.yaml` or drifted skills.

### In a registry

A registry is its own git repository, with `registry.yaml` at the root:

```sh
mkdir my-registry && cd my-registry && git init
mestiere registry init --name my-registry   # creates registry.yaml and skills/
mestiere registry new-namespace frontend --description "Frontend practices"
mestiere registry new-skill frontend/forms --description "Build accessible forms"
mestiere registry lint                    # run this in the registry's CI
```

## Supported agents

Skills are installed once in `.agents/skills/`, the shared location that Codex, Gemini CLI, Cursor, GitHub Copilot and other agents read directly. Agents with their own folder, like Claude Code (`.claude/skills/`) and Kiro (`.kiro/skills/`), get links to it.

## Documentation

- [Using mestiere in a project](docs/guide.md): `skills.yaml`, commands, updates, your own skills, CI, configuration
- [Authoring a registry](docs/registry.md): layout, shared files, lint rules
- [Architecture](docs/architecture.md): safety model, packages, design decisions
- [Development](docs/development.md): tests and releases
- [Brand assets](docs/brand.md)

## License

[MIT](LICENSE)
