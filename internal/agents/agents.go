// Package agents describes the agent harnesses mestiere installs skills for.
package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Precedence says which copy an agent loads when a skill with the same name
// is both in the project and in the home folder.
type Precedence string

const (
	GlobalWins  Precedence = "global"
	ProjectWins Precedence = "project"
	// Unknown means the agent doesn't document what it does.
	Unknown Precedence = ""
)

// Agent is a harness that loads skills from folders in the project and in the
// home folder. mestiere keeps real files in .agents/skills (~/.agents/skills
// for global skills) and links them into the folders agents read instead.
type Agent struct {
	ID   string
	Name string
	// Dir is the project folder that gets links, or "" when the agent reads
	// .agents/skills itself. Paths are relative and slash-separated.
	Dir string
	// GlobalDir is the home folder that gets links for global skills, or ""
	// when the agent reads ~/.agents/skills itself.
	GlobalDir string
	// ProjectReads and GlobalReads list every folder the agent loads skills
	// from, relative to the project root and to the home folder.
	ProjectReads, GlobalReads []string
	Precedence                Precedence
	// Home is the agent's configuration folder in the home directory; its
	// presence tells that the agent is installed.
	Home string
	// Aliases are other IDs accepted for the agent, such as vercel/skills names.
	Aliases []string
}

// CanonicalID identifies the .agents directory, which always holds the real skill files.
const CanonicalID = "agents"

const store = ".agents/skills"

// Canonical is the mandatory store every harness links to.
var Canonical = Agent{ID: CanonicalID, Name: "Agent Skills", Dir: store, GlobalDir: store}

// known lists the supported agents. Claude Code and Kiro document their
// precedence; the folders of Gemini CLI, Copilot and OpenCode follow
// vercel-labs/skills docs/agents.md, and Cursor's follow cursor.com/docs.
var known = []Agent{
	Canonical,
	{ID: "claude", Name: "Claude Code", Dir: ".claude/skills", GlobalDir: ".claude/skills",
		ProjectReads: []string{".claude/skills"}, GlobalReads: []string{".claude/skills"},
		Precedence: GlobalWins, Home: ".claude", Aliases: []string{"claude-code"}},
	{ID: "kiro", Name: "Kiro", Dir: ".kiro/skills", GlobalDir: ".kiro/skills",
		ProjectReads: []string{".kiro/skills"}, GlobalReads: []string{".kiro/skills"},
		Precedence: ProjectWins, Home: ".kiro", Aliases: []string{"kiro-cli"}},
	{ID: "cursor", Name: "Cursor",
		ProjectReads: []string{store, ".cursor/skills", ".claude/skills", ".codex/skills"},
		GlobalReads:  []string{store, ".cursor/skills", ".claude/skills", ".codex/skills"}, Home: ".cursor"},
	{ID: "codex", Name: "Codex", ProjectReads: []string{store}, GlobalReads: []string{store}, Home: ".codex"},
	{ID: "gemini", Name: "Gemini CLI", GlobalDir: ".gemini/skills",
		ProjectReads: []string{store}, GlobalReads: []string{".gemini/skills"}, Home: ".gemini", Aliases: []string{"gemini-cli"}},
	{ID: "copilot", Name: "GitHub Copilot", GlobalDir: ".copilot/skills",
		ProjectReads: []string{store}, GlobalReads: []string{".copilot/skills"}, Home: ".copilot", Aliases: []string{"github-copilot"}},
	{ID: "opencode", Name: "OpenCode", GlobalDir: ".config/opencode/skills",
		ProjectReads: []string{store}, GlobalReads: []string{".config/opencode/skills"}, Home: ".config/opencode"},
}

// Known returns every supported agent with the canonical store first.
func Known() []Agent {
	return append([]Agent(nil), known...)
}

// Harnesses returns every agent except the canonical store.
func Harnesses() []Agent {
	return Known()[1:]
}

// HarnessIDs returns the IDs accepted in the manifest's agents list.
func HarnessIDs() []string {
	var ids []string
	for _, a := range Harnesses() {
		ids = append(ids, a.ID)
	}
	return ids
}

// Installed reports whether the agent's configuration folder exists in home.
func (a Agent) Installed(home string) bool {
	if a.Home == "" || home == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(home, filepath.FromSlash(a.Home)))
	return err == nil && info.IsDir()
}

// LinkDir returns the folder that gets links in the project or, when global
// is set, in the home folder; "" means the agent reads the store directly.
func (a Agent) LinkDir(global bool) string {
	if global {
		return a.GlobalDir
	}
	return a.Dir
}

// Reads returns the folders the agent loads skills from in one scope.
func (a Agent) Reads(global bool) []string {
	if a.ID == CanonicalID {
		return []string{store}
	}
	if global {
		return a.GlobalReads
	}
	return a.ProjectReads
}

// agentEnv lists variables that coding agents set in the shells they run
// commands in. AI_AGENT and AGENT are the proposed vendor-neutral markers
// (agentsmd/agents.md#136); the rest are vendor-specific.
// shortcut: KIRO_SESSION_ID and VSCODE_AGENT_BASH_FIXUPS are observed in Kiro's agent shell, not documented; replace them once Kiro documents a marker.
var agentEnv = []string{
	"AI_AGENT", "AGENT", "CLAUDECODE", "CODEX_SANDBOX", "CURSOR_AGENT", "GEMINI_CLI",
	"KIRO_SESSION_ID", "VSCODE_AGENT_BASH_FIXUPS",
}

// InAgentShell reports whether getenv shows that a coding agent, not a person,
// runs this process. Agent shells often have a terminal attached, so a TTY
// check alone would let prompts block them.
func InAgentShell(getenv func(string) string) bool {
	for _, k := range agentEnv {
		if getenv(k) != "" {
			return true
		}
	}
	return false
}

func Lookup(id string) (Agent, bool) {
	for _, a := range known {
		if a.ID == id {
			return a, true
		}
		for _, alias := range a.Aliases {
			if alias == id {
				return a, true
			}
		}
	}
	return Agent{}, false
}

// Resolve validates harness IDs and returns them deduplicated and sorted.
// The canonical ID is accepted and dropped because it is always implied.
func Resolve(ids []string) ([]Agent, error) {
	seen := map[string]bool{}
	var out []Agent
	for _, id := range ids {
		if id == CanonicalID {
			continue
		}
		a, ok := Lookup(id)
		if !ok {
			return nil, fmt.Errorf("unknown agent %q (supported: %v)", id, HarnessIDs())
		}
		if seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
