---
title: CI and configuration
seoTitle: Run mestiere in CI and configure the cache
nav: CI and configuration
description: Check agent skills in CI with mestiere sync --frozen and doctor --strict, give CI access to private registries, and configure the registry cache.
order: 8
---

## Check a project in CI

Two commands catch every inconsistency:

```yaml
- run: mestiere sync --frozen      # fail if skills.yaml is inconsistent
- run: mestiere doctor --strict    # fail on drift, broken links or uncommitted local skills
```

## GitHub Actions example

```yaml
name: skills

on:
  pull_request:
  push:
    branches: [main]

jobs:
  skills:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version: stable
      - run: go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
      - run: mestiere sync --frozen
      - run: mestiere doctor --strict
```

To skip the Go toolchain, download a release archive instead: see [Install](/install/#install-from-a-release).

## Private registries

mestiere uses whatever git is configured with. Give the CI job credentials that can read each registry, for example:

- an SSH deploy key loaded into `ssh-agent`, for `git@` URLs;
- a token in a git credential helper, or a URL rewrite such as `git config --global url."https://x-access-token:${TOKEN}@github.com/".insteadOf "https://github.com/"`, for HTTPS URLs.

Check access with `git ls-remote <url>` in the same job. `mestiere doctor` reports a registry it can't reach as `isn't reachable`.

## Check a registry in CI

In a registry repository, run lint on every pull request:

```yaml
- run: mestiere registry lint
```

See [what lint checks](/docs/registries/#what-lint-checks).

## Configuration

mestiere has no config file of its own: a project is configured in [skills.yaml](/docs/skills-yaml/), global skills in `~/.config/mestiere/skills.yaml`. Environment variables control where files go:

| Variable | Effect |
|---|---|
| `MESTIERE_CACHE_DIR` | Cache location (highest precedence) |
| `XDG_CACHE_HOME` | Cache goes to `$XDG_CACHE_HOME/mestiere`, honored on macOS too |
| `XDG_CONFIG_HOME` | The [global](/docs/global-skills/) manifest goes to `$XDG_CONFIG_HOME/mestiere/skills.yaml` instead of `~/.config/mestiere/skills.yaml` |

By default the cache is `~/Library/Caches/mestiere` on macOS and `~/.cache/mestiere` on Linux. It holds one bare mirror per registry and is safe to delete: `mestiere cache clean` removes it, and the next command re-fetches what it needs. In CI, cache that folder between runs to save fetches.
