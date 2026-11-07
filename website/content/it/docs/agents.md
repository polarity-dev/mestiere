---
title: Agenti
seoTitle: Agent Skills per Claude Code, Kiro, Codex e Cursor
nav: Agenti
description: Come mestiere installa le Agent Skills per Claude Code, Kiro, Codex, Cursor e altri agenti, come usare le copie e come farlo configurare al tuo agente.
order: 5
---

Un agente, o harness, è uno strumento di coding che legge le skill. mestiere salva ogni skill installata una sola volta in `.agents/skills/`, la cartella condivisa che Codex, Gemini CLI, Cursor, GitHub Copilot e altri agenti leggono direttamente. Gli agenti con una cartella propria, come Claude Code e Kiro, ricevono dei link a quella cartella.

## Agenti supportati {#supported-agents}

| Agente | ID | Cartella delle skill |
|---|---|---|
| Tutti (archivio canonico) | `agents` (sempre attivo) | `.agents/skills/` |
| Claude Code | `claude` | `.claude/skills/` |
| Kiro | `kiro` | `.kiro/skills/` |

Codex, Gemini CLI, Cursor, GitHub Copilot e qualsiasi altro harness che legge `.agents/skills/` vedono le skill installate senza impostazioni aggiuntive, quindi non serve una voce in `agents`.

## Cambia gli agenti di un progetto {#change-the-agents-of-a-project}

Modifica `agents` in `skills.yaml`, poi sincronizza:

```yaml
agents:
  - claude
  - kiro
```

```sh
mestiere sync
```

`sync` aggiunge i link per i nuovi agenti e li rimuove per gli agenti che hai tolto. Per sceglierli durante la configurazione, usa `mestiere init --agents claude,kiro`.

## Symlink o copie {#symlinks-or-copies}

Di default ogni cartella di un agente riceve un symlink relativo a `.agents/skills/<name>`. Se uno strumento non segue i symlink, dai alle cartelle degli agenti delle copie reali:

```yaml
link: copy
```

Oppure sceglilo durante la configurazione con `mestiere init --link copy`.

## Insegna al tuo agente a usare mestiere {#teach-your-agent-to-use-mestiere}

Il repository di mestiere è anche un registry. Contiene due skill che insegnano agli agenti di coding a usare mestiere nel modo corretto, anche per configurarlo in un nuovo repository:

| Skill | Per |
|---|---|
| `mestiere/project` | Repository che installano skill: aggiungere, aggiornare, sincronizzare, fare eject, fare commit, risolvere gli errori di `doctor` |
| `mestiere/registry` | Repository di registry: scaffolding, scrivere `SKILL.md`, condividere file tramite `_shared/`, risolvere gli errori di lint, CI |

Si installano a livello globale, per ogni progetto sulla tua macchina. L'installazione con Homebrew lo fa già; altrimenti esegui una volta:

```sh
mestiere add -g mestiere
```

I colleghi il cui agente non le ha eseguono lo stesso comando. Poi chiedi al tuo agente, da qualsiasi repository:

```text
Configura mestiere per questo repo con il nostro registry git@github.com:polarity-dev/skills.git
```

L'agente esegue `mestiere init`, ti mostra cosa offre il registry, installa le skill adatte al codebase e fa il commit di `skills.yaml`. Le richieste successive funzionano allo stesso modo:

- "Qualcuna delle nostre skill è obsoleta? Aggiornale e riassumi cosa è cambiato."
- "Voglio modificare frontend-forms per questo repo, tienila come nostra e fanne il commit."
- "Perché mestiere doctor fallisce?"

## Leggere questa documentazione da un agente {#reading-these-docs-from-an-agent}

Ogni pagina di questo sito ha una versione Markdown allo stesso URL più `index.md`, ad esempio [/docs/commands/index.md](/docs/commands/index.md). [llms.txt](/llms.txt) le elenca tutte, e [llms-full.txt](/llms-full.txt) contiene tutta la documentazione in un solo file.
