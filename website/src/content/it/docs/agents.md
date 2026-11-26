---
title: Agenti
seoTitle: Agent Skills per Claude Code, Kiro, Codex e Cursor
nav: Agenti
description: Come mestiere installa le Agent Skills per Claude Code, Kiro, Codex, Cursor e altri agenti, come lo usano gli agenti e come farlo configurare al tuo agente.
order: 5
---

Un agente, o harness, è uno strumento di coding che legge le skill. mestiere salva ogni skill installata una sola volta in `.agents/skills/`, la cartella condivisa che Codex, Cursor, Gemini CLI, GitHub Copilot e altri agenti leggono direttamente. Gli agenti con una cartella propria ricevono dei link a quella cartella. `.agents/skills/` è la fonte di verità; tutte le altre cartelle contengono solo link.

## Agenti supportati {#supported-agents}

| Agente | ID | Link in un progetto | Link per le skill globali | Stesso nome nel progetto e in `~` |
|---|---|---|---|---|
| Tutti (archivio) | `agents` (sempre attivo) | `.agents/skills/` | `~/.agents/skills/` | |
| Claude Code | `claude` (`claude-code`) | `.claude/skills/` | `~/.claude/skills/` | carica la copia globale |
| Kiro | `kiro` (`kiro-cli`) | `.kiro/skills/` | `~/.kiro/skills/` | carica la copia del progetto |
| Cursor | `cursor` | nessuno, legge `.agents/skills/` | nessuno, legge `~/.agents/skills/` | non documentato |
| Codex | `codex` | nessuno | nessuno | non documentato |
| Gemini CLI | `gemini` (`gemini-cli`) | nessuno | `~/.gemini/skills/` | non documentato |
| GitHub Copilot | `copilot` (`github-copilot`) | nessuno | `~/.copilot/skills/` | non documentato |
| OpenCode | `opencode` | nessuno | `~/.config/opencode/skills/` | non documentato |

Elenca in `agents` quelli che hanno bisogno di link. Gli agenti che leggono `.agents/skills/` vedono le skill del progetto senza impostazioni. Anche gli ID tra parentesi sono accettati.

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

## Skill installate in entrambi i posti {#skills-installed-in-both-places}

Una skill può esistere nel progetto e nella tua home con lo stesso nome, ad esempio una skill del team bloccata in `skills.yaml` e una vecchia copia personale. Quale delle due viene usata dipende dall'agente: Claude Code carica la copia personale, quindi la versione bloccata del team non gli arriva mai.

`status` elenca queste skill sotto "also installed globally", con la copia che carica ogni agente. `doctor` avvisa quando un agente carica la copia globale, e `add` avvisa subito dopo aver installato una skill del genere. Nessuno dei due fallisce, nemmeno con `--strict`, perché la copia globale sta su una sola macchina.

Per risolvere, rimuovi la copia globale oppure rinomina una delle due:

```sh
mestiere remove -g frontend-forms   # installata a livello globale con mestiere
mestiere prune -g frontend-forms    # installata in un altro modo
```

## Usare mestiere da un agente {#running-mestiere-from-an-agent}

mestiere chiede input solo quando al terminale c'è una persona. Ogni domanda ha un flag o un argomento che la sostituisce, quindi niente richiede una persona:

```sh
mestiere init --no-input --agents claude,kiro
mestiere registry add --no-input polarity git@github.com:your-org/skills.git
mestiere add --no-input frontend backend/db-migrations
```

I prompt sono disattivati, e un argomento mancante fallisce con un suggerimento che dice cosa passare, quando vale una di queste condizioni:

- il comando ha `--no-input`;
- `MESTIERE_NO_INPUT` o `CI` è impostata;
- stdin o stdout non è un terminale;
- la shell appartiene a un agente di coding, riconosciuto da `AI_AGENT`, `AGENT`, `CLAUDECODE`, `CODEX_SANDBOX`, `CURSOR_AGENT`, `GEMINI_CLI` o dalle variabili che imposta la shell dell'agente di Kiro.

Senza prompt, mestiere imposta anche `GIT_TERMINAL_PROMPT=0` e, a meno che tu non abbia configurato `core.sshCommand` o `GIT_SSH_COMMAND`, `ssh -o BatchMode=yes`, così git fallisce invece di aspettare una password.

Le shell degli agenti hanno spesso un terminale collegato, quindi le skill di mestiere dicono agli agenti di passare `--no-input` a ogni comando invece di affidarsi al riconoscimento automatico. Per gli agenti l'output è testo semplice, con le descrizioni intere in `registry show`. `status`, `registry show`, `outdated` e `doctor` stampano JSON con `--json`. Un nome scritto male fallisce con un suggerimento `did you mean …?` che l'agente può verificare prima di riprovare.

## Insegna al tuo agente a usare mestiere {#teach-your-agent-to-use-mestiere}

Il repository di mestiere è anche un registry. Contiene due skill che insegnano agli agenti di coding a usare mestiere nel modo corretto, anche per configurarlo in un nuovo repository:

| Skill | Per |
|---|---|
| `mestiere-project` | Repository che installano skill: aggiungere, aggiornare, sincronizzare, fare eject, fare commit, fare pulizia tra le skill globali, risolvere gli errori di `doctor` |
| `mestiere-registry` | Repository di registry: scaffolding, scrivere `SKILL.md`, condividere file tramite `_shared/`, risolvere gli errori di lint, CI |

Si installano a livello globale, per ogni progetto sulla tua macchina. L'installazione con Homebrew lo fa già; altrimenti esegui una volta:

```sh
mestiere add -g mestiere
```

I colleghi il cui agente non le ha eseguono lo stesso comando. Poi chiedi al tuo agente, da qualsiasi repository:

```text
Configura mestiere per questo repo con il nostro registry git@github.com:your-org/skills.git
```

L'agente esegue `mestiere init`, ti mostra cosa offre il registry, installa le skill adatte al codebase e fa il commit di `skills.yaml`. Le richieste successive funzionano allo stesso modo:

- "Qualcuna delle nostre skill è obsoleta? Aggiornale e riassumi cosa è cambiato."
- "Voglio modificare frontend-forms per questo repo, tienila come nostra e fanne il commit."
- "Perché mestiere doctor fallisce?"
- "Fai pulizia tra le mie skill globali."

## Leggere questa documentazione da un agente {#reading-these-docs-from-an-agent}

Ogni pagina di questo sito ha una versione Markdown allo stesso URL più `index.md`, ad esempio [/docs/commands/index.md](/docs/commands/index.md). [llms.txt](/llms.txt) le elenca tutte, e [llms-full.txt](/llms-full.txt) contiene tutta la documentazione in un solo file.
