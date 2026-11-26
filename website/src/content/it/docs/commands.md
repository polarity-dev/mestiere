---
title: Riferimento comandi
seoTitle: "Riferimento comandi di mestiere: ogni comando e flag"
nav: Comandi
description: "Ogni comando di mestiere con argomenti, flag ed esempi: init, registry, add, remove, sync, update, status, doctor, eject, adopt, prune e cache."
order: 6
---

Esegui `mestiere <command> --help` per avere le stesse informazioni nel terminale. Un `mestiere` senza argomenti, in un terminale, mostra un banner con i comandi principali.

## Flag globali {#global-flags}

| Flag | Effetto |
|---|---|
| `-C, --dir <dir>` | Esegue come se partisse da `<dir>` |
| `-q, --quiet` | Stampa solo warning ed errori |
| `--no-input` | Non chiede mai niente: se manca un argomento, fallisce con un suggerimento. Per agenti e script. Vedi [Agenti](/docs/agents/#running-mestiere-from-an-agent). |
| `-v, --version` | Stampa la versione di mestiere |
| `-h, --help` | Aiuto per qualsiasi comando |

`-g, --global` agisce sulle skill installate per tutti i progetti, nella tua home directory. Funziona con `add`, `remove`, `sync`, `update`, `outdated`, `status`, `doctor`, `eject`, `adopt`, `prune` e con `registry add`, `remove`, `list` e `show`. Gli altri comandi lo rifiutano con un messaggio. Vedi [Skill globali](/docs/global-skills/).

## Alias {#aliases}

| Alias | Comando |
|---|---|
| `install` | `sync` |
| `rm`, `uninstall` | `remove` |
| `upgrade` | `update` |
| `ls`, `list` | `status` |

## Target {#targets}

I comandi che accettano skill supportano diverse forme:

| Forma | Significa |
|---|---|
| `frontend` | Il namespace `frontend` |
| `frontend/forms` | La skill `forms` di `frontend` |
| `frontend-forms` | La stessa skill, per nome installato |
| `frontend/frontend-forms` | La stessa skill, per nome della cartella |
| `polarity:frontend` | `frontend` dal registry `polarity` |

Namespace e skill possono contenere trattini, quindi un nome come `ai-tools-review` può avere più di un significato. In quel caso mestiere si ferma ed elenca le letture possibili: scrivi `ns/skill` per una skill, o `ns/` per un namespace.

Un comando, un flag, un namespace, una skill o un nome installato scritti male danno un errore con un suggerimento `did you mean …?`.

## Output e codice di uscita {#output-and-exit-status}

I comandi escono con un codice diverso da zero per qualsiasi riga `error:`. Le righe `warning:` e `hint:` non fanno fallire il comando.

In un terminale, le persone vedono un output formattato: segni di spunta, colori e uno spinner mentre git scarica. Agenti, script e pipe ricevono testo semplice con lo stesso contenuto. `NO_COLOR` disattiva i colori. `status`, `registry show`, `outdated` e `doctor` stampano JSON con `--json`.

## Configurazione del progetto {#project-setup}

### mestiere init {#mestiere-init}

Crea `skills.yaml` e la regola di `.gitignore` nel progetto corrente.

```sh
mestiere init --agents claude,kiro
```

| Flag | Effetto |
|---|---|
| `--agents <list>` | Agenti separati da virgola: `claude`, `kiro`, `cursor`, `codex`, `gemini`, `copilot`, `opencode`. Vedi [Agenti](/docs/agents/#supported-agents). |
| `--link <mode>` | Come gli agenti ricevono le skill: `symlink` (default) o `copy` |

In un terminale senza `--agents`, `init` avvia una procedura guidata:

1. Chiede quali agenti usa il team, con già selezionati quelli trovati nel progetto o nella tua home.
2. Chiede l'URL git di un registry e verifica che git lo raggiunga. Lascialo vuoto per saltare il passaggio.
3. Apre un selettore di skill filtrabile, poi chiede di confermare l'installazione.
4. Stampa i passi successivi.

## Registry {#registries}

### mestiere registry add {#mestiere-registry-add}

Aggiunge un registry al progetto.

```sh
mestiere registry add <name> <url> [--ref <branch|tag|commit>]
mestiere registry add polarity git@github.com:your-org/skills.git --ref main
```

Per default `--ref` è il branch di default del registry.

`registry add` scarica il registry e ne riconosce la struttura. Un repository senza `registry.yaml` nella root e con almeno un `skills/<name>/SKILL.md` è un repository pubblicato per `npx skills`, e `skills.yaml` riceve `layout: skills`. Vedi [Skill dai repository vercel/skills](/docs/registries/#skills-from-vercelskills-repositories).

### mestiere registry remove {#mestiere-registry-remove}

Disinstalla le skill del registry e lo toglie da `skills.yaml`.

```sh
mestiere registry remove <name> [--force]
```

`--force` cancella anche le skill con modifiche locali.

### mestiere registry list {#mestiere-registry-list}

Elenca i registry del progetto.

### mestiere registry show {#mestiere-registry-show}

Elenca i namespace di un registry con la loro descrizione, ciascuno seguito dalle sue skill, rientrate e segnate quando sono installate.

```sh
mestiere registry show [name] [--json]
```

Nel terminale di una persona ogni descrizione è tagliata a 72 caratteri; agenti e pipe ricevono il testo intero.

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

Senza argomenti, `add` apre un selettore filtrabile che elenca i namespace con le loro skill annidate sotto. Senza prompt, un target mancante è un errore. Una skill che il registry non offre dà un errore che lo dice, e suggerisce una skill con lo stesso nome installato quando esiste.

`add` avvisa subito dopo aver installato una skill che esiste anche nella tua home con lo stesso nome, quando un agente caricherebbe quella copia. Vedi [Skill installate in entrambi i posti](/docs/agents/#skills-installed-in-both-places).

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

```sh
mestiere outdated [--json]
```

## Ispezionare {#inspecting}

### mestiere status {#mestiere-status}

Mostra le skill installate, i loro link, le skill locali (comprese quelle non committate) e le skill globali da ogni cartella che un agente legge nella tua home directory. In una cartella senza `skills.yaml` elenca le skill già presenti.

```sh
mestiere status [--json]
```

Una skill presente sia nel progetto sia nella tua home compare sotto "also installed globally", con la copia che carica ogni agente. Con `-g`, ogni skill globale che mestiere non gestisce compare come copia, come link con la sua destinazione o come link rotto.

### mestiere doctor {#mestiere-doctor}

Controlla il progetto senza cambiare nulla: coerenza del lock, modifiche locali, link, `.gitignore`, accesso ai registry e skill nascoste da una copia globale.

```sh
mestiere doctor [--strict] [--offline] [--json]
```

| Flag | Effetto |
|---|---|
| `--strict` | Fallisce anche sulle skill locali non committate |
| `--offline` | Salta i controlli di rete |
| `--json` | Stampa i risultati in JSON; il codice di uscita segnala comunque i problemi |

Con `-g`, `doctor` segnala anche i link rotti, le skill con più copie separate, le skill che mestiere non gestisce e, per ogni agente installato sulla macchina, quante skill globali carica e quanti caratteri occupano i loro nomi e descrizioni. Vedi [Fare pulizia tra le skill globali](/docs/global-skills/#cleaning-up-global-skills).

Errori comuni e come risolverli:

| Messaggio | Soluzione |
|---|---|
| `has local changes` | Tienile con `mestiere eject <name>`, o scartale con `mestiere sync --force` |
| `install and lock disagree` | Esegui `mestiere sync` e fai commit di `skills.yaml` |
| `already exists and isn't a published version` | Una skill locale ha lo stesso nome: rinominala, o sovrascrivila con `--force` |
| `comes from both <a> and <b>` | Aggiungi la skill a `exclude` in uno dei registry |
| `isn't reachable` | Un problema di accesso git: controlla le tue credenziali per quell'URL |
| `is committed but ignored` | `git rm -r --cached <path>`, oppure `mestiere track <name>` per continuare a committarla |
| `<agent> loads ~/… instead of this project's <name>` | Un warning: una copia globale nasconde a quell'agente la versione del progetto. Rimuovila con `mestiere remove -g <name>`, o `mestiere prune -g <name>` se non l'ha installata mestiere, oppure rinominane una. Non fa mai fallire il comando, nemmeno con `--strict`. |
| `has N separate copies` (con `-g`) | `mestiere adopt -g <name>` se un registry la pubblica, altrimenti tieni una sola copia |

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

Tiene una skill installata come tua. I file restano; mestiere smette di gestirli ed esclude la skill dalle installazioni future del suo namespace. I registry con `layout: skills` non hanno namespace, quindi non viene scritto niente in `exclude`.

```sh
mestiere eject <skill>... [--track]
```

`--track` fa anche il commit delle skill espulse. Non si applica con `-g`.

### mestiere adopt {#mestiere-adopt}

Porta sotto la gestione di mestiere una skill installata in un altro modo, ad esempio copiata a mano o con un altro strumento.

```sh
mestiere adopt [-g] <skill>...
mestiere adopt -g shonin
```

`adopt` cerca un registry configurato che pubblichi una skill con lo stesso nome e lo stesso contenuto della cartella su disco, poi la installa da lì. La cartella diventa la copia gestita, e dei link sostituiscono le altre copie. Se nessun registry la pubblica, la skill resta tua.

### mestiere prune {#mestiere-prune}

Sposta nel cestino le skill globali che mestiere non gestisce.

```sh
mestiere prune -g [skill]...
mestiere prune -g ponytail-audit
```

`prune` sposta ogni copia e ogni link delle skill indicate in una cartella datata sotto `~/.local/share/mestiere/trash` (`$XDG_DATA_HOME/mestiere/trash` se è impostata). Non cancella niente: per ripristinare una cartella, rimettila al suo posto. Rifiuta le skill gestite da mestiere; quelle si rimuovono con `mestiere remove -g`. Senza nomi, le persone ricevono un selettore; agenti e script devono indicare le skill.

## Skill globali {#global-skills}

Con `-g`, questi comandi agiscono sulle skill di tutti i progetti sulla tua macchina invece che sul repository corrente: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject`, `adopt`, `prune` e `registry add`, `remove`, `list` e `show`.

```sh
mestiere add -g mestiere
mestiere registry add -g polarity git@github.com:your-org/skills.git
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

Il completamento è dinamico e funziona offline. Propone le skill dei registry in cache per `add`, le skill installate per `remove`, `update`, `eject`, `track` e `untrack`, le skill non gestite per `adopt` e `prune`, i nomi dei registry e i valori di `--agents` e `--link`. Il completamento non usa mai la rete: esegui `mestiere outdated` o `registry show` per aggiornare la cache.

Istruzioni di configurazione: [Installazione](/install/#shell-completions).
