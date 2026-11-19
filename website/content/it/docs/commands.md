---
title: Riferimento comandi
seoTitle: "Riferimento comandi di mestiere: ogni comando e flag"
nav: Comandi
description: Ogni comando di mestiere con argomenti, flag ed esempi: init, registry, add, remove, sync, update, outdated, status, track, eject, doctor e cache.
order: 6
---

Esegui `mestiere <command> --help` per avere le stesse informazioni nel terminale.

## Flag globali {#global-flags}

| Flag | Effetto |
|---|---|
| `-C, --dir <dir>` | Esegue come se partisse da `<dir>` |
| `-q, --quiet` | Stampa solo warning ed errori |
| `-g, --global` | Agisce sulle skill installate per tutti i progetti, nella tua home directory. Vedi [Skill globali](/docs/global-skills/). |
| `--version` | Stampa la versione di mestiere |
| `-h, --help` | Aiuto per qualsiasi comando |

## Target {#targets}

I comandi che accettano skill supportano diverse forme:

| Forma | Significa |
|---|---|
| `frontend` | Il namespace `frontend` |
| `frontend/forms` | La skill `forms` di `frontend` |
| `frontend-forms` | La stessa skill, per nome installato |
| `polarity:frontend` | `frontend` dal registry `polarity` |

## Codice di uscita {#exit-status}

I comandi escono con un codice diverso da zero per qualsiasi riga `error:`. Le righe `warning:` e `hint:` non fanno fallire il comando.

## Configurazione del progetto {#project-setup}

### mestiere init {#mestiere-init}

Crea `skills.yaml` e la regola di `.gitignore` nel progetto corrente.

```sh
mestiere init --agents claude,kiro
```

| Flag | Effetto |
|---|---|
| `--agents <list>` | Agenti separati da virgola: `claude`, `kiro`. Se lo ometti in un terminale, te lo chiede. |
| `--link <mode>` | Come gli agenti ricevono le skill: `symlink` (default) o `copy` |

## Registry {#registries}

### mestiere registry add {#mestiere-registry-add}

Aggiunge un registry al progetto.

```sh
mestiere registry add <name> <url> [--ref <branch|tag|commit>]
mestiere registry add polarity git@github.com:polarity-dev/skills.git --ref main
```

Per default `--ref` è il branch di default del registry.

### mestiere registry remove {#mestiere-registry-remove}

Disinstalla le skill del registry e lo toglie da `skills.yaml`.

```sh
mestiere registry remove <name> [--force]
```

`--force` cancella anche le skill con modifiche locali.

### mestiere registry list {#mestiere-registry-list}

Elenca i registry del progetto.

### mestiere registry show {#mestiere-registry-show}

Elenca i namespace e le skill offerti da un registry, con descrizioni e stato di installazione.

```sh
mestiere registry show [name]
```

## Installare e rimuovere {#installing-and-removing}

### mestiere add {#mestiere-add}

Installa namespace o skill.

```sh
mestiere add <namespace | namespace/skill>... [-r <registry>] [--force]
mestiere add frontend
mestiere add backend/db-migrations -r polarity
```

| Flag | Effetto |
|---|---|
| `-r, --registry <name>` | Registry da cui installare; obbligatorio quando ne sono configurati diversi |
| `--force` | Sovrascrive le cartelle esistenti con lo stesso nome |

Senza argomenti, `add` apre un selettore interattivo.

### mestiere remove {#mestiere-remove}

Disinstalla namespace o skill.

```sh
mestiere remove <namespace | namespace/skill>... [--force]
```

Rimuovere una sola skill di un namespace installato la aggiunge invece a `exclude`. Le skill modificate vengono tenute e segnalate come skill locali, a meno che tu non passi `--force`.

## Sincronizzare e aggiornare {#syncing-and-updating}

### mestiere sync {#mestiere-sync}

Allinea le cartelle delle skill al lock in `skills.yaml`. Eseguilo dopo un clone, un pull, un cambio di branch o una modifica a mano.

```sh
mestiere sync [--frozen] [--force] [--dry-run]
```

| Flag | Effetto |
|---|---|
| `--frozen` | Fallisce se `install` e `lock` non coincidono, invece di risolvere di nuovo. Usalo in CI. |
| `--force` | Sovrascrive le modifiche locali |
| `--dry-run` | Stampa cosa cambierebbe |

### mestiere update {#mestiere-update}

Porta le skill nel lock all'ultima versione del loro ref. Senza target, aggiorna tutto. Aggiornare un namespace prende anche le skill aggiunte e toglie quelle cancellate.

```sh
mestiere update [target]... [--force]
mestiere update frontend polarity:backend
```

### mestiere outdated {#mestiere-outdated}

Mostra le skill il cui contenuto è cambiato nel loro registry. Un commit che non tocca una skill non viene segnalato.

## Ispezionare {#inspecting}

### mestiere status {#mestiere-status}

Mostra le skill installate, i loro link, le skill locali (comprese quelle non committate) e le skill globali da `~/.agents/skills`, `~/.claude/skills` e `~/.kiro/skills`. In una cartella senza `skills.yaml` elenca le skill già presenti.

### mestiere doctor {#mestiere-doctor}

Controlla il progetto senza cambiare nulla: coerenza del lock, modifiche locali, link, `.gitignore` e accesso ai registry.

```sh
mestiere doctor [--strict] [--offline]
```

| Flag | Effetto |
|---|---|
| `--strict` | Fallisce anche sulle skill locali non committate |
| `--offline` | Salta i controlli di rete |

Errori comuni e come risolverli:

| Messaggio | Soluzione |
|---|---|
| `has local changes` | Tienile con `mestiere eject <name>`, o scartale con `mestiere sync --force` |
| `install and lock disagree` | Esegui `mestiere sync` e fai commit di `skills.yaml` |
| `already exists and isn't a published version` | Una skill locale ha lo stesso nome: rinominala, o sovrascrivila con `--force` |
| `comes from both <a> and <b>` | Aggiungi la skill a `exclude` in uno dei registry |
| `isn't reachable` | Un problema di accesso git: controlla le tue credenziali per quell'URL |
| `is committed but ignored` | `git rm -r --cached <path>`, oppure `mestiere track <name>` per continuare a committarla |

## Le tue skill {#your-own-skills}

### mestiere track {#mestiere-track}

Fa il commit di una skill aggiungendo eccezioni a `.gitignore` per ogni cartella in cui compare.

```sh
mestiere track <skill>...
```

### mestiere untrack {#mestiere-untrack}

Rimuove le eccezioni a `.gitignore` di una skill.

```sh
mestiere untrack <skill>...
```

### mestiere eject {#mestiere-eject}

Tiene una skill installata come tua. I file restano; mestiere smette di gestirli ed esclude la skill dalle installazioni future del suo namespace.

```sh
mestiere eject <skill>... [--track]
```

`--track` fa anche il commit delle skill espulse. Non si applica con `-g`.

## Skill globali {#global-skills}

Con `-g`, questi comandi agiscono sulle skill di tutti i progetti sulla tua macchina invece che sul repository corrente: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject` e i sottocomandi di `registry`.

```sh
mestiere add -g mestiere
mestiere registry add -g polarity git@github.com:polarity-dev/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

`init`, `track` e `untrack` non si applicano: niente di globale sta in git. Dove vanno i file e il manifest: [Skill globali](/docs/global-skills/).

## Cache {#cache}

### mestiere cache {#mestiere-cache}

Ispeziona o svuota la cache dei registry: un mirror git bare per ogni registry.

```sh
mestiere cache dir      # stampa la directory della cache
mestiere cache info     # mostra la directory e la sua dimensione
mestiere cache clean    # cancella tutti i mirror; vengono scaricati di nuovo quando servono
```

La posizione è configurabile: vedi [Configurazione](/docs/ci/#configuration).

## Scrivere un registry {#registry-authoring}

### mestiere registry init {#mestiere-registry-init}

Crea la struttura di un nuovo repository di registry.

```sh
mestiere registry init [dir] [--name <name>] [--description <text>]
```

Per default `--name` è il nome della directory.

### mestiere registry new-namespace {#mestiere-registry-new-namespace}

```sh
mestiere registry new-namespace <namespace> [--description <text>] [--owners <list>]
```

### mestiere registry new-skill {#mestiere-registry-new-skill}

Crea una skill con un template di `SKILL.md`.

```sh
mestiere registry new-skill <namespace>/<skill> [--description <text>]
```

### mestiere registry lint {#mestiere-registry-lint}

Controlla un registry rispetto alle regole sui nomi e su `SKILL.md`. Eseguilo nella CI del registry.

```sh
mestiere registry lint [dir]
```

Vedi [cosa controlla lint](/docs/registries/#what-lint-checks).

## Completamento della shell {#shell-completions}

```sh
mestiere completion bash|zsh|fish
```

Istruzioni di configurazione: [Installazione](/install/#shell-completions).
