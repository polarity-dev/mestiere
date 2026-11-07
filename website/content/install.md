---
title: Install mestiere
seoTitle: Install the mestiere CLI on macOS and Linux
nav: Install
description: Install the mestiere CLI on macOS or Linux with Homebrew, a prebuilt release or go install, give your agents its skills, and set up completions.
order: 1
---

mestiere is a single static binary for macOS and Linux, on amd64 and arm64.

The latest version is **{{version}}**. <!-- html-only -->

## Requirements

- macOS or Linux. Windows isn't supported yet.
- `git` on your `PATH`. mestiere uses your existing git setup (SSH agent, credential helpers, `gh auth`), so private registries work wherever `git clone` does.

## Install with Homebrew

On macOS and Linux:

```sh
brew install --cask polarity-dev/tap/mestiere
```

The cask also installs mestiere's own skills globally (`mestiere add -g mestiere`), so Claude Code and Kiro can set mestiere up in any repository: see [Agents](/docs/agents/#teach-your-agent-to-use-mestiere). On macOS it clears Gatekeeper's quarantine flag, because the binaries aren't notarized yet.

## Install from a release

Prebuilt archives are attached to every [GitHub release](https://github.com/polarity-dev/mestiere/releases). This detects your system, downloads the latest archive and installs the binary in `/usr/local/bin`:

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
curl -fsSL "https://github.com/polarity-dev/mestiere/releases/latest/download/mestiere_${OS}_${ARCH}.tar.gz" | tar -xz mestiere
sudo mv mestiere /usr/local/bin/
```

Archives have the same name in every release, so `releases/latest/download/` always serves the newest one:

| System | Archive |
|---|---|
| macOS, Apple silicon | `mestiere_darwin_arm64.tar.gz` |
| macOS, Intel | `mestiere_darwin_amd64.tar.gz` |
| Linux, x86-64 | `mestiere_linux_amd64.tar.gz` |
| Linux, ARM64 | `mestiere_linux_arm64.tar.gz` |

For a specific version, replace `releases/latest/download/` with `releases/download/v<version>/`.

To install without `sudo`, move the binary to any folder on your `PATH` instead, such as `~/.local/bin`.

### Verify the download

Each release has a `checksums.txt` with the SHA-256 of every archive:

```sh
BASE="https://github.com/polarity-dev/mestiere/releases/latest/download"
FILE="mestiere_${OS}_${ARCH}.tar.gz"
curl -fsSLO "$BASE/$FILE" && curl -fsSLO "$BASE/checksums.txt"
grep " $FILE\$" checksums.txt | shasum -a 256 -c -
```

### macOS Gatekeeper

Binaries aren't notarized by Apple yet. A download made with `curl` runs as is. If you downloaded the archive with a browser, macOS blocks the binary; clear the flag with:

```sh
xattr -d com.apple.quarantine mestiere
```

## Install with Go

With Go 1.27.3 or later:

```sh
go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
```

The binary goes to `$(go env GOPATH)/bin`; make sure that folder is on your `PATH`.

## Give your agents mestiere's skills

The Homebrew install does this for you. After a release or `go install`, run once:

```sh
mestiere add -g mestiere
```

This installs the `mestiere-project` and `mestiere-registry` skills for every project on your machine, in `~/.agents/skills/`, with links for the agents that have a folder in your home directory. See [Global skills](/docs/global-skills/).

## Check the installation

```sh
mestiere --version
```

## Shell completions

mestiere generates completions for bash, zsh and fish:

```sh
# bash
echo 'source <(mestiere completion bash)' >> ~/.bashrc

# zsh (with compinit enabled)
mestiere completion zsh > "${fpath[1]}/_mestiere"

# fish
mestiere completion fish > ~/.config/fish/completions/mestiere.fish
```

Run `mestiere completion <shell> --help` for other setups.

## Upgrade

```sh
brew upgrade --cask mestiere
```

The cask also updates mestiere's own global skills. With a release, run the install again: it always downloads the latest version. With Go, run `go install ...@latest`. Then run `mestiere update -g mestiere`. Projects don't depend on the mestiere version: `skills.yaml` stays the same.

## Uninstall

```sh
mestiere remove -g mestiere         # remove mestiere's global skills first
mestiere cache clean                # optional: remove the registry cache
brew uninstall --cask mestiere      # or: sudo rm /usr/local/bin/mestiere
```

The cache lives in `~/Library/Caches/mestiere` on macOS and `~/.cache/mestiere` on Linux. Projects keep their `skills.yaml` and installed skills.

## Next steps

- [Getting started](/docs/): install your first skills in a project.
- [Registries](/docs/registries/): publish your team's skills.
- [Teach your agent to use mestiere](/docs/agents/#teach-your-agent-to-use-mestiere).
