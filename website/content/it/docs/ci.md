---
title: CI e configurazione
seoTitle: Usa mestiere in CI e configura la cache
nav: CI e configurazione
description: Controlla le skill degli agenti in CI con mestiere sync --frozen e doctor --strict, dai alla CI accesso ai registry privati e configura la cache.
order: 8
---

## Controllare un progetto in CI {#check-a-project-in-ci}

Due comandi trovano ogni incoerenza:

```yaml
- run: mestiere sync --frozen      # fallisce se skills.yaml è incoerente
- run: mestiere doctor --strict    # fallisce su drift, link rotti o skill locali non committate
```

## Esempio con GitHub Actions {#github-actions-example}

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

Per fare a meno della toolchain Go, scarica invece un archivio di release: vedi [Installazione](/install/#install-from-a-release).

## Registry privati {#private-registries}

mestiere usa la configurazione che ha già git. Dai al job di CI credenziali che possano leggere ogni registry, per esempio:

- una deploy key SSH caricata in `ssh-agent`, per gli URL `git@`;
- un token in un credential helper di git, oppure una riscrittura dell'URL come `git config --global url."https://x-access-token:${TOKEN}@github.com/".insteadOf "https://github.com/"`, per gli URL HTTPS.

Verifica l'accesso con `git ls-remote <url>` nello stesso job. `mestiere doctor` segnala un registry che non riesce a raggiungere come `isn't reachable`.

## Controllare un registry in CI {#check-a-registry-in-ci}

In un repository di registry, esegui lint su ogni pull request:

```yaml
- run: mestiere registry lint
```

Vedi [cosa controlla lint](/docs/registries/#what-lint-checks).

## Configurazione {#configuration}

mestiere non ha un suo file di configurazione: un progetto si configura in [skills.yaml](/docs/skills-yaml/), le skill globali in `~/.config/mestiere/skills.yaml`. Le variabili d'ambiente decidono dove finiscono i file:

| Variabile | Effetto |
|---|---|
| `MESTIERE_CACHE_DIR` | Posizione della cache (precedenza massima) |
| `XDG_CACHE_HOME` | La cache va in `$XDG_CACHE_HOME/mestiere`, rispettata anche su macOS |
| `XDG_CONFIG_HOME` | Il manifest [globale](/docs/global-skills/) va in `$XDG_CONFIG_HOME/mestiere/skills.yaml` invece che in `~/.config/mestiere/skills.yaml` |

Per default la cache è `~/Library/Caches/mestiere` su macOS e `~/.cache/mestiere` su Linux. Contiene un mirror bare per ogni registry e puoi cancellarla senza rischi: `mestiere cache clean` la rimuove, e il comando successivo scarica di nuovo quello che gli serve. In CI, metti in cache quella cartella tra un'esecuzione e l'altra per risparmiare fetch.
