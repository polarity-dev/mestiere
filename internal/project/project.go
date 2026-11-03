// Package project implements mestiere's operations on a project with a skills.yaml.
package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/gitignore"
	"github.com/polarity-dev/mestiere/internal/gitx"
	"github.com/polarity-dev/mestiere/internal/manifest"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/registry"
)

const (
	LinkSymlink = "symlink"
	LinkCopy    = "copy"
)

var registryNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type Project struct {
	Root     string
	Manifest *manifest.Manifest
	// Offline prevents network access; operations that need it fail or are skipped.
	Offline bool
	// Global marks the per-user project rooted at $HOME, where .gitignore rules don't apply.
	Global  bool
	mirrors map[string]*registry.Mirror
}

// BuiltinURL is mestiere's own registry, configured in every new global manifest.
const BuiltinURL = "https://github.com/polarity-dev/mestiere.git"

// OpenGlobal opens the per-user manifest, whose skills go to the agent
// directories under $HOME. Without a manifest it starts with mestiere's own
// registry and the agents that already have a directory there.
func OpenGlobal() (*Project, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(globalConfigDir(home), manifest.FileName)
	m, err := manifest.Load(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		m = manifest.New(p)
		m.Agents = homeAgents(home)
		// shortcut: the registry follows its default branch, not the binary's release tag; pin ref to the version if skills ever document unreleased commands.
		m.Registries = []*manifest.Registry{{Name: "mestiere", URL: BuiltinURL, Lock: map[string]manifest.LockEntry{}}}
	case err != nil:
		return nil, err
	}
	proj := newProject(home, m)
	proj.Global = true
	if err := proj.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return proj, nil
}

// globalConfigDir follows XDG_CONFIG_HOME on every platform, like CacheDir.
func globalConfigDir(home string) string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "mestiere")
	}
	return filepath.Join(home, ".config", "mestiere")
}

// homeAgents returns the harnesses with a config directory in home, or all of
// them when none has one, so global installs don't create folders for unused agents.
func homeAgents(home string) []string {
	var ids []string
	for _, a := range agents.Harnesses() {
		if info, err := os.Stat(filepath.Join(home, filepath.Dir(filepath.FromSlash(a.Dir)))); err == nil && info.IsDir() {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		return agents.HarnessIDs()
	}
	return ids
}

// inRepo reports whether .gitignore rules govern the project's skill folders.
func (p *Project) inRepo() bool { return !p.Global && gitignore.InRepo(p.Root) }

func Open(start string) (*Project, error) {
	p, err := manifest.Find(start)
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(p)
	if err != nil {
		return nil, err
	}
	proj := newProject(filepath.Dir(p), m)
	if err := proj.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return proj, nil
}

// Unmanaged returns a read-only view of a directory without skills.yaml, rooted
// at the enclosing git repository because that is where harnesses look for skills.
// Saving it would create a manifest, so callers must only inspect it.
func Unmanaged(start string) (*Project, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	if top, err := gitx.Run(root, "rev-parse", "--show-toplevel"); err == nil && top != "" {
		root = top
	}
	return newProject(root, manifest.New(filepath.Join(root, manifest.FileName))), nil
}

func newProject(root string, m *manifest.Manifest) *Project {
	return &Project{Root: root, Manifest: m, mirrors: map[string]*registry.Mirror{}}
}

// Init creates or updates skills.yaml in dir and ensures the .gitignore rule.
// A nil agentIDs keeps the existing agents.
func Init(dir string, agentIDs []string, link string) (*Project, *Report, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, err
	}
	r := &Report{}
	p := filepath.Join(abs, manifest.FileName)
	var m *manifest.Manifest
	switch _, err := os.Stat(p); {
	case err == nil:
		if m, err = manifest.Load(p); err != nil {
			return nil, nil, err
		}
	case errors.Is(err, fs.ErrNotExist):
		m = manifest.New(p)
		r.Action("create %s", manifest.FileName)
	default:
		return nil, nil, err
	}
	if agentIDs != nil {
		m.Agents = agentIDs
	}
	if link != "" {
		m.Link = link
	}
	proj := newProject(abs, m)
	if err := proj.validate(); err != nil {
		return nil, nil, err
	}
	if err := m.Save(); err != nil {
		return nil, nil, err
	}
	added, warnings, err := gitignore.EnsureRule(abs, proj.ignoreProbes())
	if err != nil {
		return nil, nil, err
	}
	if added {
		r.Action("add %q to .gitignore", gitignore.Rule)
	}
	r.Warnings = append(r.Warnings, warnings...)
	return proj, r, nil
}

func (p *Project) validate() error {
	m := p.Manifest
	if _, err := agents.Resolve(m.Agents); err != nil {
		return err
	}
	switch m.Link {
	case "", LinkSymlink, LinkCopy:
	default:
		return fmt.Errorf("link must be %q or %q, got %q", LinkSymlink, LinkCopy, m.Link)
	}
	seen := map[string]bool{}
	for _, reg := range m.Registries {
		if !registryNameRe.MatchString(reg.Name) {
			return fmt.Errorf("invalid registry name %q: use lowercase letters, digits, - and _", reg.Name)
		}
		if seen[reg.Name] {
			return fmt.Errorf("registry %q is defined twice", reg.Name)
		}
		seen[reg.Name] = true
		if reg.URL == "" {
			return fmt.Errorf("registry %s: url is required", reg.Name)
		}
		for _, raw := range reg.Install {
			if _, err := naming.ParseSpec(raw); err != nil {
				return fmt.Errorf("registry %s: install %q: %w", reg.Name, raw, err)
			}
		}
		for _, raw := range reg.Exclude {
			spec, err := naming.ParseSpec(raw)
			if err != nil {
				return fmt.Errorf("registry %s: exclude %q: %w", reg.Name, raw, err)
			}
			if spec.IsNamespace() {
				return fmt.Errorf("registry %s: exclude %q must name a single skill", reg.Name, raw)
			}
		}
	}
	return nil
}

func (p *Project) harnesses() []agents.Agent {
	h, _ := agents.Resolve(p.Manifest.Agents)
	return h
}

func (p *Project) ignoreProbes() []string {
	probes := []string{path.Join(agents.Canonical.Dir, "mestiere-probe")}
	for _, a := range p.harnesses() {
		probes = append(probes, path.Join(a.Dir, "mestiere-probe"))
	}
	return probes
}

func (p *Project) agentDir(a agents.Agent) string {
	return filepath.Join(p.Root, filepath.FromSlash(a.Dir))
}

func (p *Project) canonicalDir() string { return p.agentDir(agents.Canonical) }

func (p *Project) rel(abs string) string {
	r, err := filepath.Rel(p.Root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(r)
}

// resolveURL makes relative local registry paths independent of the working directory.
func (p *Project) resolveURL(url string) string {
	if strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") {
		return filepath.Join(p.Root, url)
	}
	return url
}

func (p *Project) mirror(reg *manifest.Registry) (*registry.Mirror, error) {
	url := p.resolveURL(reg.URL)
	if m, ok := p.mirrors[url]; ok {
		return m, nil
	}
	m, err := registry.Open(url)
	if err != nil {
		return nil, err
	}
	p.mirrors[url] = m
	return m, nil
}

func (p *Project) fetch(m *registry.Mirror) error {
	if p.Offline {
		if m.Exists() {
			return nil
		}
		return fmt.Errorf("registry %s isn't cached and network access is disabled", m.URL)
	}
	return m.Fetch()
}

func (p *Project) linkMode() string {
	if p.Manifest.Link == "" {
		return LinkSymlink
	}
	return p.Manifest.Link
}

func (p *Project) Save() error { return p.Manifest.Save() }

// pickRegistry returns the named registry, or the only one when name is empty.
func (p *Project) pickRegistry(name string) (*manifest.Registry, error) {
	regs := p.Manifest.Registries
	if name != "" {
		if reg := p.Manifest.Registry(name); reg != nil {
			return reg, nil
		}
		return nil, fmt.Errorf("unknown registry %q", name)
	}
	switch len(regs) {
	case 0:
		return nil, errors.New("no registries configured; add one with `mestiere registry add <name> <url>`")
	case 1:
		return regs[0], nil
	default:
		return nil, errors.New("several registries are configured; choose one with --registry")
	}
}

func (p *Project) fetchRef(reg *manifest.Registry) (*registry.Mirror, string, error) {
	m, err := p.mirror(reg)
	if err != nil {
		return nil, "", err
	}
	if err := p.fetch(m); err != nil {
		return nil, "", err
	}
	commit, err := m.Resolve(reg.Ref)
	if err != nil {
		return nil, "", err
	}
	return m, commit, nil
}

// Catalog lists what a registry offers at its configured ref.
func (p *Project) Catalog(name string) (map[string][]string, error) {
	reg, err := p.pickRegistry(name)
	if err != nil {
		return nil, err
	}
	m, commit, err := p.fetchRef(reg)
	if err != nil {
		return nil, err
	}
	return m.Catalog(commit)
}

type Offered struct {
	Spec        naming.Spec
	Description string
	Installed   bool
}

type Listing struct {
	Registry string
	Commit   string
	Skills   []Offered
}

// Browse lists every skill a registry offers at its configured ref, with
// descriptions and whether this project has it locked.
func (p *Project) Browse(name string) (*Listing, error) {
	reg, err := p.pickRegistry(name)
	if err != nil {
		return nil, err
	}
	m, commit, err := p.fetchRef(reg)
	if err != nil {
		return nil, err
	}
	catalog, err := m.Catalog(commit)
	if err != nil {
		return nil, err
	}
	var specs []naming.Spec
	for ns, skills := range catalog {
		for _, s := range skills {
			specs = append(specs, naming.Spec{Namespace: ns, Skill: s})
		}
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].String() < specs[j].String() })
	descriptions, err := m.Descriptions(commit, specs)
	if err != nil {
		return nil, err
	}
	l := &Listing{Registry: reg.Name, Commit: commit}
	for _, s := range specs {
		_, installed := reg.Lock[s.InstalledName()]
		l.Skills = append(l.Skills, Offered{Spec: s, Description: descriptions[s], Installed: installed})
	}
	return l, nil
}

func (p *Project) AddRegistry(name, url, ref string) error {
	if p.Manifest.Registry(name) != nil {
		return fmt.Errorf("registry %q already exists", name)
	}
	p.Manifest.Registries = append(p.Manifest.Registries, &manifest.Registry{
		Name: name, URL: url, Ref: ref, Lock: map[string]manifest.LockEntry{},
	})
	if err := p.validate(); err != nil {
		p.Manifest.RemoveRegistry(name)
		return err
	}
	return p.Save()
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func without(list []string, drop func(string) bool) []string {
	var out []string
	for _, v := range list {
		if !drop(v) {
			out = append(out, v)
		}
	}
	return out
}
