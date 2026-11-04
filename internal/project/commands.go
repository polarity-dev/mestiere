package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/gitignore"
	"github.com/polarity-dev/mestiere/internal/manifest"
	"github.com/polarity-dev/mestiere/internal/naming"
)

func indexSpec(list []string, spec naming.Spec) int {
	for i, raw := range list {
		if s, err := naming.ParseSpec(raw); err == nil && s == spec {
			return i
		}
	}
	return -1
}

func hasSpec(list []string, spec naming.Spec) bool { return indexSpec(list, spec) >= 0 }

func withoutSpec(list []string, drop func(naming.Spec) bool) []string {
	return without(list, func(raw string) bool {
		s, err := naming.ParseSpec(raw)
		return err == nil && drop(s)
	})
}

var errGitOnly = errors.New("global skills aren't in a git repository; track and untrack only work in a project")

// Add requests namespaces or skills from a registry and installs them.
func (p *Project) Add(registryName string, args []string, force bool) (*Report, error) {
	if len(args) == 0 {
		return nil, errors.New("nothing to add")
	}
	reg, err := p.pickRegistry(registryName)
	if err != nil {
		return nil, err
	}
	r := &Report{}
	var reincluded []target
	var catalog map[string][]string
	var catalogErr error
	offered := func(c naming.Spec) bool {
		if catalog == nil && catalogErr == nil {
			catalog, catalogErr = p.Catalog(reg.Name)
		}
		_, hasNS := catalog[c.Namespace]
		return hasNS && (c.IsNamespace() || contains(catalog[c.Namespace], c.Skill))
	}
	for _, a := range args {
		spec, err := pickSpec(a, "in registry "+reg.Name, offered)
		if catalogErr != nil {
			return nil, catalogErr
		}
		if err != nil {
			return nil, err
		}
		nsSpec := naming.Spec{Namespace: spec.Namespace}
		if spec.IsNamespace() {
			if hasSpec(reg.Install, spec) {
				r.Warn("%s is already installed", spec)
				continue
			}
			reg.Install = withoutSpec(reg.Install, spec.Covers)
			reg.Install = append(reg.Install, spec.String())
			continue
		}
		wasExcluded := hasSpec(reg.Exclude, spec)
		reg.Exclude = withoutSpec(reg.Exclude, func(s naming.Spec) bool { return s == spec })
		switch {
		case hasSpec(reg.Install, spec), hasSpec(reg.Install, nsSpec):
			if wasExcluded {
				reincluded = append(reincluded, target{registry: reg.Name, spec: spec, raw: a})
			} else {
				r.Warn("%s is already installed", spec)
			}
		default:
			reg.Install = append(reg.Install, spec.String())
		}
	}
	changes, err := p.resolve(resolveOptions{targets: reincluded})
	if err != nil {
		return nil, err
	}
	if err := p.checkAddConflicts(changes, force); err != nil {
		return nil, err
	}
	if err := p.Save(); err != nil {
		return nil, err
	}
	for _, c := range changes {
		r.Action("lock %s", c)
	}
	sr, err := p.Sync(SyncOptions{Force: force})
	r.Merge(sr)
	return r, err
}

// checkAddConflicts refuses to install over a directory that isn't a published
// version of the incoming skill, before anything is written.
func (p *Project) checkAddConflicts(changes []Change, force bool) error {
	if force {
		return nil
	}
	desired, err := p.desired()
	if err != nil {
		return err
	}
	s := p.newSyncer(SyncOptions{}, &Report{}, desired)
	for _, c := range changes {
		if c.Kind != Added {
			continue
		}
		m := desired[c.Name]
		dir := filepath.Join(p.canonicalDir(), c.Name)
		info, err := os.Lstat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err == nil && info.IsDir() {
			if st := s.classify(dir, m); st == stateCurrent || st == statePublished {
				continue
			}
		}
		return fmt.Errorf("%s already exists and isn't a published version of %s; rename it or use --force", p.rel(dir), m.entry.Path)
	}
	return nil
}

// Remove drops namespaces or skills from install and deletes their files when
// they are unmodified. Skills installed through a namespace are excluded.
func (p *Project) Remove(args []string, force bool) (*Report, error) {
	targets, err := p.parseTargets(args)
	if err != nil {
		return nil, err
	}
	before, err := p.desired()
	if err != nil {
		return nil, err
	}
	for _, t := range targets {
		if t.wholeRegistry() {
			return nil, fmt.Errorf("%s names a whole registry; use `mestiere registry remove`", t.raw)
		}
		matched := false
		for _, reg := range p.Manifest.Registries {
			if !t.appliesTo(reg.Name) {
				continue
			}
			n := len(reg.Install)
			reg.Install = withoutSpec(reg.Install, t.spec.Covers)
			if len(reg.Install) < n {
				matched = true
			}
			if !t.spec.IsNamespace() && hasSpec(reg.Install, naming.Spec{Namespace: t.spec.Namespace}) {
				if !hasSpec(reg.Exclude, t.spec) {
					reg.Exclude = append(reg.Exclude, t.spec.String())
				}
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("%s isn't installed", t.raw)
		}
	}
	for _, reg := range p.Manifest.Registries {
		reg.Lock = prunedLock(reg)
	}
	after, err := p.desired()
	if err != nil {
		return nil, err
	}
	var gone []managed
	for name, m := range before {
		if _, ok := after[name]; !ok {
			gone = append(gone, m)
		}
	}
	r := &Report{}
	p.uninstall(gone, r, force)
	if err := p.Save(); err != nil {
		return r, err
	}
	sr, err := p.Sync(SyncOptions{})
	r.Merge(sr)
	return r, err
}

// uninstall deletes the files of skills that left the lock. Modified skills
// are kept, along with the links that point to them.
func (p *Project) uninstall(items []managed, r *Report, force bool) {
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	s := p.newSyncer(SyncOptions{Force: force}, r, nil)
	for _, m := range items {
		dir := filepath.Join(p.canonicalDir(), m.name)
		kept := false
		if info, err := os.Lstat(dir); err == nil && info.IsDir() {
			if st := s.classify(dir, m); st == stateCurrent || st == statePublished || force {
				s.do("remove "+p.rel(dir), func() error { return os.RemoveAll(dir) })
			} else {
				kept = true
				r.Warn("kept %s because it has local changes; it's now a local skill", p.rel(dir))
			}
		}
		for _, a := range agents.Harnesses() {
			dst := filepath.Join(p.agentDir(a), m.name)
			info, err := os.Lstat(dst)
			if err != nil {
				continue
			}
			remove := func() error { return os.RemoveAll(dst) }
			switch {
			case info.Mode()&fs.ModeSymlink != 0:
				if s.isStoreLink(dst) && !kept {
					s.do("unlink "+p.rel(dst), remove)
				}
			case info.IsDir():
				if st := s.classify(dst, m); st == stateCurrent || st == statePublished || force {
					s.do("remove "+p.rel(dst), remove)
				} else {
					r.Warn("kept %s because it has local changes", p.rel(dst))
				}
			}
		}
	}
}

// Update moves the lock to the latest ref for the targets, or for everything.
func (p *Project) Update(args []string, force bool) ([]Change, *Report, error) {
	targets, err := p.parseTargets(args)
	if err != nil {
		return nil, nil, err
	}
	for _, t := range targets {
		if !p.targetInstalled(t) {
			return nil, nil, fmt.Errorf("%s isn't installed", t.raw)
		}
	}
	changes, err := p.resolve(resolveOptions{all: len(targets) == 0, targets: targets})
	if err != nil {
		return nil, nil, err
	}
	if err := p.Save(); err != nil {
		return nil, nil, err
	}
	r, err := p.Sync(SyncOptions{Force: force})
	return changes, r, err
}

func (p *Project) targetInstalled(t target) bool {
	for _, reg := range p.Manifest.Registries {
		if !t.appliesTo(reg.Name) {
			continue
		}
		if t.wholeRegistry() {
			return true
		}
		for _, spec := range installSpecs(reg) {
			if spec.Covers(t.spec) || t.spec.Covers(spec) {
				return true
			}
		}
	}
	return false
}

// Outdated reports what Update would change, without writing anything.
func (p *Project) Outdated() ([]Change, error) {
	saved := map[*manifest.Registry]map[string]manifest.LockEntry{}
	for _, reg := range p.Manifest.Registries {
		saved[reg] = reg.Lock
	}
	defer func() {
		for reg, lock := range saved {
			reg.Lock = lock
		}
	}()
	return p.resolve(resolveOptions{all: true})
}

// Eject turns installed skills into local skills: the files stay, and
// mestiere stops managing them.
func (p *Project) Eject(args []string, track bool) (*Report, error) {
	r := &Report{}
	var names []string
	for _, a := range args {
		spec, err := p.installedSpec("", a)
		if err != nil {
			return nil, err
		}
		if spec.IsNamespace() {
			return nil, fmt.Errorf("%s is a namespace; eject skills one at a time", a)
		}
		name := spec.InstalledName()
		var owner *manifest.Registry
		for _, reg := range p.Manifest.Registries {
			if _, ok := reg.Lock[name]; ok {
				owner = reg
			}
		}
		if owner == nil {
			return nil, fmt.Errorf("%s isn't installed from a registry", name)
		}
		delete(owner.Lock, name)
		owner.Install = withoutSpec(owner.Install, func(s naming.Spec) bool { return s == spec })
		if !hasSpec(owner.Exclude, spec) {
			owner.Exclude = append(owner.Exclude, spec.String())
		}
		r.Action("eject %s; it's now a local skill", name)
		names = append(names, name)
	}
	if track && p.Global {
		return nil, errGitOnly
	}
	if err := p.Save(); err != nil {
		return r, err
	}
	if track {
		tr, err := p.Track(names)
		r.Merge(tr)
		return r, err
	}
	if p.Global {
		return r, nil
	}
	r.Hint("ejected skills are still ignored by git; commit them with `mestiere track %s`", strings.Join(names, " "))
	return r, nil
}

func localName(arg string) (string, error) {
	if spec, err := naming.ParseSpec(arg); err == nil && !spec.IsNamespace() && strings.Contains(arg, "/") {
		return spec.InstalledName(), nil
	}
	if arg == "" || strings.ContainsAny(arg, `/\`) || strings.HasPrefix(arg, ".") {
		return "", fmt.Errorf("invalid skill name %q", arg)
	}
	return arg, nil
}

// Track adds .gitignore exceptions so a skill is committed from every
// skills directory it exists in.
func (p *Project) Track(args []string) (*Report, error) {
	if p.Global {
		return nil, errGitOnly
	}
	r := &Report{}
	for _, a := range args {
		name, err := localName(a)
		if err != nil {
			return r, err
		}
		var rels []string
		for _, ag := range agents.Known() {
			if _, err := os.Lstat(filepath.Join(p.agentDir(ag), name)); err == nil {
				rels = append(rels, path.Join(ag.Dir, name))
			}
		}
		if len(rels) == 0 {
			return r, fmt.Errorf("%s doesn't exist in any skills directory", name)
		}
		added, warnings, err := gitignore.Track(p.Root, rels)
		if err != nil {
			return r, err
		}
		for _, line := range added {
			r.Action("add %q to .gitignore", line)
		}
		if len(added) == 0 {
			r.Warn("%s is already tracked", name)
		}
		r.Warnings = append(r.Warnings, warnings...)
	}
	return r, nil
}

func (p *Project) Untrack(args []string) (*Report, error) {
	if p.Global {
		return nil, errGitOnly
	}
	r := &Report{}
	for _, a := range args {
		name, err := localName(a)
		if err != nil {
			return r, err
		}
		var rels []string
		for _, ag := range agents.Known() {
			rels = append(rels, path.Join(ag.Dir, name))
		}
		removed, err := gitignore.Untrack(p.Root, rels)
		if err != nil {
			return r, err
		}
		for _, line := range removed {
			r.Action("remove %q from .gitignore", line)
		}
		if len(removed) == 0 {
			r.Warn("%s has no exception in .gitignore", name)
		} else if p.inRepo() {
			r.Hint("files git already tracks stay tracked; run `git rm -r --cached` on them to stop committing %s", name)
		}
	}
	return r, nil
}

// RemoveRegistry uninstalls a registry's skills and drops it from skills.yaml.
func (p *Project) RemoveRegistry(name string, force bool) (*Report, error) {
	reg := p.Manifest.Registry(name)
	if reg == nil {
		return nil, fmt.Errorf("unknown registry %q", name)
	}
	r := &Report{}
	var items []managed
	for skill, e := range reg.Lock {
		items = append(items, managed{reg: reg, name: skill, entry: e})
	}
	p.uninstall(items, r, force)
	p.Manifest.RemoveRegistry(name)
	if err := p.Save(); err != nil {
		return r, err
	}
	r.Action("remove registry %s", name)
	sr, err := p.Sync(SyncOptions{})
	r.Merge(sr)
	return r, err
}
