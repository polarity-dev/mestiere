# AGENTS.md

mestiere is a Go CLI that installs Agent Skills from git registries into projects. This repository is also a registry: `registry.yaml` and `skills/mestiere/` hold the skills that teach agents to use the CLI, and Homebrew installs them globally.

Read [docs/architecture.md](docs/architecture.md) before changing anything in `internal/project`, `internal/registry` or `internal/fsx`. User-facing behavior is documented in [docs/guide.md](docs/guide.md) and [docs/registry.md](docs/registry.md).

## Commands

Run all of these before calling a change done. They match the release workflow, and a failure there blocks the release.

```sh
test -z "$(gofmt -l .)"           # formatting
go vet ./...
go test -race ./...               # uses real git repositories in temp dirs
go run ./cmd/mestiere registry lint
goreleaser check                  # only if .goreleaser.yaml changed
```

Requires Go 1.27.3+ and `git` on `PATH`.

## Layout

- `cmd/mestiere/main.go`: entry point only.
- `internal/cli`: Cobra commands and output. `NewRootCmd(stdout, stderr, interactive)` is the entry for tests.
- `internal/project`: resolve, sync, status, doctor and the add/remove/update/eject/track commands.
- `internal/registry`: bare mirrors, catalog, materializing skills, history lookups.
- `internal/fsx`: git-compatible tree hashing, safe copy and replace.
- `internal/manifest`: `skills.yaml` read and write, comments preserved.
- `internal/authoring`: `registry init/new-*` scaffolding and `registry lint`.
- `internal/gitignore`, `internal/naming`, `internal/skillmd`, `internal/agents`: small single-purpose packages.
- `internal/testutil`: helpers for throwaway git repos and registries.

## Invariants

These are the product. Don't weaken them to make a change easier.

- Never replace or delete a skill folder unless its tree hash matches a version published in the registry's history. Anything else is reported and left alone, unless the user passes `--force`.
- The lock in `skills.yaml` is the only state. Don't add marker files, databases or hidden state.
- Shell out to the system `git` binary. Don't add a Go git library; auth must behave exactly like the user's own `git`.
- Writes go to a temporary sibling folder and are renamed into place.
- Installs are flat (`.agents/skills/<ns>-<skill>`); namespaces exist only in the registry and in names.
- `manifest` must preserve user comments and ordering in `skills.yaml`.

## Conventions

- Keep dependencies to the current set (cobra, huh, yaml.v3). Ask before adding one.
- Tests that touch git or the filesystem call `testutil.Isolate(t)` first, so they never read the developer's git config, cache or home-level skills. Build registries with `testutil.NewRegistry`, not fixtures on disk.
- Prefer integration-style tests that drive real git repos over mocks.
- Comments explain why, not what. A known limitation gets `// shortcut: <limit>, <when to upgrade>`.
- Commit messages use conventional prefixes (`feat:`, `fix:`, `docs:`, `test:`, `chore:`); `docs:`, `test:` and `chore:` are left out of the changelog.

## Bundled skills

`skills/mestiere/` is shipped to users, so treat it like code.

- `internal/cli/bundled_test.go` lints the registry and checks that every `mestiere …` command and `--flag` mentioned in the skills exists. When you rename or remove a command or flag, update the skills in the same change.
- `references/concepts.md` in each skill is a symlink to `_shared/references/concepts.md`. Edit the shared file and keep the links.
- When CLI behavior changes, update `docs/` and the matching `SKILL.md` too.

## Docs style

Short, plain sentences in present tense. No em dashes, no exclamation marks, no hype. Match the tone of the README.

## Don't

- Don't push tags or run `goreleaser release`. Tagging `vX.Y.Z` publishes a release and updates the `polarity-dev/homebrew-tap` cask.
- Don't hand-edit the cask in `polarity-dev/homebrew-tap`. GoReleaser generates `Casks/mestiere.rb` from the `homebrew_casks` section of `.goreleaser.yaml`, so change it there.
- Don't commit `dist/`, built binaries or anything under `.agents/`, `.claude/` or `.kiro/skills/`.
- Windows isn't supported. Don't add Windows-specific code paths unless asked.
