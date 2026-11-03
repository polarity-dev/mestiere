---
title: Installa mestiere
seoTitle: Installa la CLI di mestiere su macOS e Linux
nav: Installazione
description: Installa la CLI di mestiere su macOS o Linux con Homebrew, una release precompilata o go install, dai le sue skill ai tuoi agenti e configura i completamenti.
order: 1
---

mestiere è un singolo binario statico per macOS e Linux, su amd64 e arm64.

L'ultima versione è **{{version}}**. <!-- html-only -->

## Requisiti {#requirements}

- macOS o Linux. Windows non è ancora supportato.
- `git` nel tuo `PATH`. mestiere usa la tua configurazione git esistente (SSH agent, credential helper, `gh auth`), quindi i registry privati funzionano ovunque funzioni `git clone`.

## Installa con Homebrew {#install-with-homebrew}

Su macOS e Linux:

```sh
brew install --cask polarity-dev/tap/mestiere
```

Il cask installa anche le skill di mestiere a livello globale (`mestiere add -g mestiere`), così Claude Code e Kiro possono configurare mestiere in qualsiasi repository: vedi [Agenti](/docs/agents/#teach-your-agent-to-use-mestiere). Su macOS rimuove il flag di quarantena di Gatekeeper, perché i binari non sono ancora notarizzati.

## Installa da una release {#install-from-a-release}

Gli archivi precompilati sono allegati a ogni [release su GitHub](https://github.com/polarity-dev/mestiere/releases). Questo script rileva il tuo sistema, scarica l'archivio più recente e installa il binario in `/usr/local/bin`:

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
curl -fsSL "https://github.com/polarity-dev/mestiere/releases/latest/download/mestiere_${OS}_${ARCH}.tar.gz" | tar -xz mestiere
sudo mv mestiere /usr/local/bin/
```

Gli archivi hanno lo stesso nome in ogni release, quindi `releases/latest/download/` dà sempre il più recente:

| Sistema | Archivio |
|---|---|
| macOS, Apple silicon | `mestiere_darwin_arm64.tar.gz` |
| macOS, Intel | `mestiere_darwin_amd64.tar.gz` |
| Linux, x86-64 | `mestiere_linux_amd64.tar.gz` |
| Linux, ARM64 | `mestiere_linux_arm64.tar.gz` |

Per una versione precisa, sostituisci `releases/latest/download/` con `releases/download/v<version>/`.

Per installare senza `sudo`, sposta invece il binario in una qualsiasi cartella nel tuo `PATH`, come `~/.local/bin`.

### Verifica il download {#verify-the-download}

Ogni release ha un `checksums.txt` con lo SHA-256 di ogni archivio:

```sh
BASE="https://github.com/polarity-dev/mestiere/releases/latest/download"
FILE="mestiere_${OS}_${ARCH}.tar.gz"
curl -fsSLO "$BASE/$FILE" && curl -fsSLO "$BASE/checksums.txt"
grep " $FILE\$" checksums.txt | shasum -a 256 -c -
```

### Gatekeeper di macOS {#macos-gatekeeper}

I binari non sono ancora notarizzati da Apple. Un download fatto con `curl` funziona così com'è. Se hai scaricato l'archivio con un browser, macOS blocca il binario; rimuovi il flag con:

```sh
xattr -d com.apple.quarantine mestiere
```

## Installa con Go {#install-with-go}

Con Go 1.27.3 o successivo:

```sh
go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
```

Il binario finisce in `$(go env GOPATH)/bin`; assicurati che quella cartella sia nel tuo `PATH`.

## Dai ai tuoi agenti le skill di mestiere {#give-your-agents-mestieres-skills}

L'installazione con Homebrew lo fa per te. Dopo una release o `go install`, esegui una volta:

```sh
mestiere add -g mestiere
```

Questo installa le skill `mestiere-project` e `mestiere-registry` per ogni progetto sulla tua macchina, in `~/.agents/skills/`, con i link per gli agenti che hanno una cartella nella tua home. Vedi [Skill globali](/docs/global-skills/).

## Controlla l'installazione {#check-the-installation}

```sh
mestiere --version
```

## Completamenti della shell {#shell-completions}

mestiere genera i completamenti per bash, zsh e fish:

```sh
# bash
echo 'source <(mestiere completion bash)' >> ~/.bashrc

# zsh (con compinit attivo)
mestiere completion zsh > "${fpath[1]}/_mestiere"

# fish
mestiere completion fish > ~/.config/fish/completions/mestiere.fish
```

Esegui `mestiere completion <shell> --help` per altre configurazioni.

## Aggiornamento {#upgrade}

```sh
brew upgrade --cask mestiere
```

Il cask aggiorna anche le skill globali di mestiere. Con una release, esegui di nuovo l'installazione: scarica sempre l'ultima versione. Con Go, esegui `go install ...@latest`. Poi esegui `mestiere update -g mestiere`. I progetti non dipendono dalla versione di mestiere: `skills.yaml` resta uguale.

## Disinstallazione {#uninstall}

```sh
mestiere remove -g mestiere         # prima rimuovi le skill globali di mestiere
mestiere cache clean                # opzionale: rimuove la cache dei registry
brew uninstall --cask mestiere      # oppure: sudo rm /usr/local/bin/mestiere
```

La cache si trova in `~/Library/Caches/mestiere` su macOS e in `~/.cache/mestiere` su Linux. I progetti mantengono il loro `skills.yaml` e le skill installate.

## Prossimi passi {#next-steps}

- [Primi passi](/docs/): installa le tue prime skill in un progetto.
- [Registry](/docs/registries/): pubblica le skill del tuo team.
- [Insegna al tuo agente a usare mestiere](/docs/agents/#teach-your-agent-to-use-mestiere).
