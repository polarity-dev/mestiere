// Package agents describes the agent harnesses mestiere installs skills for.
package agents

import (
	"fmt"
	"sort"
)

// Agent is a harness that discovers skills from a project-relative directory.
type Agent struct {
	ID string
	// Dir is project-relative and slash-separated.
	Dir string
}

// CanonicalID identifies the .agents directory, which always holds the real skill files.
const CanonicalID = "agents"

// Canonical is the mandatory store every harness links to.
var Canonical = Agent{ID: CanonicalID, Dir: ".agents/skills"}

var known = []Agent{
	Canonical,
	{ID: "claude", Dir: ".claude/skills"},
	{ID: "kiro", Dir: ".kiro/skills"},
}

// Known returns every supported agent with the canonical store first.
func Known() []Agent {
	return append([]Agent(nil), known...)
}

// Harnesses returns the agents that receive links to the canonical store.
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
	}
	return Agent{}, false
}

// Resolve validates harness IDs and returns them deduplicated and sorted.
// The canonical ID is accepted and dropped because it is always implied.
func Resolve(ids []string) ([]Agent, error) {
	seen := map[string]bool{}
	var out []Agent
	for _, id := range ids {
		if id == CanonicalID || seen[id] {
			continue
		}
		a, ok := Lookup(id)
		if !ok {
			return nil, fmt.Errorf("unknown agent %q (supported: %v)", id, HarnessIDs())
		}
		seen[id] = true
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
