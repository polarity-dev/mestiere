---
title: Skill globali
seoTitle: Installa Agent Skills per ogni progetto con mestiere -g
nav: Skill globali
description: Installa Agent Skills per ogni progetto sulla tua macchina con mestiere -g: dove vanno file e manifest, quali comandi valgono e quando preferire un progetto.
order: 5.5
---

Le skill globali si installano una volta per tutta la tua macchina invece che per un solo repository. Gli agenti le caricano in ogni progetto. Aggiungi `-g` a un comando per agire su di esse:

```sh
mestiere add -g mestiere                  # le skill di mestiere; l'installazione con Homebrew esegue questo comando
mestiere registry add -g polarity git@github.com:polarity-dev/skills.git
mestiere add -g -r polarity frontend
mestiere status -g
```

## Dove vanno le cose {#where-things-go}

| Cosa | Dove |
|---|---|
| Manifest | `~/.config/mestiere/skills.yaml`, oppure `$XDG_CONFIG_HOME/mestiere/skills.yaml` |
| File delle skill | `~/.agents/skills/<namespace>-<skill>/` |
| Link | `~/.claude/skills/`, `~/.kiro/skills/`, per gli agenti che hanno una cartella nella tua home directory |

Il manifest ha lo stesso formato dello [skills.yaml](/docs/skills-yaml/) di un progetto, con `install`, `exclude` e il lock. Un nuovo manifest globale ha già configurato il registry `mestiere`, quindi `mestiere add -g mestiere` funziona senza `registry add`.

Se nessuno degli agenti ha ancora una cartella nella tua home directory, mestiere li collega tutti. Altrimenti non crea cartelle per gli agenti che non usi.

## Comandi {#commands}

Questi comandi accettano `-g`: `add`, `remove`, `update`, `outdated`, `sync`, `status`, `doctor`, `eject` e i sottocomandi di `registry`. Funzionano esattamente come in un progetto, con lo stesso [modello di sicurezza](/docs/how-it-works/#safety-model): una skill globale che hai modificato non viene mai sovrascritta.

`init`, `track`, `untrack` e `eject --track` non si applicano, perché niente di globale è in git.

## Le skill di mestiere {#mestieres-own-skills}

`mestiere add -g mestiere` installa `mestiere-project` e `mestiere-registry`, che insegnano al tuo agente a configurare e usare mestiere in qualsiasi repository. L'installazione con Homebrew lo fa per te e le tiene aggiornate con `brew upgrade`. Vedi [Agenti](/docs/agents/#teach-your-agent-to-use-mestiere).

Per rimuoverle, ad esempio prima di disinstallare:

```sh
mestiere remove -g mestiere
```

## Vedere le skill globali da un progetto {#seeing-global-skills-from-a-project}

`mestiere status` in un progetto elenca anche le skill in `~/.agents/skills`, `~/.claude/skills` e `~/.kiro/skills`, globali o no, perché anche da lì le caricano gli agenti. Funziona anche nelle cartelle senza `skills.yaml`.

## Globali o di progetto? {#global-or-project}

Per le skill su cui un team fa affidamento, preferisci l'installazione nel progetto. Sono bloccate in uno `skills.yaml` con commit, quindi colleghi e CI ricevono le stesse versioni. Le skill globali valgono per una sola macchina: usale per le skill personali e per strumenti come le skill di mestiere.
