# Authoring a registry

A registry is a plain git repository of skills, organized by namespace. Any host that `git clone` can reach works, public or private.

```sh
mkdir my-registry && cd my-registry && git init
mestiere registry init --name my-registry   # creates registry.yaml and skills/
mestiere registry new-namespace frontend --description "Frontend practices"
mestiere registry new-skill frontend/forms --description "Build accessible forms"
mestiere registry lint                    # run this in the registry's CI
```

## Layout

```
my-registry/                              # registry repository
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

## Shared files

A skill links to files in its namespace's `_shared/` folder with a relative symlink, either to a single file or to a whole folder. On install, mestiere replaces each link with a real copy of the target. Every installed skill is therefore self-contained, and its `SKILL.md` can keep using paths like `scripts/lint-a11y.sh`. When a shared file changes, only the skills that link to it are reported as updated.

## Skill README

A skill folder may hold a `README.md` next to `SKILL.md`, in any case (`readme.md` works too). Use it to present the skill to people: what it's for, use cases, examples. Git hosts show it when someone browses the folder. mestiere never installs it, so it doesn't reach the agent's context. Only the top-level file is skipped; a `README.md` inside a subfolder is installed like any other file.

## What `mestiere registry lint` checks

- Namespace names use lowercase letters and digits. Skill names use lowercase letters, digits and single hyphens. Installed names are 64 characters at most.
- A skill lives in `skills/<namespace>/<namespace>-<skill>/`: the folder is the installed name, as the Agent Skills spec requires.
- `SKILL.md` has frontmatter whose `name` equals the installed name, and a description of 1 to 1024 characters.
- Links stay inside the skill or point into the same namespace's `_shared/`. Links into `_shared/` must reach a real file, not another link.
- Skills and `_shared/` contain no hidden files. Hidden files are skipped when installed copies are compared with the registry, so a published hidden file could never match.
- Shared files that no skill links to produce a warning.

## Versioning

Projects lock each skill to a registry commit, so you don't need to tag releases. Tags are still useful when consumers want to pin a whole registry, for example `ref: v2026.10`.
