---
title: Registries
seoTitle: Create an Agent Skills registry with mestiere
nav: Registries
description: Create a git registry of Agent Skills with mestiere, organize it in namespaces, share files between skills, and validate it with registry lint.
order: 4
---

A registry is a git repository of skills that projects install from. It can be public or private, and needs no server: mestiere reads it with `git`. mestiere also installs skills from repositories published for `npx skills`: see [Skills from vercel/skills repositories](#skills-from-vercelskills-repositories).

## Create a registry

```sh
mkdir skills && cd skills && git init
mestiere registry init --name polarity --description "Polarity's team skills"
mestiere registry new-namespace frontend --description "Frontend practices" --owners web-team
mestiere registry new-skill frontend/forms --description "Build accessible forms"
mestiere registry lint
```

Commit and push. Projects add it with `mestiere registry add polarity <git-url>`.

## Layout

```
skills/                                   # registry repository
├── registry.yaml                         # schemaVersion, name, description
└── skills/
    ├── frontend/
    │   ├── namespace.yaml                # description, owners
    │   ├── _shared/                      # files several skills link to
    │   │   ├── scripts/lint-a11y.sh
    │   │   └── references/design-tokens.md
    │   ├── frontend-forms/               # folder = installed name
    │   │   ├── SKILL.md                  # name: frontend-forms
    │   │   ├── README.md                 # for people browsing the registry, not installed
    │   │   ├── references/
    │   │   │   └── design-tokens.md -> ../../_shared/references/design-tokens.md
    │   │   └── scripts/
    │   │       └── lint-a11y.sh -> ../../_shared/scripts/lint-a11y.sh
    │   └── frontend-tables/
    │       ├── SKILL.md
    │       └── scripts -> ../_shared/scripts
    └── backend/
        └── backend-db-migrations/
            ├── SKILL.md
            └── scripts/migrate.sh
```

## Naming rules

- **Namespaces** and **skills** use lowercase letters, digits and single hyphens: `frontend`, `ai-tools`, `forms`, `db-migrations`. Choose namespaces by domain or team.
- The **installed name** is `<namespace>-<skill>`, e.g. `frontend-forms`, at most 64 characters.
- A skill's folder is its installed name: `skills/frontend/frontend-forms/`, as the Agent Skills spec requires.
- No two skills may share an installed name: `ai/code-review` and `ai-code/review` both install as `ai-code-review`, so lint rejects the pair.

## Writing SKILL.md

`new-skill` generates a `SKILL.md` with frontmatter. Then:

- Keep the generated `name`: it must equal the installed name, `<namespace>-<skill>`.
- Write a `description` of 1 to 1024 characters. It decides when an agent activates the skill: say what the skill does and when to use it, with the words a user would actually say.
- Keep `SKILL.md` focused. Move long material into `references/` and link to it by relative path.
- Refer to bundled files by paths relative to the skill folder, e.g. `scripts/check.sh`.
- Make scripts executable (`chmod +x`) and give them a shebang.

## Skill README

A skill folder may hold a `README.md` next to `SKILL.md`, in any case (`readme.md` works too). Use it to present the skill to people: what it's for, use cases, examples. Git hosts show it when someone browses the folder. mestiere never installs it, so it doesn't reach the agent's context. Don't link to it from `SKILL.md`.

Only the top-level file is skipped; a `README.md` inside a subfolder is installed like any other file.

## Sharing files between skills

Put a file in the namespace's `_shared/` folder and link it from each skill with a relative symlink, to a single file or to a whole folder:

```sh
cd skills/frontend/frontend-forms
mkdir -p scripts
ln -s ../../_shared/scripts/lint-a11y.sh scripts/lint-a11y.sh   # one file
ln -s ../_shared/references references                           # a whole folder
```

On install, mestiere replaces each link with a real copy of its target. Every installed skill is self-contained, and its `SKILL.md` keeps using paths like `scripts/lint-a11y.sh`. When a shared file changes, only the skills that link to it are reported as updated.

Links may only point inside the skill or into the same namespace's `_shared/`. Files inside `_shared/` must be real files, not links.

## What lint checks

`mestiere registry lint` fails on any `error:` line:

- Namespace, skill and installed names follow the naming rules, and no two skills share an installed name.
- Each skill lives in `skills/<namespace>/<namespace>-<skill>/`.
- `SKILL.md` has frontmatter whose `name` equals the installed name, and a description of 1 to 1024 characters.
- Links stay inside the skill or point into the same namespace's `_shared/`, and links into `_shared/` reach a real file.
- Skills and `_shared/` contain no hidden files. Hidden files are skipped when installed copies are compared with the registry, so a published hidden file could never match.

Shared files that no skill links to produce a warning, which doesn't fail lint.

| Error | Fix |
|---|---|
| `name must be "<ns>-<skill>"` | Set the frontmatter `name` to the installed name |
| `skill folders are named <ns>-<skill>` | Rename the folder to the installed name (`git mv skills/<ns>/<skill> skills/<ns>/<ns>-<skill>`) |
| `invalid namespace` / `invalid skill name` | Rename the folder to match the naming rules |
| `installs as <name>, like <folder>` | Two skills share an installed name; rename one of them |
| `hidden files aren't installed` | Delete the file, or rename it without the leading dot (`.env.example` → `env.example`) |
| `links outside the skill and skills/<ns>/_shared/` | Move the target into `_shared/` and link it from there |
| `which is itself a link` / `links aren't allowed inside _shared` | Replace the link in `_shared/` with the real file |

## Test a skill before publishing

From a project, add your local checkout as a registry and install from it. Only committed content is installed, so commit on a branch first:

```sh
mestiere registry add local /path/to/registry --ref my-branch
mestiere add frontend/forms
mestiere registry remove local
```

## Renaming and removing skills

- Projects that installed the whole namespace drop a removed skill on their next `mestiere update`.
- Projects that installed the skill individually get an error and must remove it, so announce removals.
- A rename is a removal plus an addition.

## Run lint in CI

```yaml
- uses: actions/setup-go@v7
  with:
    go-version: stable
- run: go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
- run: mestiere registry lint
```

More CI setups: [CI](/docs/ci/).

## Skills from vercel/skills repositories

Some projects publish skills for `npx skills` instead of as a mestiere registry, such as Next.js in `vercel/next.js`. mestiere installs them too:

```sh
mestiere registry add next https://github.com/vercel/next.js
mestiere registry show next
mestiere add -r next next-dev-loop
```

`registry add` looks at the repository. With no `registry.yaml` at the root and at least one `skills/<name>/SKILL.md`, it records `layout: skills` in `skills.yaml`:

```yaml
registries:
  next:
    url: https://github.com/vercel/next.js
    layout: skills
    install:
      - next-dev-loop
```

- Skills install under their own name, the folder name `npx skills` would create, so `next-dev-loop` lands in `.agents/skills/next-dev-loop/`.
- Only `skills/<name>/SKILL.md` counts. mestiere doesn't search other folders, or read `skills.sh.json` or plugin manifests.
- `install` lists skill names. There are no namespaces, so there's no `exclude`.
- The lock, `sync`, `update`, `outdated`, `doctor` and the [safety model](/docs/how-it-works/#safety-model) work as for any registry.
- The mirror follows only `ref` (the default branch when unset) and downloads file contents only for the skills you install. `vercel/next.js` takes about 130 MB in the cache instead of several gigabytes. `ref` must be a branch or a tag.

Use a full git URL, as for any registry. Private repositories work with your git credentials.
