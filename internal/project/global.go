package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/skillmd"
)

// Origins of a skill folder mestiere doesn't manage.
const (
	OriginCopy   = "copy"
	OriginLink   = "link"
	OriginBroken = "broken link"
)

// originOf tells a real folder from a link, and where the link points.
func originOf(dir string) (origin, target string) {
	info, err := os.Lstat(dir)
	if err != nil || info.Mode()&fs.ModeSymlink == 0 {
		return OriginCopy, ""
	}
	target, _ = os.Readlink(dir)
	if isBroken(dir) {
		return OriginBroken, target
	}
	return OriginLink, target
}

// scanDirs are the folders, relative to the root, where this scope's skills
// live: the link folders in a project, every folder an agent reads at home.
func (p *Project) scanDirs() []string {
	if p.Global {
		return globalDirs()
	}
	var dirs []string
	for _, a := range p.linkAgents() {
		dirs = append(dirs, p.linkRel(a))
	}
	return dirs
}

// AgentLoad is what one agent loads from the home folder in every session:
// each skill's name and description sit in its context before any work starts.
type AgentLoad struct {
	Agent  string `json:"agent"`
	Skills int    `json:"skills"`
	Chars  int    `json:"chars"`
}

// GlobalLoad measures, for each agent in use on this machine, the global
// skills it loads and the characters their names and descriptions take.
func (p *Project) GlobalLoad() []AgentLoad {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	configured := map[string]bool{}
	if p.Global {
		for _, a := range p.harnesses() {
			configured[a.ID] = true
		}
	}
	var out []AgentLoad
	for _, a := range agents.Harnesses() {
		if !configured[a.ID] && !a.Installed(home) {
			continue
		}
		load := AgentLoad{Agent: a.ID}
		seen := map[string]bool{}
		for _, dir := range a.Reads(true) {
			base := filepath.Join(home, filepath.FromSlash(dir))
			for _, name := range skillDirs(base) {
				if seen[name] {
					continue
				}
				seen[name] = true
				load.Skills++
				load.Chars += len(name)
				if data, err := os.ReadFile(filepath.Join(base, name, "SKILL.md")); err == nil {
					if fm, err := skillmd.Parse(data); err == nil {
						load.Chars += len(fm.Description)
					}
				}
			}
		}
		out = append(out, load)
	}
	return out
}

// doctorGlobal reports global skills that cost context without anyone
// keeping them current: broken links, drifting copies, unmanaged skills.
func (p *Project) doctorGlobal(r *Report, st *Status) {
	copies := map[string][]string{}
	var unmanaged []string
	seen := map[string]bool{}
	for _, l := range st.Local {
		switch l.Origin {
		case OriginBroken:
			r.Warn("~/%s is a broken link to %s; `mestiere prune -g %s` moves it to the trash", l.Path, l.Target, l.Name)
		case OriginCopy:
			copies[l.Name] = append(copies[l.Name], "~/"+l.Path)
		}
		if !seen[l.Name] {
			seen[l.Name] = true
			unmanaged = append(unmanaged, l.Name)
		}
	}
	for _, name := range sortedKeys(copies) {
		if paths := copies[name]; len(paths) > 1 {
			r.Warn("%s has %d separate copies (%s) that can drift apart", name, len(paths), strings.Join(paths, ", "))
		}
	}
	if len(unmanaged) > 0 {
		r.Hint("%d global skills aren't managed by mestiere: %s. `mestiere adopt -g <name>` manages one a registry publishes; `mestiere prune -g <name>` moves one to the trash",
			len(unmanaged), strings.Join(unmanaged, ", "))
	}
	for _, l := range p.GlobalLoad() {
		r.Hint("%s loads %d global skills in every session, %d characters of names and descriptions", l.Agent, l.Skills, l.Chars)
	}
}

// Adopt puts skills mestiere doesn't manage under its management, when a
// configured registry publishes a skill with the same name and content. The
// folders already on disk become the installed copies.
func (p *Project) Adopt(args []string) (*Report, error) {
	r := &Report{}
	desired, err := p.desired()
	if err != nil {
		return nil, err
	}
	s := p.newSyncer(SyncOptions{DryRun: true}, &Report{}, desired)
	for _, name := range args {
		if _, ok := desired[name]; ok {
			r.Warn("%s is already managed by mestiere", name)
			continue
		}
		dirs := p.foldersNamed(name)
		if len(dirs) == 0 {
			return r, fmt.Errorf("%s isn't in any skills folder", name)
		}
		reg, spec, ok := p.publisher(s, name, dirs)
		if !ok {
			return r, fmt.Errorf("no configured registry publishes %s with this content; keep it as your own skill, or add the registry that publishes it", name)
		}
		ar, err := p.Add(reg, []string{spec.String()}, false)
		r.Merge(ar)
		if err != nil {
			return r, err
		}
	}
	return r, nil
}

// publisher finds the registry skill whose published content matches one of dirs.
func (p *Project) publisher(s *syncer, name string, dirs []string) (string, naming.Spec, bool) {
	cands, _ := naming.Candidates(name)
	plain, plainErr := naming.ParsePlain(name)
	for _, reg := range p.Manifest.Registries {
		specs := cands
		if reg.Flat() {
			if plainErr != nil {
				continue
			}
			specs = []naming.Spec{plain}
		}
		for _, spec := range specs {
			if spec.IsNamespace() {
				continue
			}
			for _, dir := range dirs {
				if s.published(reg, spec, dir) == statePublished {
					return reg.Name, spec, true
				}
			}
		}
	}
	return "", naming.Spec{}, false
}

// foldersNamed lists the real skill folders called name in this scope.
func (p *Project) foldersNamed(name string) []string {
	var out []string
	for _, dir := range p.scanDirs() {
		full := filepath.Join(p.Root, filepath.FromSlash(dir), name)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			out = append(out, full)
		}
	}
	return out
}

// TrashDir is where prune moves skills, outside the cache so cleaning the
// cache never loses them.
func TrashDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "mestiere", "trash"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "mestiere", "trash"), nil
}

// Prune moves global skills mestiere doesn't manage, with every link and copy
// of them in the home folder, to a dated folder in the trash. Nothing is
// deleted, so moving a folder back undoes it.
func (p *Project) Prune(names []string) (*Report, error) {
	if !p.Global {
		return nil, errors.New("prune cleans up global skills; run it with -g. In a project, local skills are yours to delete or commit")
	}
	desired, err := p.desired()
	if err != nil {
		return nil, err
	}
	trash, err := TrashDir()
	if err != nil {
		return nil, err
	}
	batch := filepath.Join(trash, time.Now().Format("20060102-150405"))
	r := &Report{}
	for _, name := range names {
		if _, ok := desired[name]; ok {
			return r, fmt.Errorf("%s is managed by mestiere; use `mestiere remove -g %s`", name, name)
		}
		var moved []string
		for _, dir := range p.scanDirs() {
			src := filepath.Join(p.Root, filepath.FromSlash(dir), name)
			if _, err := os.Lstat(src); err != nil {
				continue
			}
			dst := filepath.Join(batch, filepath.FromSlash(dir), name)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return r, err
			}
			if err := os.Rename(src, dst); err != nil {
				return r, err
			}
			moved = append(moved, path.Join("~", dir, name))
		}
		if len(moved) == 0 {
			return r, fmt.Errorf("%s isn't in any global skills folder", name)
		}
		for _, m := range moved {
			r.Action("trash %s", m)
		}
	}
	if len(r.Actions) > 0 {
		r.Hint("moved to %s; move a folder back to restore it", batch)
	}
	return r, nil
}

// UnmanagedNames lists the skills in this scope that mestiere doesn't manage.
func (p *Project) UnmanagedNames() []string {
	st, err := p.Status()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, l := range st.Local {
		seen[l.Name] = true
	}
	return sortedKeys(seen)
}
