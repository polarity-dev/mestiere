package project

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/polarity-dev/mestiere/internal/manifest"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/registry"
)

type ChangeKind string

const (
	Added   ChangeKind = "added"
	Updated ChangeKind = "updated"
	Removed ChangeKind = "removed"
)

type Change struct {
	Registry string
	Name     string
	Kind     ChangeKind
	From, To string
}

func (c Change) String() string {
	switch c.Kind {
	case Added:
		return fmt.Sprintf("%s: added at %s (%s)", c.Name, registry.Short(c.To), c.Registry)
	case Updated:
		return fmt.Sprintf("%s: %s -> %s (%s)", c.Name, registry.Short(c.From), registry.Short(c.To), c.Registry)
	default:
		return fmt.Sprintf("%s: removed (%s)", c.Name, c.Registry)
	}
}

// target selects what update/outdated/remove act on.
type target struct {
	registry string // empty matches every registry
	spec     naming.Spec
	raw      string
}

func (t target) appliesTo(reg string) bool { return t.registry == "" || t.registry == reg }

func (t target) wholeRegistry() bool { return t.spec.Namespace == "" }

// parseTargets accepts "ns", "ns/skill", "ns-skill", "reg:ns..." and "reg:".
func (p *Project) parseTargets(args []string) ([]target, error) {
	var out []target
	for _, a := range args {
		t := target{raw: a}
		rest := a
		if reg, after, ok := strings.Cut(a, ":"); ok {
			if p.Manifest.Registry(reg) == nil {
				return nil, fmt.Errorf("unknown registry %q in %q", reg, a)
			}
			t.registry, rest = reg, after
		}
		if rest != "" {
			spec, err := naming.ParseSpec(rest)
			if err != nil {
				return nil, err
			}
			t.spec = spec
		} else if t.registry == "" {
			return nil, errors.New("empty target")
		}
		out = append(out, t)
	}
	return out, nil
}

type resolveOptions struct {
	all     bool
	targets []target
}

func (o resolveOptions) touches(reg string) bool {
	if o.all {
		return true
	}
	for _, t := range o.targets {
		if t.appliesTo(reg) {
			return true
		}
	}
	return false
}

// entryBumped reports whether an install entry is re-resolved at the latest ref.
func (o resolveOptions) entryBumped(reg string, entry naming.Spec) bool {
	if o.all {
		return true
	}
	for _, t := range o.targets {
		if !t.appliesTo(reg) {
			continue
		}
		if t.wholeRegistry() || (t.spec.Namespace == entry.Namespace && (t.spec.IsNamespace() || t.spec.Skill == entry.Skill)) {
			return true
		}
	}
	return false
}

func (o resolveOptions) skillBumped(reg string, skill naming.Spec) bool {
	if o.all {
		return true
	}
	for _, t := range o.targets {
		if t.appliesTo(reg) && (t.wholeRegistry() || t.spec.Covers(skill)) {
			return true
		}
	}
	return false
}

func excludedSet(reg *manifest.Registry) map[string]bool {
	out := map[string]bool{}
	for _, raw := range reg.Exclude {
		if spec, err := naming.ParseSpec(raw); err == nil && !spec.IsNamespace() {
			out[spec.InstalledName()] = true
		}
	}
	return out
}

func installSpecs(reg *manifest.Registry) []naming.Spec {
	var out []naming.Spec
	for _, raw := range reg.Install {
		if spec, err := naming.ParseSpec(raw); err == nil {
			out = append(out, spec)
		}
	}
	return out
}

// lockedSpec returns the skill a lock entry installs. Locks written by mestiere
// 1.0 record the V1Path, which is still where the locked commit keeps the
// skill; the next update relocks them at the current path.
func lockedSpec(name string, e manifest.LockEntry) (naming.Spec, bool) {
	ns, skill, ok := naming.Split(name)
	spec := naming.Spec{Namespace: ns, Skill: skill}
	if !ok || (e.Path != spec.RegistryPath() && e.Path != spec.V1Path()) {
		return naming.Spec{}, false
	}
	return spec, true
}

// prunedLock keeps the lock entries still requested by install and not excluded.
func prunedLock(reg *manifest.Registry) map[string]manifest.LockEntry {
	excluded := excludedSet(reg)
	specs := installSpecs(reg)
	out := map[string]manifest.LockEntry{}
	for name, e := range reg.Lock {
		skill, ok := lockedSpec(name, e)
		if !ok || excluded[name] {
			continue
		}
		for _, s := range specs {
			if s.Covers(skill) {
				out[name] = e
				break
			}
		}
	}
	return out
}

// uncovered lists install entries that have no lock entry yet.
func uncovered(reg *manifest.Registry, lock map[string]manifest.LockEntry) []string {
	excluded := excludedSet(reg)
	var out []string
	for _, spec := range installSpecs(reg) {
		if spec.IsNamespace() {
			// A namespace whose skills are all excluded has nothing to lock;
			// treating it as unresolved would re-resolve it on every sync.
			if len(entriesIn(lock, spec.Namespace)) == 0 && !excludesIn(reg, spec.Namespace) {
				out = append(out, spec.String())
			}
			continue
		}
		if _, ok := lock[spec.InstalledName()]; !ok && !excluded[spec.InstalledName()] {
			out = append(out, spec.String())
		}
	}
	return out
}

func excludesIn(reg *manifest.Registry, ns string) bool {
	for _, raw := range reg.Exclude {
		if spec, err := naming.ParseSpec(raw); err == nil && spec.Namespace == ns {
			return true
		}
	}
	return false
}

func entriesIn(lock map[string]manifest.LockEntry, ns string) map[string]manifest.LockEntry {
	out := map[string]manifest.LockEntry{}
	for name, e := range lock {
		if skill, ok := lockedSpec(name, e); ok && skill.Namespace == ns {
			out[name] = e
		}
	}
	return out
}

// drift describes why install/exclude and lock disagree, if they do.
func drift(reg *manifest.Registry) []string {
	var out []string
	pruned := prunedLock(reg)
	for name, e := range reg.Lock {
		if _, ok := pruned[name]; !ok {
			out = append(out, fmt.Sprintf("registry %s: lock has %s (%s) which install no longer requests", reg.Name, name, e.Path))
		} else if e.Commit == "" || e.Tree == "" {
			out = append(out, fmt.Sprintf("registry %s: lock entry %s is incomplete", reg.Name, name))
		}
	}
	for _, entry := range uncovered(reg, pruned) {
		out = append(out, fmt.Sprintf("registry %s: %s is requested but not locked", reg.Name, entry))
	}
	sort.Strings(out)
	return out
}

func (p *Project) resolve(opts resolveOptions) ([]Change, error) {
	var changes []Change
	for _, reg := range p.Manifest.Registries {
		c, err := p.resolveRegistry(reg, opts)
		if err != nil {
			return nil, fmt.Errorf("registry %s: %w", reg.Name, err)
		}
		changes = append(changes, c...)
	}
	if _, err := p.desired(); err != nil {
		return nil, err
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Name < changes[j].Name })
	return changes, nil
}

func (p *Project) resolveRegistry(reg *manifest.Registry, opts resolveOptions) ([]Change, error) {
	old := reg.Lock
	pruned := prunedLock(reg)
	if !opts.touches(reg.Name) && len(uncovered(reg, pruned)) == 0 {
		reg.Lock = pruned
		return diffLocks(reg.Name, old, pruned), nil
	}
	m, err := p.mirror(reg)
	if err != nil {
		return nil, err
	}
	if err := p.fetch(m); err != nil {
		return nil, err
	}
	commit, err := m.Resolve(reg.Ref)
	if err != nil {
		return nil, err
	}
	catalog, err := m.Catalog(commit)
	if err != nil {
		return nil, err
	}
	at := func(skill naming.Spec) (manifest.LockEntry, error) {
		if !contains(catalog[skill.Namespace], skill.Skill) {
			return manifest.LockEntry{}, registry.ErrNotFound
		}
		content, err := m.Materialize(commit, skill)
		var contentErr *registry.ContentError
		if errors.As(err, &contentErr) {
			return manifest.LockEntry{}, fmt.Errorf("%w at %s (see `mestiere registry lint`)", err, registry.Short(commit))
		}
		if err != nil {
			return manifest.LockEntry{}, err
		}
		return manifest.LockEntry{Path: skill.RegistryPath(), Commit: commit, Tree: content.Tree}, nil
	}
	excluded := excludedSet(reg)
	next := map[string]manifest.LockEntry{}
	for _, spec := range installSpecs(reg) {
		if !spec.IsNamespace() {
			name := spec.InstalledName()
			if excluded[name] {
				continue
			}
			if e, ok := pruned[name]; ok && !opts.entryBumped(reg.Name, spec) {
				next[name] = e
				continue
			}
			e, err := at(spec)
			if errors.Is(err, registry.ErrNotFound) {
				return nil, fmt.Errorf("skill %s no longer exists at %s", spec, registry.Short(commit))
			}
			if err != nil {
				return nil, err
			}
			next[name] = e
			continue
		}
		covered := entriesIn(pruned, spec.Namespace)
		if opts.entryBumped(reg.Name, spec) || len(covered) == 0 {
			skills, ok := catalog[spec.Namespace]
			if !ok {
				return nil, fmt.Errorf("namespace %s doesn't exist at %s", spec.Namespace, registry.Short(commit))
			}
			for _, s := range skills {
				skill := naming.Spec{Namespace: spec.Namespace, Skill: s}
				if excluded[skill.InstalledName()] {
					continue
				}
				e, err := at(skill)
				if err != nil {
					return nil, err
				}
				next[skill.InstalledName()] = e
			}
			continue
		}
		for name, e := range covered {
			skill, _ := lockedSpec(name, e)
			if !opts.skillBumped(reg.Name, skill) {
				next[name] = e
				continue
			}
			bumped, err := at(skill)
			if errors.Is(err, registry.ErrNotFound) {
				continue // namespace installs drop skills deleted upstream
			}
			if err != nil {
				return nil, err
			}
			next[name] = bumped
		}
		for _, t := range opts.targets {
			if !t.appliesTo(reg.Name) || t.wholeRegistry() || t.spec.IsNamespace() || t.spec.Namespace != spec.Namespace {
				continue
			}
			name := t.spec.InstalledName()
			if _, ok := next[name]; ok || excluded[name] {
				continue
			}
			e, err := at(t.spec)
			if errors.Is(err, registry.ErrNotFound) {
				return nil, fmt.Errorf("skill %s doesn't exist at %s", t.spec, registry.Short(commit))
			}
			if err != nil {
				return nil, err
			}
			next[name] = e
		}
	}
	reg.Lock = next
	return diffLocks(reg.Name, old, next), nil
}

// diffLocks reports content changes. A new commit with an identical tree is not a change.
func diffLocks(reg string, old, next map[string]manifest.LockEntry) []Change {
	var out []Change
	for name, e := range next {
		prev, ok := old[name]
		switch {
		case !ok:
			out = append(out, Change{Registry: reg, Name: name, Kind: Added, To: e.Commit})
		case prev.Tree != e.Tree:
			out = append(out, Change{Registry: reg, Name: name, Kind: Updated, From: prev.Commit, To: e.Commit})
		}
	}
	for name, e := range old {
		if _, ok := next[name]; !ok {
			out = append(out, Change{Registry: reg, Name: name, Kind: Removed, From: e.Commit})
		}
	}
	return out
}

type managed struct {
	reg   *manifest.Registry
	name  string
	entry manifest.LockEntry
}

// desired maps every locked installed name to its registry entry.
func (p *Project) desired() (map[string]managed, error) {
	out := map[string]managed{}
	for _, reg := range p.Manifest.Registries {
		for name, e := range reg.Lock {
			if prev, ok := out[name]; ok {
				return nil, fmt.Errorf("skill %s comes from both %s and %s; exclude it from one of them", name, prev.reg.Name, reg.Name)
			}
			out[name] = managed{reg: reg, name: name, entry: e}
		}
	}
	return out, nil
}

func (p *Project) allExcluded() map[string]bool {
	out := map[string]bool{}
	for _, reg := range p.Manifest.Registries {
		for name := range excludedSet(reg) {
			out[name] = true
		}
	}
	return out
}
