package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/polarity-dev/mestiere/internal/agents"
)

// globalDirs lists every home folder some agent loads skills from, the store first.
func globalDirs() []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range agents.Known() {
		for _, dir := range a.Reads(true) {
			if !seen[dir] {
				seen[dir] = true
				out = append(out, dir)
			}
		}
	}
	return out
}

// Shadow is a skill that an agent finds both in the project and in the home
// folder. Loads says which copy the agent uses; Unknown when it isn't documented.
type Shadow struct {
	Name  string            `json:"name"`
	Agent string            `json:"agent"`
	Path  string            `json:"global_path"`
	Loads agents.Precedence `json:"loads"`
}

// shadowWarning explains a shadow whose global copy wins or may win.
func (s Shadow) warning() string {
	if s.Loads == agents.GlobalWins {
		return fmt.Sprintf("%s loads %s instead of this project's %s; remove the global copy or rename one of them", s.Agent, s.Path, s.Name)
	}
	return fmt.Sprintf("%s is also in %s, and %s doesn't document which copy it loads", s.Name, s.Path, s.Agent)
}

// warnShadows reports, for the named skills or all when names is nil, the
// shadows where the project copy may not be the one the agent loads.
func (p *Project) warnShadows(r *Report, names map[string]bool) {
	for _, s := range p.Shadows() {
		if names != nil && !names[s.Name] {
			continue
		}
		switch s.Loads {
		case agents.GlobalWins:
			r.Warn("%s", s.warning())
		case agents.Unknown:
			r.Hint("%s", s.warning())
		}
	}
}

// Shadows finds project skills that an agent in use, configured or installed
// on this machine, also finds in the home folder under the same name. A team locks a version in the
// project, so a global copy that wins hides that version from the agent.
func (p *Project) Shadows() []Shadow {
	if p.Global {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil || samePath(home, p.Root) {
		return nil
	}
	configured := map[string]bool{}
	for _, a := range p.harnesses() {
		configured[a.ID] = true
	}
	var out []Shadow
	for _, a := range agents.Harnesses() {
		if !configured[a.ID] && !a.Installed(home) {
			continue
		}
		project := map[string]string{}
		for _, dir := range a.Reads(false) {
			for _, name := range skillDirs(filepath.Join(p.Root, filepath.FromSlash(dir))) {
				if _, ok := project[name]; !ok {
					project[name] = filepath.Join(p.Root, filepath.FromSlash(dir), name)
				}
			}
		}
		seen := map[string]bool{}
		for _, dir := range a.Reads(true) {
			for _, name := range skillDirs(filepath.Join(home, filepath.FromSlash(dir))) {
				local, ok := project[name]
				global := filepath.Join(home, filepath.FromSlash(dir), name)
				if !ok || seen[name] || samePath(local, global) {
					continue
				}
				seen[name] = true
				out = append(out, Shadow{Name: name, Agent: a.ID, Path: path.Join("~", dir, name), Loads: a.Precedence})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Agent < out[j].Agent
	})
	return out
}
