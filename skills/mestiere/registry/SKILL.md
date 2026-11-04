---
name: mestiere-registry
description: Author and maintain a mestiere skill registry. Use when the user wants to create a registry, add namespaces or skills, share scripts or references between skills, fix `mestiere registry lint` errors, or set up CI for a registry repository.
---

# Maintaining a mestiere registry

A registry is a git repository that projects install skills from. Read
`references/concepts.md` for naming rules and how installs work.

## Layout

```
registry.yaml                         # schemaVersion: 1, name, description
skills/
  <namespace>/
    namespace.yaml                    # description, owners (optional)
    _shared/                          # optional: files several skills link to
    <skill>/
      SKILL.md                        # name: <namespace>-<skill>
      README.md                       # optional: for people, never installed
      scripts/ references/ assets/
```

## Tasks

**Check the CLI.** Run `mestiere --version`. If it's missing, ask the user to install it with
`brew install --cask polarity-dev/tap/mestiere`, or with
`go install github.com/polarity-dev/mestiere/cmd/mestiere@latest` followed by `mestiere add -g mestiere`.

**Create a registry.** In an empty git repository run `mestiere registry init --name <name>`.
Projects use it with `mestiere registry add <name> <git-url>`, which works with any URL their
git credentials can clone.

**Add a namespace.** `mestiere registry new-namespace <ns> --description "..." --owners team-a`.
Namespace names are lowercase letters and digits only. Choose them by domain or team
(`frontend`, `payments`), since installed names become `<ns>-<skill>`.

**Add a skill.** `mestiere registry new-skill <ns>/<skill> --description "..."`, then write
`SKILL.md`. Follow these rules:

- Keep the generated `name`; it must equal `<ns>-<skill>`.
- The `description` decides when an agent activates the skill. State what it does and when to
  use it, with the words a user would actually say. Keep it under 1024 characters.
- Keep `SKILL.md` focused. Move long material into `references/` and link to it by relative path.
- Refer to bundled files by paths relative to the skill folder, e.g. `scripts/check.sh`.
- Mark scripts executable (`chmod +x`) and give them a shebang.
- Put a description for people (use cases, examples) in an optional `README.md` next to
  `SKILL.md`. It shows on the registry's web page and is never installed, so it stays out of
  the agent's context. Don't link to it from `SKILL.md`.

**Share files between skills.** Put the file in `skills/<ns>/_shared/` and link it from each
skill with a relative symlink:

```sh
cd skills/<ns>/<skill>
mkdir -p scripts
ln -s ../../_shared/scripts/check.sh scripts/check.sh   # one file
ln -s ../_shared/references references                   # a whole folder
```

Installs replace links with real copies. Links may only point inside the skill or into the
same namespace's `_shared/`, never to another link. Files in `_shared/` can't be links.

**Rename or remove a skill.** Projects that installed the whole namespace drop a removed skill
on their next `mestiere update`. Projects that installed it individually get an error and must
remove it, so announce removals. Renaming is a removal plus an addition.

## Validate

Run `mestiere registry lint` after every change and fix every `error:` line before committing.
Common errors:

| Error | Fix |
|---|---|
| `name must be "<ns>-<skill>"` | Set the frontmatter `name` to the installed name |
| `invalid namespace` / `invalid skill name` | Rename the folder to match the naming rules |
| `hidden files aren't installed` | Delete the file, or rename it without the leading dot (`.env.example` → `env.example`) |
| `links outside the skill and skills/<ns>/_shared/` | Move the target into `_shared/` and link it from there |
| `which is itself a link` / `links aren't allowed inside _shared` | Replace the link in `_shared/` with the real file |

Warnings such as `no skill links to this file` don't fail lint; remove the unused file or link it.

## Test before publishing

From a project, add the local checkout as a registry and install from it:

```sh
mestiere registry add local /path/to/registry --ref <your-branch>
mestiere add <ns>/<skill>
mestiere registry remove local
```

Only committed content is installed, so commit on a branch first.

## CI

Run lint on every pull request:

```yaml
- uses: actions/setup-go@v7
  with:
    go-version: stable
- run: go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
- run: mestiere registry lint
```
