---
title: mestiere
seoTitle: "mestiere: condividi le Agent Skills tra i repository del tuo team"
description: mestiere installa le Agent Skills del tuo team da un registry git in ogni repository, fissate e verificate, per Claude Code, Kiro e altri agenti.
order: 0
---

mestiere è uno strumento da riga di comando per macOS e Linux. Il tuo team tiene le sue [Agent Skills](https://agentskills.io) in un solo repository git, un *registry*, organizzato per namespace. Ogni progetto installa le skill che gli servono, fissate a un commit esatto del registry, e ogni harness di agenti nel repository le vede.

Il *mestiere* è il saper fare che una bottega passa da una persona all'altra. mestiere fa lo stesso per gli agenti di programmazione.

## Installazione {#install}

```sh
brew install --cask polarity-dev/tap/mestiere
```

Homebrew funziona su macOS e Linux. Gli archivi precompilati (amd64, arm64) sono in ogni [release su GitHub](https://github.com/polarity-dev/mestiere/releases), e funziona anche `go install`. Istruzioni complete: [Installa mestiere](/install/).

## Avvio rapido con il tuo agente {#quick-start-with-your-agent}

L'installazione con Homebrew dà anche ai tuoi agenti le skill di mestiere, a livello globale. Se hai installato in un altro modo, esegui una volta `mestiere add -g mestiere`. Poi chiedi al tuo agente, da qualsiasi repository:

```text
Configura mestiere per questo repo con il nostro registry git@github.com:your-org/skills.git
```

L'agente esegue `mestiere init`, ti mostra cosa offre il registry, installa le skill adatte al codice e fa il commit di `skills.yaml`. I prompt successivi funzionano allo stesso modo: "Qualcuna delle nostre skill è obsoleta? Aggiornale e riassumi cosa è cambiato", oppure "Voglio modificare frontend-forms per questo repo, tienila come nostra e fai il commit".

## Avvio rapido {#quick-start}

```sh
mestiere init --agents claude,kiro
mestiere registry add polarity git@github.com:your-org/skills.git
mestiere add frontend backend/db-migrations
git add skills.yaml .gitignore && git commit -m "Add team skills"
```

Un collega che clona il repository esegue `mestiere sync`. Passo per passo: [Primi passi](/docs/).

## Come funziona {#how-it-works}

1. Un registry è un repository git di skill raggruppate in namespace, come `frontend` o `backend`.
2. `mestiere add` copia le skill in `.agents/skills/<namespace>-<skill>/` e blocca ognuna su un commit del registry e un hash del contenuto in `skills.yaml`.
3. Codex, Gemini CLI, Cursor e GitHub Copilot leggono `.agents/skills/` direttamente. Claude Code e Kiro ricevono symlink relativi in `.claude/skills/` e `.kiro/skills/`.
4. `mestiere sync` allinea qualsiasi checkout al lock; `mestiere update` porta le skill all'ultima versione del registry.

## Perché mestiere {#why-mestiere}

- **Una sola fonte di verità.** Le skill vivono in un registry git, pubblico o privato, revisionato come qualsiasi altro codice. Ogni skill può avere un README per le persone, che non entra mai nel contesto dell'agente.
- **Organizzate per namespace.** Sfoglia un registry namespace per namespace, poi installa un intero namespace o una singola skill. Le nuove skill di un namespace installato arrivano al prossimo aggiornamento.
- **Riproducibili.** Ogni installazione è bloccata su un commit del registry e un hash del contenuto. Colleghi e CI ricevono skill identiche byte per byte.
- **Tutti gli agenti insieme.** I file vivono una sola volta in `.agents/skills/`, che Codex, Gemini CLI, Cursor e altri leggono direttamente. Gli agenti con una cartella propria, come Claude Code e Kiro, ricevono symlink verso di essa.
- **Non ti intralcia.** Le skill installate sono ignorate da git per impostazione predefinita. Le tue skill restano accanto a loro intatte, e fare il commit di una skill richiede un solo comando.
- **Sicuro per costruzione.** mestiere sostituisce o elimina una cartella solo quando può dimostrare che è una copia non modificata di una skill pubblicata. Le skill modificate non vengono mai sovrascritte in silenzio.
- **Nessun lock-in.** I registry sono normali repository git. Le skill installate sono normali cartelle che seguono la specifica Agent Skills.

## Domande frequenti {#frequently-asked-questions}

### Cos'è mestiere?

mestiere è uno strumento da riga di comando che installa Agent Skills da repository git nei tuoi progetti. Funziona come un package manager per skill: dichiari cosa vuoi in `skills.yaml`, mestiere lo risolve in commit esatti del registry, e chiunque esegua `mestiere sync` riceve gli stessi file.

### Cosa sono le Agent Skills?

Una Agent Skill è una cartella con un file `SKILL.md`, più gli script, i riferimenti e gli asset che le servono. Agenti di programmazione come Claude Code, Kiro, Codex e Cursor leggono la descrizione della skill e la caricano quando un'attività corrisponde. Il formato è descritto su [agentskills.io](https://agentskills.io).

### Quali agenti di programmazione supporta mestiere?

Tutti gli agenti che leggono le Agent Skills: Claude Code, Kiro, Cursor, Codex, Gemini CLI, GitHub Copilot, OpenCode e altri. I file installati vivono in `.agents/skills/`, la cartella condivisa che la maggior parte di loro legge direttamente. Claude Code (`.claude/skills/`) e Kiro (`.kiro/skills/`) ricevono link verso di essa. Vedi [Agenti](/docs/agents/).

### Posso usare un repository di skill privato?

Sì. mestiere usa il `git` di sistema, quindi un registry funziona ovunque funzioni `git clone`: chiavi SSH, credential helper e `gh auth` inclusi. In CI, dai al job credenziali che possano leggere il registry.

### Come condivido le skill con tutto il team?

Fai il commit di `skills.yaml` e `.gitignore`. Colleghi e CI eseguono `mestiere sync`, che installa esattamente le versioni nel lock. In CI, `mestiere sync --frozen` fallisce se il file non è coerente. Vedi [CI](/docs/ci/).

### Devo fare il commit delle skill installate?

No. Le skill installate sono ignorate da git per impostazione predefinita, perché `skills.yaml` basta per riprodurle. Per fare comunque il commit di una skill, esegui `mestiere track <skill>`.

### mestiere sovrascriverà una skill che ho modificato?

No. Prima di sostituire o eliminare una cartella, mestiere controlla che il suo contenuto corrisponda a una versione pubblicata nel registry. Le skill modificate fermano `sync`; tieni le tue modifiche con `mestiere eject <skill>`. Vedi il [modello di sicurezza](/docs/how-it-works/#safety-model).

### Il mio agente può configurare mestiere per me?

Sì. mestiere include le sue skill, e l'installazione con Homebrew le dà ai tuoi agenti per ogni progetto. Chiedi "Configura mestiere per questo repo con il nostro registry <url>" e l'agente esegue `init`, propone le skill e fa il commit di `skills.yaml`. Nella shell di un agente mestiere non resta mai in attesa di una risposta, e `--json` gli dà un output da elaborare. Vedi [Agenti](/docs/agents/#running-mestiere-from-an-agent).

### Posso installare skill pubblicate per npx skills?

Sì. Aggiungi il repository come qualsiasi registry, ad esempio `mestiere registry add next https://github.com/vercel/next.js`, poi `mestiere add -r next next-dev-loop`. mestiere riconosce la struttura e blocca le skill come tutte le altre. Vedi [Skill dai repository vercel/skills](/docs/registries/#skills-from-vercelskills-repositories).

### Posso installare skill per tutti i progetti sulla mia macchina?

Sì, con `-g`: `mestiere add -g -r polarity frontend` installa in `~/.agents/skills/`, con i link per i tuoi agenti, e il manifest in `~/.config/mestiere/skills.yaml`. Preferisci le installazioni di progetto per le skill su cui il team fa affidamento, perché solo uno `skills.yaml` nel commit dà a colleghi e CI le stesse versioni. Vedi [Skill globali](/docs/global-skills/).

### Come aggiorno le skill?

`mestiere outdated` elenca le skill il cui contenuto è cambiato nel registry. `mestiere update` le porta tutte all'ultima versione, oppure solo quelle che indichi. Vedi [Come funzionano gli aggiornamenti](/docs/how-it-works/#how-updates-work).

### Come so che è uscita una nuova versione di mestiere?

mestiere controlla una volta al giorno se c'è una release più recente e te lo dice quando esiste. Aggiorna con `brew upgrade --cask mestiere`, che aggiorna anche le skill di mestiere. Imposta `MESTIERE_NO_UPDATE_CHECK=1` per disattivare l'avviso. Vedi [Aggiornamento](/install/#upgrade).

### mestiere funziona su Windows?

Non ancora. Sono supportati macOS e Linux su amd64 e arm64; il supporto per Windows è nella roadmap.
