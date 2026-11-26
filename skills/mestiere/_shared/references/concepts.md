# mestiere concepts

mestiere installs Agent Skills from git registries into a repository. Read this
before running commands you haven't used yet.

## Vocabulary

- **Registry**: a git repository with `registry.yaml` and `skills/<namespace>/<namespace>-<skill>/SKILL.md`.
- **Namespace**: a folder of related skills, e.g. `frontend` or `ai-tools`. Lowercase letters, digits and single hyphens.
- **Skill**: a folder with `SKILL.md` plus optional scripts, references and assets.
  Names use lowercase letters, digits and single hyphens.
- **Installed name**: `<namespace>-<skill>`, e.g. `frontend/forms` installs as `frontend-forms`.
  The `name` in the skill's frontmatter must equal it.
- **`_shared/`**: a namespace folder whose files skills reach through relative symlinks.
  On install every such link becomes a real copy, so installed skills are self-contained.

- **Skills-layout registry**: a repository published for `npx skills` (no `registry.yaml`,
  skills at `skills/<name>/SKILL.md`), such as `vercel/next.js`. `skills.yaml` marks it with
  `layout: skills`. Its skills have no namespace and install under their own name.

## Project files

- `skills.yaml` holds `agents`, and per registry `url`, `ref`, `install`, `exclude` and `lock`.
  `install`, `exclude` and `agents` may be edited by hand. Never edit `lock`; mestiere writes it.
- Installed files live in `.agents/skills/<installed-name>/`. Each configured agent gets a
  relative symlink: `.claude/skills/<name>`, `.kiro/skills/<name>`.
- `.gitignore` contains `**/.*/skills/*`, so every skill folder is ignored unless a
  `!<dir>/skills/<name>` exception commits it.
- Global skills live in `~/.agents/skills/`, `~/.claude/skills/` and `~/.kiro/skills/`, and
  agents load them in every project. `mestiere status` lists them. `-g` manages the ones
  listed in `~/.config/mestiere/skills.yaml`; the rest belong to the user, as in a project.

## Target syntax

Commands taking targets accept `frontend` (namespace), `frontend/forms` or `frontend-forms`
(skill), and `polarity:frontend` (registry-qualified). With several registries configured,
`add` needs `-r <registry>`.

## Safety rules mestiere enforces

- It only manages skills listed in `lock`. Everything else in a skills folder belongs to the user.
- It replaces or deletes a folder only when the folder's content matches a version published
  in the registry. Edited folders make `sync` stop with an error.
- Hidden files (names starting with `.`) are ignored in installed skills and rejected in registries.
- A skill's top-level `README.md` (any case) is for people browsing the registry and is never installed.

## Running mestiere from an agent

Every command works without prompts. Pass `--no-input` on every command you run, even when
your shell looks like a terminal: a prompt has no one to answer it and blocks your loop.
Write all arguments explicitly, such as targets for `add` and `--agents` for `init`. With
`--no-input`, a command missing an argument fails at once and the error names what to pass.
mestiere also turns prompts off by itself when `CI` or `MESTIERE_NO_INPUT` is set, or when it
recognizes an agent shell.

`status`, `registry show`, `outdated` and `doctor` accept `--json` when you need to parse
the result; the plain text carries the same content.

Mistyped commands, flags and names fail with a `did you mean …?` suggestion. Read it,
check it matches what the user asked for, then retry.

## Exit status

Commands exit non-zero on any `error:` line. `warning:` and `hint:` lines don't fail the command.
