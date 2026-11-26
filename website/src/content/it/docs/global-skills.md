---
title: Skill globali
seoTitle: Installa Agent Skills per ogni progetto con mestiere -g
nav: Skill globali
description: "Installa Agent Skills per tutti i tuoi progetti con mestiere -g: dove vanno i file, quali comandi valgono, come fare pulizia e quando preferire un progetto."
order: 5.5
---

Le skill globali si installano una volta per tutta la tua macchina invece che per un solo repository. Gli agenti le caricano in ogni progetto. Aggiungi `-g` a un comando per agire su di esse:

```sh
mestiere add -g mestiere                  # le skill di mestiere; l'installazione con Homebrew esegue questo comando
mestiere registry add -g polarity git@github.com:your-org/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

## Dove vanno le cose {#where-things-go}

| Cosa | Dove |
|---|---|
| Manifest | `~/.config/mestiere/skills.yaml`, oppure `$XDG_CONFIG_HOME/mestiere/skills.yaml` |
| File delle skill | `~/.agents/skills/<namespace>-<skill>/` |
| Link | `~/.claude/skills/`, `~/.kiro/skills/`, `~/.gemini/skills/`, `~/.copilot/skills/`, `~/.config/opencode/skills/`, per gli agenti installati sulla tua macchina |
| Cestino | `~/.local/share/mestiere/trash/`, oppure `$XDG_DATA_HOME/mestiere/trash/`, per le skill spostate da `prune` |

Il manifest ha lo stesso formato dello [skills.yaml](/docs/skills-yaml/) di un progetto, con `install`, `exclude` e il lock. Un nuovo manifest globale ha già configurato il registry `mestiere`, quindi `mestiere add -g mestiere` funziona senza `registry add`.

Un nuovo manifest globale collega gli agenti che hanno una cartella di configurazione nella tua home directory, come `~/.claude`, e Claude Code e Kiro se non ne trova nessuna. Non crea cartelle per gli agenti che non usi. Cursor e Codex leggono `~/.agents/skills/` e non hanno bisogno di link. La tabella completa è in [Agenti](/docs/agents/#supported-agents).

## Comandi {#commands}

Questi comandi accettano `-g`: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject`, `adopt`, `prune` e `registry add`, `remove`, `list` e `show`. Funzionano esattamente come in un progetto, con lo stesso [modello di sicurezza](/docs/how-it-works/#safety-model): una skill globale che hai modificato non viene mai sovrascritta.

`init`, `track`, `untrack` e `eject --track` non si applicano, perché niente di globale è in git. Gli altri comandi rifiutano `-g` con un messaggio.

## Le skill di mestiere {#mestieres-own-skills}

`mestiere add -g mestiere` installa `mestiere-project` e `mestiere-registry`, che insegnano al tuo agente a configurare e usare mestiere in qualsiasi repository. L'installazione con Homebrew lo fa per te e le tiene aggiornate con `brew upgrade`. Vedi [Agenti](/docs/agents/#teach-your-agent-to-use-mestiere).

Per rimuoverle, ad esempio prima di disinstallare:

```sh
mestiere remove -g mestiere
```

## Fare pulizia tra le skill globali {#cleaning-up-global-skills}

Le skill globali si caricano in ogni sessione di ogni progetto, quindi quelle dimenticate occupano contesto ovunque. `mestiere doctor -g` segnala:

- le skill che mestiere non gestisce;
- i link rotti, e le skill con più copie separate che possono divergere;
- per ogni agente installato sulla macchina, quante skill globali carica e quanti caratteri occupano i loro nomi e descrizioni.

`mestiere status -g` mostra ogni skill non gestita come copia, come link con la sua destinazione o come link rotto. Poi intervieni su ognuna:

```sh
mestiere adopt -g shonin          # la gestisce, se un registry configurato pubblica lo stesso contenuto
mestiere prune -g ponytail-audit  # sposta nel cestino ogni sua copia e ogni suo link
```

`adopt` funziona quando un registry configurato pubblica una skill con lo stesso nome e lo stesso contenuto, ad esempio una che hai copiato a mano o installato con un altro strumento. La cartella diventa la copia gestita, e dei link sostituiscono le altre copie.

`prune` sposta le skill in una cartella datata sotto `~/.local/share/mestiere/trash` (`$XDG_DATA_HOME/mestiere/trash` se è impostata). Non cancella niente: per ripristinare una cartella, rimettila al suo posto. Rifiuta le skill gestite da mestiere; quelle si rimuovono con `mestiere remove -g`. Senza nomi, le persone ricevono un selettore; agenti e script devono indicare le skill.

## Vedere le skill globali da un progetto {#seeing-global-skills-from-a-project}

`mestiere status` in un progetto elenca anche le skill in ogni cartella che un agente legge nella tua home directory, globali o no, perché anche da lì le caricano gli agenti. Funziona anche nelle cartelle senza `skills.yaml`.

Una skill presente sia nel progetto sia nella tua home compare sotto "also installed globally", con la copia che carica ogni agente: Claude Code carica quella globale, Kiro quella del progetto. `doctor` avvisa quando un agente carica la copia globale. Vedi [Skill installate in entrambi i posti](/docs/agents/#skills-installed-in-both-places).

## Globali o di progetto? {#global-or-project}

Per le skill su cui un team fa affidamento, preferisci l'installazione nel progetto. Sono bloccate in uno `skills.yaml` con commit, quindi colleghi e CI ricevono le stesse versioni. Le skill globali valgono per una sola macchina: usale per le skill personali e per strumenti come le skill di mestiere.
