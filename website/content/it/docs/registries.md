---
title: Registry
seoTitle: Crea un registry di Agent Skills con mestiere
nav: Registry
description: Crea un registry git di Agent Skills con mestiere, organizzalo in namespace, condividi file tra le skill e validalo con registry lint.
order: 4
---

Un registry è un repository git di skill da cui i progetti installano. Può essere pubblico o privato e non richiede un server: mestiere lo legge con `git`.

## Crea un registry {#create-a-registry}

```sh
mkdir skills && cd skills && git init
mestiere registry init --name polarity --description "Polarity's team skills"
mestiere registry new-namespace frontend --description "Frontend practices" --owners web-team
mestiere registry new-skill frontend/forms --description "Build accessible forms"
mestiere registry lint
```

Fai commit e push. I progetti lo aggiungono con `mestiere registry add polarity <git-url>`.

## Struttura {#layout}

```
skills/                                   # repository del registry
├── registry.yaml                         # schemaVersion, name, description
└── skills/
    ├── frontend/
    │   ├── namespace.yaml                # description, owners
    │   ├── _shared/                      # file a cui puntano più skill
    │   │   ├── scripts/lint-a11y.sh
    │   │   └── references/design-tokens.md
    │   ├── forms/
    │   │   ├── SKILL.md                  # name: frontend-forms
    │   │   ├── references/
    │   │   │   └── design-tokens.md -> ../../_shared/references/design-tokens.md
    │   │   └── scripts/
    │   │       └── lint-a11y.sh -> ../../_shared/scripts/lint-a11y.sh
    │   └── tables/
    │       ├── SKILL.md
    │       └── scripts -> ../_shared/scripts
    └── backend/
        └── db-migrations/
            ├── SKILL.md
            └── scripts/migrate.sh
```

## Regole per i nomi {#naming-rules}

- I **namespace** usano solo lettere minuscole e cifre, senza trattini: `frontend`, `payments`, `infra2`. Sceglili per dominio o per team.
- Le **skill** usano lettere minuscole, cifre e trattini singoli: `forms`, `db-migrations`.
- Il **nome installato** è `<namespace>-<skill>`, ad esempio `frontend-forms`, al massimo 64 caratteri. Dato che i namespace non hanno trattini, la corrispondenza funziona sempre in entrambe le direzioni.

## Scrivere SKILL.md {#writing-skillmd}

`new-skill` genera un `SKILL.md` con il frontmatter. Poi:

- Mantieni il `name` generato: deve essere uguale al nome installato, `<namespace>-<skill>`.
- Scrivi una `description` da 1 a 1024 caratteri. Decide quando un agente attiva la skill: di' cosa fa la skill e quando usarla, con le parole che un utente userebbe davvero.
- Tieni `SKILL.md` focalizzato. Sposta il materiale lungo in `references/` e collegalo con un percorso relativo.
- Riferisciti ai file inclusi con percorsi relativi alla cartella della skill, ad esempio `scripts/check.sh`.
- Rendi gli script eseguibili (`chmod +x`) e dagli uno shebang.

## Condividere file tra le skill {#sharing-files-between-skills}

Metti un file nella cartella `_shared/` del namespace e collegalo da ogni skill con un symlink relativo, a un singolo file o a un'intera cartella:

```sh
cd skills/frontend/forms
mkdir -p scripts
ln -s ../../_shared/scripts/lint-a11y.sh scripts/lint-a11y.sh   # un file
ln -s ../_shared/references references                           # un'intera cartella
```

Durante l'installazione, mestiere sostituisce ogni link con una copia reale della sua destinazione. Ogni skill installata è autonoma, e il suo `SKILL.md` continua a usare percorsi come `scripts/lint-a11y.sh`. Quando un file condiviso cambia, vengono segnalate come aggiornate solo le skill che lo collegano.

I link possono puntare solo dentro la skill o dentro `_shared/` dello stesso namespace. I file dentro `_shared/` devono essere file reali, non link.

## Cosa controlla lint {#what-lint-checks}

`mestiere registry lint` fallisce su qualsiasi riga `error:`:

- I nomi di namespace, skill e nomi installati rispettano le regole per i nomi.
- `SKILL.md` ha un frontmatter il cui `name` è uguale al nome installato, e una descrizione da 1 a 1024 caratteri.
- I link restano dentro la skill o puntano dentro `_shared/` dello stesso namespace, e i link verso `_shared/` arrivano a un file reale.
- Le skill e `_shared/` non contengono file nascosti. I file nascosti vengono saltati quando le copie installate sono confrontate con il registry, quindi un file nascosto pubblicato non potrebbe mai corrispondere.

I file condivisi che nessuna skill collega producono un warning, che non fa fallire lint.

| Errore | Soluzione |
|---|---|
| `name must be "<ns>-<skill>"` | Imposta il `name` del frontmatter al nome installato |
| `invalid namespace` / `invalid skill name` | Rinomina la cartella secondo le regole per i nomi |
| `hidden files aren't installed` | Cancella il file, o rinominalo senza il punto iniziale (`.env.example` → `env.example`) |
| `links outside the skill and skills/<ns>/_shared/` | Sposta la destinazione in `_shared/` e collegala da lì |
| `which is itself a link` / `links aren't allowed inside _shared` | Sostituisci il link in `_shared/` con il file reale |

## Prova una skill prima di pubblicarla {#test-a-skill-before-publishing}

Da un progetto, aggiungi la tua copia locale come registry e installa da lì. Viene installato solo il contenuto con commit, quindi prima fai commit su un branch:

```sh
mestiere registry add local /path/to/registry --ref my-branch
mestiere add frontend/forms
mestiere registry remove local
```

## Rinominare e rimuovere skill {#renaming-and-removing-skills}

- I progetti che hanno installato l'intero namespace perdono una skill rimossa al loro prossimo `mestiere update`.
- I progetti che hanno installato la skill singolarmente ricevono un errore e devono rimuoverla, quindi annuncia le rimozioni.
- Una rinomina è una rimozione più un'aggiunta.

## Esegui lint in CI {#run-lint-in-ci}

```yaml
- uses: actions/setup-go@v7
  with:
    go-version: stable
- run: go install github.com/polarity-dev/mestiere/cmd/mestiere@latest
- run: mestiere registry lint
```

Altre configurazioni di CI: [CI](/docs/ci/).
