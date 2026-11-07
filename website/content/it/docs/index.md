---
title: Primi passi
seoTitle: "Primi passi con mestiere: installa le Agent Skills"
nav: Primi passi
description: Configura mestiere in un repository, aggiungi un registry di skill, installa namespace o singole skill e condividile con il team e con la CI.
order: 2
---

Questa guida installa le skill di un registry in un progetto e le condivide con il tuo team. Presuppone che mestiere sia [installato](/install/) e che tu possa fare `git clone` del registry.

> Preferisci farlo fare al tuo agente? Con le skill di mestiere installate (l'installazione con Homebrew le aggiunge), chiedi "Configura mestiere per questo repo con il nostro registry <git-url>". L'agente esegue gli stessi passi descritti sotto e ti chiede quali skill installare. Vedi [Agenti](/docs/agents/#teach-your-agent-to-use-mestiere).

## 1. Inizializza il progetto {#1-initialize-the-project}

Nella root del tuo repository:

```sh
mestiere init --agents claude,kiro
```

Questo comando crea `skills.yaml` e aggiunge una regola a `.gitignore`, così le skill installate restano fuori da git. `--agents` sceglie gli harness che ricevono le skill: `claude`, `kiro` o entrambi. Senza il flag, `init` te lo chiede.

## 2. Aggiungi un registry {#2-add-a-registry}

Un registry è un repository git di skill. Dagli un nome breve e indica il suo URL git:

```sh
mestiere registry add polarity git@github.com:polarity-dev/skills.git
```

mestiere segue il branch di default del registry. Usa `--ref` per un branch, un tag o un commit, ad esempio `--ref v2026.10`. Se hai dubbi, verifica prima l'accesso con `git ls-remote <url>`.

## 3. Guarda cosa offre {#3-see-what-it-offers}

```sh
mestiere registry show polarity
```

Questo comando elenca ogni namespace e ogni skill con la sua descrizione, e segna quello che è già installato.

## 4. Installa le skill {#4-install-skills}

```sh
mestiere add frontend                 # un intero namespace
mestiere add backend/db-migrations    # una singola skill
```

- L'installazione di un **namespace** segue il namespace: le skill aggiunte in seguito arrivano con il prossimo `mestiere update`.
- L'installazione di una **skill** porta solo quella skill.
- Con più registry configurati, scegline uno con `-r <registry>`.
- Senza argomenti, `add` apre un selettore interattivo.

Ogni skill finisce in `.agents/skills/<namespace>-<skill>/`, ad esempio `.agents/skills/frontend-forms/`, e ogni agente riceve un link relativo a quella cartella:

```
.agents/skills/frontend-forms/                  # file reali
.claude/skills/frontend-forms -> ../../.agents/skills/frontend-forms
.kiro/skills/frontend-forms   -> ../../.agents/skills/frontend-forms
```

## 5. Fai commit e condividi {#5-commit-and-share}

```sh
git add skills.yaml .gitignore
git commit -m "Add team skills"
```

Ora `skills.yaml` contiene sia quello che hai chiesto sia il commit esatto del registry per ogni skill. Un collega che clona il repository esegue:

```sh
mestiere sync
```

Esegui di nuovo `mestiere sync` dopo un pull o un cambio di branch. In CI usa `mestiere sync --frozen`: vedi [CI](/docs/ci/).

## Tieni aggiornate le skill {#keep-skills-up-to-date}

```sh
mestiere outdated              # skill il cui contenuto è cambiato nel registry
mestiere update                # porta tutto all'ultima versione
mestiere update frontend       # oppure solo alcuni target
```

Poi fai commit di `skills.yaml`. Un nuovo commit del registry che non tocca una skill non viene segnalato. Dettagli: [Come funzionano gli aggiornamenti](/docs/how-it-works/#how-updates-work).

## Rimuovi le skill {#remove-skills}

```sh
mestiere remove backend/db-migrations
```

Se rimuovi una skill di un namespace installato, mestiere la aggiunge a `exclude`, così il resto del namespace resta. Le skill con modifiche locali vengono mantenute e segnalate.

## Le tue skill {#your-own-skills}

mestiere gestisce solo le skill elencate nel lock. Tutto il resto in una cartella di skill è tuo: non viene mai modificato, spostato o cancellato.

- **Scrivi una skill** direttamente in `.agents/skills/`, `.claude/skills/` o `.kiro/skills/`, poi esegui `mestiere track <name>` per farne il commit.
- **Personalizza una skill del registry** con `mestiere eject <name>`. I file restano dove sono e mestiere smette di aggiornarli. Aggiungi `--track` per farne il commit.
- **Trova le skill locali senza commit** con `mestiere status`. Di default sono ignorate, quindi così scopri una skill che i tuoi colleghi non riceverebbero mai.

## Skill per ogni progetto {#skills-for-every-project}

Aggiungi `-g` per installare le skill per tutta la tua macchina invece che per un solo repository, ad esempio `mestiere add -g -r polarity frontend`. Per le skill su cui un team fa affidamento, preferisci l'installazione nel progetto. Vedi [Skill globali](/docs/global-skills/).

## Controlla il progetto {#check-the-project}

```sh
mestiere status     # skill installate, i loro link, skill locali e globali
mestiere doctor     # controlla tutto, senza cambiare niente
```

`status` funziona anche in una cartella senza `skills.yaml`: elenca le skill già presenti e quelle globali.

`doctor` esce con un errore in caso di drift, link rotti o file incoerenti, e ti dice come risolvere ogni problema.

## Prossimi passi {#next-steps}

- [Riferimento skills.yaml](/docs/skills-yaml/): ogni impostazione, il lock e `.gitignore`.
- [Riferimento comandi](/docs/commands/): ogni comando e ogni flag.
- [Registry](/docs/registries/): pubblica le tue skill.
- [Skill globali](/docs/global-skills/): skill per ogni progetto sulla tua macchina.
