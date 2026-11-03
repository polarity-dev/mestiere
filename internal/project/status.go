package project

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/gitignore"
)

type State string

const (
	StateOK         State = "ok"
	StateMissing    State = "missing"
	StateOutdated   State = "outdated"
	StateModified   State = "modified"
	StateUnverified State = "unverified"
	StateWrong      State = "wrong"
)

type SkillStatus struct {
	Name     string
	Registry string
	Commit   string
	State    State
	// Links maps each configured harness ID to the state of its entry.
	Links map[string]State
	// Committed is true when a .gitignore exception commits the skill.
	Committed bool
}

// LocalSkill is a skill directory mestiere doesn't manage.
type LocalSkill struct {
	Name      string
	Path      string
	Committed bool
}

type Status struct {
	Drift  []string
	Skills []SkillStatus
	Local  []LocalSkill
	// Global lists skills in the home-level agent directories; Committed is always false.
	Global []LocalSkill
}

func (p *Project) Status() (*Status, error) {
	desired, err := p.desired()
	if err != nil {
		return nil, err
	}
	st := &Status{}
	for _, reg := range p.Manifest.Registries {
		st.Drift = append(st.Drift, drift(reg)...)
	}
	s := p.newSyncer(SyncOptions{DryRun: true}, &Report{}, desired)
	inRepo := p.inRepo()
	committed := func(rel string) bool {
		if !inRepo {
			return false
		}
		ignored, err := gitignore.IsIgnored(p.Root, rel)
		return err == nil && !ignored
	}
	for _, m := range desired {
		dir := filepath.Join(p.canonicalDir(), m.name)
		ss := SkillStatus{
			Name: m.name, Registry: m.reg.Name, Commit: m.entry.Commit,
			State: p.contentStatus(s, dir, m), Links: map[string]State{},
			Committed: committed(p.rel(dir)),
		}
		for _, a := range p.harnesses() {
			ss.Links[a.ID] = p.linkStatus(s, a, m)
		}
		st.Skills = append(st.Skills, ss)
	}
	sort.Slice(st.Skills, func(i, j int) bool { return st.Skills[i].Name < st.Skills[j].Name })
	for _, a := range agents.Known() {
		for _, name := range skillDirs(p.agentDir(a)) {
			if _, ok := desired[name]; ok {
				continue
			}
			rel := path.Join(a.Dir, name)
			st.Local = append(st.Local, LocalSkill{Name: name, Path: rel, Committed: committed(rel)})
		}
	}
	// Harnesses also load skills from the same directories under $HOME.
	if home, err := os.UserHomeDir(); err == nil && !samePath(home, p.Root) {
		for _, a := range agents.Known() {
			for _, name := range skillDirs(filepath.Join(home, filepath.FromSlash(a.Dir))) {
				st.Global = append(st.Global, LocalSkill{Name: name, Path: path.Join("~", a.Dir, name)})
			}
		}
	}
	return st, nil
}

// skillDirs lists the non-hidden directories in dir, following symlinks.
func skillDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || !info.IsDir() {
			continue
		}
		names = append(names, name)
	}
	return names
}

func samePath(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

func (p *Project) contentStatus(s *syncer, dir string, m managed) State {
	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return StateMissing
	}
	if err != nil || !info.IsDir() {
		return StateWrong
	}
	switch s.classify(dir, m) {
	case stateCurrent:
		return StateOK
	case statePublished:
		return StateOutdated
	case stateUnverifiable:
		return StateUnverified
	default:
		return StateModified
	}
}

func (p *Project) linkStatus(s *syncer, a agents.Agent, m managed) State {
	dst := filepath.Join(p.agentDir(a), m.name)
	src := filepath.Join(p.canonicalDir(), m.name)
	info, err := os.Lstat(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return StateMissing
	case err != nil:
		return StateWrong
	case info.Mode()&fs.ModeSymlink != 0:
		if readlink(dst) == linkTarget(src, dst) {
			return StateOK
		}
		return StateWrong
	case info.IsDir():
		return p.contentStatus(s, dst, m)
	default:
		return StateWrong
	}
}
