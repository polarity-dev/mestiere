---
title: Registry
seoTitle: Crea un registry di Agent Skills con mestiere
nav: Registry
description: Crea un registry git di Agent Skills con mestiere, organizzalo in namespace, condividi file tra le skill e validalo con registry lint.
order: 4
---

Un registry è un repository git di skill da cui i progetti installano. Può essere pubblico o privato e non richiede un server: mestiere lo legge con `git`. mestiere installa anche le skill dai repository pubblicati per `npx skills`: vedi [Skill dai repository vercel/skills](#skills-from-vercelskills-repositories).

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
    │   ├── frontend-forms/               # cartella = nome installato
    │   │   ├── SKILL.md                  # name: frontend-forms
    │   │   ├── README.md                 # per chi sfoglia il registry, non installato
    │   │   ├── references/
    │   │   │   └── design-tokens.md -> ../../_shared/references/design-tokens.md
    │   │   └── scripts/
    │   │       └── lint-a11y.sh -> ../../_shared/scripts/lint-a11y.sh
    │   └── frontend-tables/
    │       ├── SKILL.md
    │       └── scripts -> ../_shared/scripts
    └── backend/
        └── backend-db-migrations/
            ├── SKILL.md
            └── scripts/migrate.sh
```

## Regole per i nomi {#naming-rules}

- I **namespace** e le **skill** usano lettere minuscole, cifre e trattini singoli: `frontend`, `ai-tools`, `forms`, `db-migrations`. Scegli i namespace per dominio o per team.
- Il **nome installato** è `<namespace>-<skill>`, ad esempio `frontend-forms`, al massimo 64 caratteri.
- La cartella di una skill è il suo nome installato: `skills/frontend/frontend-forms/`, come richiede la specifica Agent Skills.
- Due skill non possono avere lo stesso nome installato: `ai/code-review` e `ai-code/review` si installano entrambe come `ai-code-review`, quindi lint rifiuta la coppia.

## Scrivere SKILL.md {#writing-skillmd}

`new-skill` genera un `SKILL.md` con il frontmatter. Poi:

- Mantieni il `name` generato: deve essere uguale al nome installato, `<namespace>-<skill>`.
- Scrivi una `description` da 1 a 1024 caratteri. Decide quando un agente attiva la skill: di' cosa fa la skill e quando usarla, con le parole che un utente userebbe davvero.
- Tieni `SKILL.md` focalizzato. Sposta il materiale lungo in `references/` e collegalo con un percorso relativo.
- Riferisciti ai file inclusi con percorsi relativi alla cartella della skill, ad esempio `scripts/check.sh`.
- Rendi gli script eseguibili (`chmod +x`) e dagli uno shebang.

## README della skill {#skill-readme}

Una cartella di skill può contenere un `README.md` accanto a `SKILL.md`, in qualsiasi combinazione di maiuscole e minuscole (anche `readme.md` va bene). Usalo per presentare la skill alle persone: a cosa serve, casi d'uso, esempi. Gli host git lo mostrano quando qualcuno apre la cartella. mestiere non lo installa mai, quindi non entra nel contesto dell'agente. Non collegarlo da `SKILL.md`.

Viene saltato solo il file al primo livello; un `README.md` dentro una sottocartella si installa come qualsiasi altro file.

## Condividere file tra le skill {#sharing-files-between-skills}

Metti un file nella cartella `_shared/` del namespace e collegalo da ogni skill con un symlink relativo, a un singolo file o a un'intera cartella:

```sh
cd skills/frontend/frontend-forms
mkdir -p scripts
ln -s ../../_shared/scripts/lint-a11y.sh scripts/lint-a11y.sh   # un file
ln -s ../_shared/references references                           # un'intera cartella
```

Durante l'installazione, mestiere sostituisce ogni link con una copia reale della sua destinazione. Ogni skill installata è autonoma, e il suo `SKILL.md` continua a usare percorsi come `scripts/lint-a11y.sh`. Quando un file condiviso cambia, vengono segnalate come aggiornate solo le skill che lo collegano.

I link possono puntare solo dentro la skill o dentro `_shared/` dello stesso namespace. I file dentro `_shared/` devono essere file reali, non link.

## Cosa controlla lint {#what-lint-checks}

`mestiere registry lint` fallisce su qualsiasi riga `error:`:

- I nomi di namespace, skill e nomi installati rispettano le regole per i nomi, e nessuna coppia di skill ha lo stesso nome installato.
- Ogni skill sta in `skills/<namespace>/<namespace>-<skill>/`.
- `SKILL.md` ha un frontmatter il cui `name` è uguale al nome installato, e una descrizione da 1 a 1024 caratteri.
- I link restano dentro la skill o puntano dentro `_shared/` dello stesso namespace, e i link verso `_shared/` arrivano a un file reale.
- Le skill e `_shared/` non contengono file nascosti. I file nascosti vengono saltati quando le copie installate sono confrontate con il registry, quindi un file nascosto pubblicato non potrebbe mai corrispondere.

I file condivisi che nessuna skill collega producono un warning, che non fa fallire lint.

| Errore | Soluzione |
|---|---|
| `name must be "<ns>-<skill>"` | Imposta il `name` del frontmatter al nome installato |
| `skill folders are named <ns>-<skill>` | Rinomina la cartella con il nome installato (`git mv skills/<ns>/<skill> skills/<ns>/<ns>-<skill>`) |
| `invalid namespace` / `invalid skill name` | Rinomina la cartella secondo le regole per i nomi |
| `installs as <name>, like <folder>` | Due skill hanno lo stesso nome installato; rinominane una |
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

## Skill dai repository vercel/skills {#skills-from-vercelskills-repositories}

Alcuni progetti pubblicano le skill per `npx skills` invece che come registry di mestiere, come Next.js in `vercel/next.js`. mestiere installa anche queste:

```sh
mestiere registry add next https://github.com/vercel/next.js
mestiere registry show next
mestiere add -r next next-dev-loop
```

`registry add` esamina il repository. Se non c'è un `registry.yaml` nella root e c'è almeno un `skills/<name>/SKILL.md`, scrive `layout: skills` in `skills.yaml`:

```yaml
registries:
  next:
    url: https://github.com/vercel/next.js
    layout: skills
    install:
      - next-dev-loop
```

- Le skill si installano con il loro nome, lo stesso nome di cartella che creerebbe `npx skills`, quindi `next-dev-loop` finisce in `.agents/skills/next-dev-loop/`.
- Conta solo `skills/<name>/SKILL.md`. mestiere non cerca in altre cartelle e non legge `skills.sh.json` né i manifest dei plugin.
- `install` elenca nomi di skill. Non ci sono namespace, quindi non c'è `exclude`.
- Il lock, `sync`, `update`, `outdated`, `doctor` e il [modello di sicurezza](/docs/how-it-works/#safety-model) funzionano come per qualsiasi registry.
- Il mirror segue solo `ref` (il branch di default se non è impostato) e scarica il contenuto dei file solo per le skill che installi. `vercel/next.js` occupa circa 130 MB in cache invece di diversi gigabyte. `ref` deve essere un branch o un tag.

Usa un URL git completo, come per qualsiasi registry. I repository privati funzionano con le tue credenziali git.
