---
title: Come funziona
seoTitle: "Come funziona mestiere: lock, aggiornamenti e modello di sicurezza"
nav: Come funziona
description: Come mestiere fissa le skill a commit del registry, rileva gli aggiornamenti per skill con gli hash dei tree git e non sovrascrive mai una skill modificata.
order: 7
---

## Come funzionano gli aggiornamenti {#how-updates-work}

mestiere fissa ogni skill a un commit del registry, come fanno i lockfile dei pacchetti. Chi scrive un registry non deve taggare le versioni.

- `add` risolve il `ref` del registry e fissa le nuove skill a quel commit.
- `outdated` scarica il registry ed elenca le skill il cui contenuto è cambiato. Un nuovo commit che non tocca una skill non viene segnalato.
- `update` sposta in avanti il lock, per tutto o solo per i target che indichi. Aggiornare un namespace prende anche le skill aggiunte e toglie quelle cancellate.
- `sync` installa esattamente quello che dice il lock. `sync --frozen` fallisce se `install` e `lock` non coincidono, ed è quello che deve eseguire la CI.

Fissa un intero progetto con `ref: v2026.10` (un tag) o con l'ID di un commit.

## Modello di sicurezza {#safety-model}

Il lock è l'unico stato che mestiere conserva. Non ci sono database nascosti o file marker che possono andare fuori sync. Prima di sostituire o cancellare una cartella, mestiere controlla la cartella stessa:

1. Calcola l'hash del tree git della cartella su disco, saltando i file nascosti come `.DS_Store` e i file di swap degli editor.
2. Cerca la skill del registry che corrisponde al nome della cartella, e ogni versione che quella skill ha avuto nella storia del registry.
3. Solo se l'hash corrisponde a una di quelle versioni la cartella è un'installazione intatta, e solo allora si può sostituire o rimuovere senza rischi.

Per ogni cartella in una directory di skill:

- **Nel lock, l'hash corrisponde alla versione nel lock:** niente da fare.
- **Nel lock, l'hash corrisponde a una versione pubblicata più vecchia:** aggiornata alla versione nel lock.
- **Nel lock, l'hash non corrisponde a nessuna versione pubblicata:** `sync` si ferma. La cartella ha modifiche locali; usa `--force` o `eject`.
- **Non nel lock, nome e contenuto corrispondono a una skill pubblicata:** rimossa, perché è stata tolta da `skills.yaml`.
- **Non nel lock, qualsiasi altro caso:** lasciata com'è. È una tua skill.

`--force` è sempre una scelta esplicita. Lo stesso controllo fa funzionare i flussi di lavoro abituali del team senza passaggi in più:

| Situazione | Risultato |
|---|---|
| Un collega aggiorna una skill; fai pull ed esegui `sync` | La tua copia più vecchia e non modificata viene aggiornata |
| Un collega rimuove una skill; fai pull ed esegui `sync` | La cartella rimasta viene rimossa |
| Hai modificato una skill installata | `sync` si ferma e suggerisce `--force` o `eject` |
| Hai copiato una skill per farne una tua | La copia non viene mai toccata |
| Offline, con un registry in cache | `sync` installa dalla cache |

## Architettura {#architecture}

- **Registry (git):** `skills/<namespace>/<skill>/SKILL.md` e `skills/<namespace>/_shared/`.
- **Cache locale:** un mirror git bare per ogni registry, riempito con `git fetch`.
- **Progetto:** `skills.yaml` (install e lock), `.agents/skills/<namespace>-<skill>/` con i file veri, e symlink da `.claude/skills/` e `.kiro/skills/`.

`add` e `update` risolvono `install` sul mirror del registry e scrivono `lock`. `sync` confronta il lock con le cartelle delle skill su disco e applica solo i passaggi che può dimostrare sicuri: install, upgrade, link, unlink, remove.

## Scelte di progettazione {#design-decisions}

- **Il binario `git` di sistema, non una libreria git in Go,** così l'autenticazione funziona esattamente come per il tuo `git`.
- **Gli hash dei tree git come identità del contenuto.** L'identità di una skill è il tree ID della sua cartella installata, quindi le modifiche si rilevano per singola skill. Per le skill senza link condivisi è il tree ID del registry stesso.
- **La storia come prova di proprietà.** La proprietà non viene mai registrata in un file di stato. Si decide in base al fatto che il contenuto di una cartella sia mai stato pubblicato.
- **Installazioni piatte, registry con namespace.** Gli harness trovano le skill a un solo livello di profondità, quindi i namespace esistono nel registry e nei nomi installati, non come cartelle.
- **Le scritture passano per una cartella temporanea accanto a quella finale,** che poi viene rinominata al suo posto, così un'esecuzione interrotta non lascia mai una skill scritta a metà.
