---
title: Riferimento skills.yaml
seoTitle: Riferimento skills.yaml per i progetti mestiere
nav: skills.yaml
description: Riferimento per skills.yaml, il file di progetto di mestiere che elenca agenti, registry, namespace e skill installati, esclusioni e il lock.
order: 3
---

`skills.yaml` sta nella root del progetto. Un solo file contiene sia quello che hai chiesto sia quello che è stato risolto. Le [skill globali](/docs/global-skills/) usano lo stesso formato in `~/.config/mestiere/skills.yaml`.

## Esempio {#example}

```yaml
version: 1
agents:
  - claude
  - kiro
registries:
  polarity:
    url: git@github.com:polarity-dev/skills.git
    ref: main                     # branch, tag o commit; di default il branch di default del registry
    install:
      - frontend                  # intero namespace
      - backend/db-migrations     # singola skill
    exclude:
      - frontend/legacy-forms     # salta una skill di un namespace installato
    lock: # scritto da mestiere
      frontend-forms:
        path: skills/frontend/forms
        commit: 3f2a9c1e...
        tree: 9be1d07a...
```

## Chiavi {#keys}

| Chiave | Modificata da | Significato |
|---|---|---|
| `version` | mestiere | Versione del formato del file. Sempre `1`. |
| `agents` | te | Harness che ricevono i link: `claude`, `kiro`. `.agents/skills/` è sempre usata. |
| `registries.<name>.url` | te | URL git del registry: SSH, HTTPS o un percorso locale. |
| `registries.<name>.ref` | te | Branch, tag o commit da seguire. Di default è il branch di default del registry. |
| `registries.<name>.install` | te | Namespace (`frontend`) e skill (`backend/db-migrations`) da installare. |
| `registries.<name>.exclude` | te | Skill di un namespace installato da saltare. |
| `registries.<name>.lock` | mestiere | Skill risolte: percorso nel registry, commit e hash del contenuto. Non modificarlo mai. |
| `link` | te | Opzionale. `copy` dà alle cartelle degli agenti delle copie invece dei symlink. |
| `strict` | te | Opzionale. `true` fa fallire `doctor` sulle skill locali senza commit. |

Quando riscrive il file, mestiere conserva i tuoi commenti e l'ordine delle chiavi. Dopo aver modificato a mano `install`, `exclude` o `agents`, esegui `mestiere sync`.

## Il lock {#the-lock}

Ogni voce sotto `lock` ha come chiave il nome installato, `<namespace>-<skill>`:

- `path`: dove si trova la skill nel registry.
- `commit`: il commit del registry a cui è stata risolta.
- `tree`: l'ID del tree git della cartella installata. Cambia solo quando cambia il contenuto di quella skill, ed è così che `outdated` ignora i commit che non toccano una skill.

`add` e `update` scrivono il lock. `sync` installa esattamente quello che indica.

## Fissare la versione di un progetto {#pinning-a-project}

Segui un tag o un commit invece di un branch:

```yaml
registries:
  polarity:
    url: git@github.com:polarity-dev/skills.git
    ref: v2026.10
```

Chi gestisce un registry non deve per forza creare tag di versione: ogni installazione è già bloccata su un commit.

## Struttura installata {#installed-layout}

```
.agents/skills/frontend-forms/                  # file reali
.claude/skills/frontend-forms -> ../../.agents/skills/frontend-forms
.kiro/skills/frontend-forms   -> ../../.agents/skills/frontend-forms
```

I link sono relativi, quindi continuano a funzionare quando sposti il repository. Con `link: copy`, le cartelle degli agenti ricevono invece delle copie.

## .gitignore {#gitignore}

`mestiere init` aggiunge una regola, e solo se le regole che hai già non coprono le cartelle delle skill:

```gitignore
**/.*/skills/*
```

Ignora ogni cartella di skill dentro qualsiasi directory nascosta, pacchetti annidati compresi, ma non tocca file come `.claude/settings.json`. Basta una riga di eccezione per fare il commit di un'intera skill:

```gitignore
!.claude/skills/my-skill
```

`mestiere track <skill>` e `mestiere untrack <skill>` aggiungono e rimuovono queste righe per ogni cartella in cui compare la skill. Scriverle a mano funziona allo stesso modo.
