import type { Lang } from "../lib/site.ts"

export const REGISTRY_URL = "git@github.com:your-org/skills.git"

export const HOME_CODE = {
  brew: "brew install --cask polarity-dev/tap/mestiere",
  install: `mestiere init --agents claude,kiro
mestiere registry add polarity ${REGISTRY_URL}
mestiere add frontend backend/db-migrations
git add skills.yaml .gitignore
git commit -m "Add team skills"`,
  publish: `mkdir skills && cd skills && git init
mestiere registry init --name polarity
mestiere registry new-namespace frontend --description "Frontend practices"
mestiere registry new-skill frontend/forms --description "Build accessible forms"
mestiere registry lint`,
}

interface Feature {
  title: string
  html: string
}

interface HomeStrings {
  hero: {
    eyebrow: string
    title: string
    lead: string
    start: string
    install: string
    how: string
    metaHtml: string
  }

  how: { title: string, steps: Feature[] }
  quickStart: {
    title: string
    install: { title: string, html: string, more: string }
    publish: { title: string, html: string, more: string }
  }

  why: { title: string, leadHtml: string, features: Feature[] }
  agents: { title: string, introHtml: string, prompt: string, html: string[], more: string }
  faq: { title: string }
  cta: { title: string, commands: string, install: string, note: string, star: string }
}

export const HOME: Record<Lang, HomeStrings> = {
  en: {
    hero: {
      eyebrow: "Agent Skills manager for Claude Code, Kiro, Codex, Cursor and more",
      title: "Your team's agent skills, in every repository.",
      lead: "Keep skills in one git registry. Every project installs the ones it needs, pinned to an exact version.",
      start: "Get started",
      install: "Install mestiere",
      how: "How it works",
      metaHtml:
        'macOS and Linux · free and open source on <a href="https://github.com/polarity-dev/mestiere">GitHub</a>',
    },
    how: {
      title: "How it works",
      steps: [
        {
          title: "Keep skills in a registry",
          html: 'A registry is a plain git repository of <a href="https://agentskills.io">Agent Skills</a>, grouped in namespaces like <code>frontend</code> or <code>backend</code>. Review skills like any other code.',
        },
        {
          title: "Install them in each project",
          html: "<code>mestiere add frontend</code> copies the skills into <code>.agents/skills/</code> and locks each one to a registry commit in <code>skills.yaml</code>.",
        },
        {
          title: "Every agent sees them",
          html: "Codex, Gemini CLI, Cursor and Copilot read <code>.agents/skills/</code> directly; Claude Code and Kiro get links in their own folders. Teammates and CI run <code>mestiere sync</code> and get the same files.",
        },
      ],
    },
    quickStart: {
      title: "Quick start",
      install: {
        title: "Install skills in a project",
        html: "A teammate who clones the repository runs <code>mestiere sync</code>.",
        more: "Read the getting started guide",
      },
      publish: {
        title: "Publish skills in a registry",
        html: "Push it, and any project can add it.",
        more: "Learn how to author a registry",
      },
    },
    why: {
      title: "Why mestiere",
      leadHtml:
        "<em>Mestiere</em> is Italian for craft: the know-how a workshop hands from one person to the next. mestiere does that for coding agents.",
      features: [
        {
          title: "One source of truth",
          html: "Skills live in a git registry, public or private, reviewed like any other code. Each skill can carry a README for people that never reaches the agent's context.",
        },
        {
          title: "Organized by namespace",
          html: "Browse a registry namespace by namespace, then install a whole namespace or a single skill. New skills in an installed namespace arrive on the next update.",
        },
        {
          title: "Reproducible",
          html: "Every install is locked to a registry commit and a content hash. Teammates and CI get byte-identical skills.",
        },
        {
          title: "Every agent at once",
          html: "Files live once in <code>.agents/skills/</code>, which Codex, Gemini CLI, Cursor and others read directly. Claude Code and Kiro get symlinks to it.",
        },
        {
          title: "Out of your way",
          html: "Installed skills are ignored by git by default. Your own skills sit next to them untouched, and committing any skill is one command.",
        },
        {
          title: "Safe by construction",
          html: 'mestiere replaces a folder only when it can prove it\'s an unmodified copy of a published skill. <a href="/docs/how-it-works/#safety-model">See the safety model</a>.',
        },
        {
          title: "No lock-in",
          html: "Registries are plain git repositories. Installed skills are plain folders that follow the Agent Skills spec.",
        },
        {
          title: "Your git, your auth",
          html: "mestiere runs the system <code>git</code>, so SSH keys, credential helpers and <code>gh auth</code> just work.",
        },
      ],
    },
    agents: {
      title: "Or let your agent set it up",
      introHtml:
        "The Homebrew install also gives your agents mestiere's own skills, for every project on your machine. Then ask your agent, from any repository:",
      prompt: `Set up mestiere for this repo with our registry ${REGISTRY_URL}`,
      html: [
        'The agent runs <code>mestiere init</code>, shows you what the registry offers, installs the skills that fit the codebase and commits <code>skills.yaml</code>. Later prompts work the same way: "Are any of our skills outdated? Update them and summarize what changed".',
        "mestiere never waits on a prompt in an agent's shell: a missing argument fails with a hint, and <code>--json</code> gives parseable output.",
        'Installed another way? Run <code>mestiere add -g mestiere</code> once. Agents that read the web can start from <a href="/llms.txt">llms.txt</a>.',
      ],
      more: "More on agents",
    },
    faq: { title: "Frequently asked questions" },
    cta: {
      title: "Give every repository the same skills",
      commands: "Browse the commands",
      install: "Install mestiere",
      note: "mestiere runs in the terminal, on macOS and Linux.",
      star: "Star on GitHub",
    },
  },
  it: {
    hero: {
      eyebrow: "Gestore di Agent Skills per Claude Code, Kiro, Codex, Cursor e altri",
      title: "Le skill degli agenti del tuo team, in ogni repository.",
      lead: "Tieni le skill in un solo registry git. Ogni progetto installa quelle che gli servono, fissate a una versione esatta.",
      start: "Primi passi",
      install: "Installa mestiere",
      how: "Come funziona",
      metaHtml:
        'macOS e Linux · gratuito e open source su <a href="https://github.com/polarity-dev/mestiere">GitHub</a>',
    },
    how: {
      title: "Come funziona",
      steps: [
        {
          title: "Tieni le skill in un registry",
          html: 'Un registry è un normale repository git di <a href="https://agentskills.io">Agent Skills</a>, raggruppate in namespace come <code>frontend</code> o <code>backend</code>. Revisiona le skill come qualsiasi altro codice.',
        },
        {
          title: "Installale in ogni progetto",
          html: "<code>mestiere add frontend</code> copia le skill in <code>.agents/skills/</code> e blocca ognuna su un commit del registry in <code>skills.yaml</code>.",
        },
        {
          title: "Ogni agente le vede",
          html: "Codex, Gemini CLI, Cursor e Copilot leggono <code>.agents/skills/</code> direttamente; Claude Code e Kiro ricevono link nelle proprie cartelle. Colleghi e CI eseguono <code>mestiere sync</code> e ricevono gli stessi file.",
        },
      ],
    },
    quickStart: {
      title: "Avvio rapido",
      install: {
        title: "Installa le skill in un progetto",
        html: "Un collega che clona il repository esegue <code>mestiere sync</code>.",
        more: "Leggi la guida Primi passi",
      },
      publish: {
        title: "Pubblica le skill in un registry",
        html: "Fai il push, e qualsiasi progetto può aggiungerlo.",
        more: "Scopri come creare un registry",
      },
    },
    why: {
      title: "Perché mestiere",
      leadHtml:
        "Il <em>mestiere</em> è il saper fare che una bottega passa da una persona all'altra. mestiere fa lo stesso per gli agenti di programmazione.",
      features: [
        {
          title: "Una sola fonte di verità",
          html: "Le skill vivono in un registry git, pubblico o privato, revisionato come qualsiasi altro codice. Ogni skill può avere un README per le persone, che non entra mai nel contesto dell'agente.",
        },
        {
          title: "Organizzate per namespace",
          html: "Sfoglia un registry namespace per namespace, poi installa un intero namespace o una singola skill. Le nuove skill di un namespace installato arrivano al prossimo aggiornamento.",
        },
        {
          title: "Riproducibili",
          html: "Ogni installazione è bloccata su un commit del registry e un hash del contenuto. Colleghi e CI ricevono skill identiche byte per byte.",
        },
        {
          title: "Tutti gli agenti insieme",
          html: "I file vivono una sola volta in <code>.agents/skills/</code>, che Codex, Gemini CLI, Cursor e altri leggono direttamente. Claude Code e Kiro ricevono symlink verso di essa.",
        },
        {
          title: "Non ti intralcia",
          html: "Le skill installate sono ignorate da git per impostazione predefinita. Le tue skill restano accanto a loro intatte, e fare il commit di una skill richiede un solo comando.",
        },
        {
          title: "Sicuro per costruzione",
          html: 'mestiere sostituisce una cartella solo quando può dimostrare che è una copia non modificata di una skill pubblicata. <a href="/docs/how-it-works/#safety-model">Vedi il modello di sicurezza</a>.',
        },
        {
          title: "Nessun lock-in",
          html: "I registry sono normali repository git. Le skill installate sono normali cartelle che seguono la specifica Agent Skills.",
        },
        {
          title: "Il tuo git, la tua autenticazione",
          html: "mestiere usa il <code>git</code> di sistema, quindi chiavi SSH, credential helper e <code>gh auth</code> funzionano senza altro.",
        },
      ],
    },
    agents: {
      title: "Oppure fallo configurare al tuo agente",
      introHtml:
        "L'installazione con Homebrew dà anche ai tuoi agenti le skill di mestiere, per ogni progetto sulla tua macchina. Poi chiedi al tuo agente, da qualsiasi repository:",
      prompt: `Configura mestiere per questo repo con il nostro registry ${REGISTRY_URL}`,
      html: [
        'L\'agente esegue <code>mestiere init</code>, ti mostra cosa offre il registry, installa le skill adatte al codice e fa il commit di <code>skills.yaml</code>. I prompt successivi funzionano allo stesso modo: "Qualcuna delle nostre skill è obsoleta? Aggiornale e riassumi cosa è cambiato".',
        "Nella shell di un agente mestiere non resta mai in attesa di una risposta: se manca un argomento fallisce con un suggerimento, e <code>--json</code> dà un output da elaborare.",
        'Hai installato in un altro modo? Esegui una volta <code>mestiere add -g mestiere</code>. Gli agenti che leggono il web possono partire da <a href="/llms.txt">llms.txt</a>.',
      ],
      more: "Di più sugli agenti",
    },
    faq: { title: "Domande frequenti" },
    cta: {
      title: "Dai a ogni repository le stesse skill",
      commands: "Sfoglia i comandi",
      install: "Installa mestiere",
      note: "mestiere funziona nel terminale, su macOS e Linux.",
      star: "Aggiungi una stella su GitHub",
    },
  },
}
