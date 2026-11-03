package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/manifest"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/registry"
)

type SyncOptions struct {
	// Frozen fails instead of re-resolving when install and lock disagree.
	Frozen bool
	// Force overwrites content that can't be proven to be a published registry version.
	Force bool
	// DryRun reports actions without performing them.
	DryRun bool
}

// contentState classifies a directory against the registry.
type contentState int

const (
	// stateCurrent matches the locked tree.
	stateCurrent contentState = iota
	// statePublished matches some other published version of the same skill,
	// so it can be replaced or removed without losing anything.
	statePublished
	// stateForeign matches no published version: local edits or a user skill.
	stateForeign
	// stateUnverifiable can't be checked because the registry cache is unavailable.
	stateUnverifiable
)

// Sync makes the skill directories match the lock. Lock is the only state:
// whether a directory may be replaced or deleted is decided by comparing its
// content with the registry's history, never by remembering past installs.
func (p *Project) Sync(opts SyncOptions) (*Report, error) {
	r := &Report{}
	var issues []string
	for _, reg := range p.Manifest.Registries {
		issues = append(issues, drift(reg)...)
	}
	if len(issues) > 0 {
		switch {
		case opts.Frozen:
			return r, fmt.Errorf("%s install and lock disagree; run `mestiere sync` and commit the result:\n  %s",
				manifest.FileName, strings.Join(issues, "\n  "))
		case opts.DryRun:
			for _, issue := range issues {
				r.Problem("%s", issue)
			}
		default:
			changes, err := p.resolve(resolveOptions{})
			if err != nil {
				return r, err
			}
			if err := p.Save(); err != nil {
				return r, err
			}
			for _, c := range changes {
				r.Action("lock %s", c)
			}
		}
	}
	desired, err := p.desired()
	if err != nil {
		return r, err
	}
	p.newSyncer(opts, r, desired).run()
	return r, nil
}

type syncer struct {
	p        *Project
	opts     SyncOptions
	r        *Report
	desired  map[string]managed
	excluded map[string]bool
	failed   map[string]bool
	hashes   map[string]string
	symlink  *bool
}

func (p *Project) newSyncer(opts SyncOptions, r *Report, desired map[string]managed) *syncer {
	return &syncer{
		p: p, opts: opts, r: r, desired: desired,
		excluded: p.allExcluded(),
		failed:   map[string]bool{},
		hashes:   map[string]string{},
	}
}

func (s *syncer) do(desc string, fn func() error) bool {
	if s.opts.DryRun {
		s.r.Action("%s", desc)
		return true
	}
	if err := fn(); err != nil {
		s.r.Problem("%s: %v", desc, err)
		return false
	}
	s.r.Action("%s", desc)
	return true
}

func (s *syncer) run() {
	names := make([]string, 0, len(s.desired))
	for name := range s.desired {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s.ensureCanonical(s.desired[name])
	}
	configured := map[string]bool{}
	for _, a := range s.p.harnesses() {
		configured[a.ID] = true
		for _, name := range names {
			s.ensureHarness(a, s.desired[name])
		}
	}
	s.cleanCanonical()
	for _, a := range agents.Harnesses() {
		s.cleanHarness(a, configured[a.ID])
	}
	if !s.opts.DryRun {
		s.removeStaging()
	}
}

func (s *syncer) ensureCanonical(m managed) {
	dir := filepath.Join(s.p.canonicalDir(), m.name)
	rel := s.p.rel(dir)
	install := func() error { return s.install(m, dir) }
	info, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		s.track(m.name, s.do(fmt.Sprintf("install %s at %s", m.name, registry.Short(m.entry.Commit)), install))
	case err != nil:
		s.fail(m.name, "%s: %v", rel, err)
	case !info.IsDir():
		if s.opts.Force {
			s.track(m.name, s.do("replace "+rel, install))
		} else {
			s.fail(m.name, "%s exists but isn't a directory; remove it or use --force", rel)
		}
	default:
		switch s.classify(dir, m) {
		case stateCurrent:
		case statePublished:
			s.track(m.name, s.do(fmt.Sprintf("update %s to %s", m.name, registry.Short(m.entry.Commit)), install))
		case stateUnverifiable:
			if s.opts.Force {
				s.track(m.name, s.do("overwrite "+rel, install))
			} else {
				s.fail(m.name, "%s differs from the lock and the registry cache isn't available to check it; use --force to overwrite", rel)
			}
		case stateForeign:
			if s.opts.Force {
				s.track(m.name, s.do("overwrite local changes in "+rel, install))
			} else {
				s.fail(m.name, "%s has local changes; use --force to overwrite them or `mestiere eject %s` to keep them", rel, m.name)
			}
		}
	}
}

func (s *syncer) track(name string, ok bool) {
	if !ok {
		s.failed[name] = true
	}
}

func (s *syncer) fail(name, format string, args ...any) {
	s.failed[name] = true
	s.r.Problem(format, args...)
}

func (s *syncer) install(m managed, dir string) error {
	mirror, err := s.p.mirror(m.reg)
	if err != nil {
		return err
	}
	if !mirror.HasObject(m.entry.Commit) {
		if err := s.p.fetch(mirror); err != nil {
			return err
		}
		if !mirror.HasObject(m.entry.Commit) {
			return fmt.Errorf("commit %s isn't in registry %s; it may have been removed by a force push (run `mestiere update %s`)",
				registry.Short(m.entry.Commit), m.reg.Name, m.name)
		}
	}
	spec, ok := naming.ParsePath(m.entry.Path)
	if !ok || spec.InstalledName() != m.name {
		return fmt.Errorf("lock entry path %q doesn't match %s", m.entry.Path, m.name)
	}
	content, err := mirror.Materialize(m.entry.Commit, spec)
	if err != nil {
		return err
	}
	if content.Tree != m.entry.Tree {
		return fmt.Errorf("%s at %s doesn't match the locked tree %s; the lock was edited or the registry rewritten (run `mestiere update %s`)",
			m.entry.Path, registry.Short(m.entry.Commit), registry.Short(m.entry.Tree), m.name)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	staged, err := fsx.TempSibling(dir)
	if err != nil {
		return err
	}
	if err := mirror.WriteContent(content, staged); err != nil {
		os.RemoveAll(staged)
		return err
	}
	s.forget(dir)
	return fsx.ReplaceDir(staged, dir)
}

func (s *syncer) forget(dir string) {
	for key := range s.hashes {
		if strings.HasSuffix(key, ":"+dir) {
			delete(s.hashes, key)
		}
	}
}

func (s *syncer) ensureHarness(a agents.Agent, m managed) {
	if s.failed[m.name] {
		return
	}
	dst := filepath.Join(s.p.agentDir(a), m.name)
	src := filepath.Join(s.p.canonicalDir(), m.name)
	rel := s.p.rel(dst)
	place := func() error { return s.place(src, dst) }
	info, err := os.Lstat(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		s.do("link "+rel, place)
	case err != nil:
		s.r.Problem("%s: %v", rel, err)
	case info.Mode()&fs.ModeSymlink != 0:
		if !s.isStoreLink(dst) && !isBroken(dst) && !s.opts.Force {
			s.r.Problem("%s is a symlink to somewhere else; remove it or use --force", rel)
			return
		}
		if s.mode() == LinkSymlink && readlink(dst) == linkTarget(src, dst) {
			return
		}
		s.do("relink "+rel, place)
	case info.IsDir():
		st := s.classify(dst, m)
		if st == stateCurrent && s.mode() == LinkCopy {
			return
		}
		if st == stateCurrent || st == statePublished || s.opts.Force {
			s.do("refresh "+rel, place)
			return
		}
		s.r.Problem("%s has local changes; edit %s instead, or use --force to overwrite", rel, s.p.rel(src))
	default:
		if s.opts.Force {
			s.do("replace "+rel, place)
		} else {
			s.r.Problem("%s exists but isn't a directory; remove it or use --force", rel)
		}
	}
}

func (s *syncer) place(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	staged, err := fsx.TempSibling(dst)
	if err != nil {
		return err
	}
	if s.mode() == LinkSymlink {
		err = os.Symlink(linkTarget(src, dst), staged)
	} else {
		err = fsx.CopyDir(src, staged)
	}
	if err != nil {
		os.RemoveAll(staged)
		return err
	}
	s.forget(dst)
	return fsx.ReplaceDir(staged, dst)
}

// mode returns how harness entries are materialized. Without an explicit link
// setting, platforms that can't create symlinks (Windows without developer
// mode) fall back to copies.
func (s *syncer) mode() string {
	if s.p.Manifest.Link != "" {
		return s.p.Manifest.Link
	}
	if s.symlink == nil {
		ok := symlinksSupported()
		s.symlink = &ok
		if !ok {
			s.r.Warn("symlinks aren't supported here; copying skills into agent directories (set `link: copy` in %s to make this explicit)", manifest.FileName)
		}
	}
	if *s.symlink {
		return LinkSymlink
	}
	return LinkCopy
}

func symlinksSupported() bool {
	dir, err := os.MkdirTemp("", "mestiere-probe-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	return os.Symlink("target", filepath.Join(dir, "link")) == nil
}

func (s *syncer) cleanCanonical() {
	dir := s.p.canonicalDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || s.excluded[name] {
			continue
		}
		if _, ok := s.desired[name]; ok {
			continue
		}
		p := filepath.Join(dir, name)
		if info, err := os.Lstat(p); err != nil || !info.IsDir() {
			continue
		}
		switch s.publishedByName(name, p) {
		case statePublished:
			s.do(fmt.Sprintf("remove %s (no longer in %s)", s.p.rel(p), manifest.FileName), func() error { return os.RemoveAll(p) })
		case stateUnverifiable:
			s.r.Warn("can't check %s against its registry; leaving it in place", s.p.rel(p))
		}
	}
}

func (s *syncer) cleanHarness(a agents.Agent, configured bool) {
	dir := s.p.agentDir(a)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || s.excluded[name] {
			continue
		}
		m, isDesired := s.desired[name]
		if isDesired && configured {
			continue
		}
		p := filepath.Join(dir, name)
		rel := s.p.rel(p)
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		remove := func() error { return os.RemoveAll(p) }
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			if !s.isStoreLink(p) {
				continue
			}
			if isDesired {
				s.do(fmt.Sprintf("unlink %s (agent %s isn't configured)", rel, a.ID), remove)
			} else if isBroken(p) {
				s.do("remove broken link "+rel, remove)
			}
		case info.IsDir():
			var st contentState
			if isDesired {
				st = s.classify(p, m)
			} else {
				st = s.publishedByName(name, p)
			}
			if st == stateCurrent || st == statePublished {
				s.do("remove "+rel, remove)
			}
		}
	}
}

func (s *syncer) removeStaging() {
	for _, a := range agents.Known() {
		entries, err := os.ReadDir(s.p.agentDir(a))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), fsx.TempPrefix) {
				os.RemoveAll(filepath.Join(s.p.agentDir(a), e.Name()))
			}
		}
	}
}

func (s *syncer) hash(dir string, format fsx.ObjectFormat) (string, error) {
	key := string(format) + ":" + dir
	if h, ok := s.hashes[key]; ok {
		return h, nil
	}
	h, err := fsx.TreeHash(dir, format)
	if err != nil {
		return "", err
	}
	s.hashes[key] = h
	return h, nil
}

func (s *syncer) classify(dir string, m managed) contentState {
	h, err := s.hash(dir, fsx.FormatOf(m.entry.Tree))
	if err != nil {
		return stateForeign
	}
	if h == m.entry.Tree {
		return stateCurrent
	}
	spec, ok := naming.ParsePath(m.entry.Path)
	if !ok {
		return stateForeign
	}
	return s.published(m.reg, spec, dir)
}

// published checks dir against every version spec has had in the registry.
//
// The mirror is fetched only when it doesn't exist: every directory this
// machine installed came through the mirror, so its history already contains
// those versions.
func (s *syncer) published(reg *manifest.Registry, spec naming.Spec, dir string) contentState {
	mirror, err := s.p.mirror(reg)
	if err != nil {
		return stateUnverifiable
	}
	if !mirror.Exists() {
		if err := s.p.fetch(mirror); err != nil {
			return stateUnverifiable
		}
	}
	trees, err := mirror.HistoricalTrees(spec)
	if err != nil {
		return stateUnverifiable
	}
	h, err := s.hash(dir, mirror.ObjectFormat())
	if err != nil {
		return stateForeign
	}
	if trees[h] {
		return statePublished
	}
	return stateForeign
}

// publishedByName checks a directory that isn't locked. Both its name and its
// content must match the same registry skill, so copies under other names are
// never treated as installs.
func (s *syncer) publishedByName(name, dir string) contentState {
	ns, skill, ok := naming.Split(name)
	if !ok {
		return stateForeign
	}
	spec := naming.Spec{Namespace: ns, Skill: skill}
	result := stateForeign
	for _, reg := range s.p.Manifest.Registries {
		switch s.published(reg, spec, dir) {
		case statePublished:
			return statePublished
		case stateUnverifiable:
			result = stateUnverifiable
		}
	}
	return result
}

func (s *syncer) isStoreLink(p string) bool {
	target := readlink(p)
	if target == "" {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(p), target)
	}
	store := s.p.canonicalDir() + string(filepath.Separator)
	return strings.HasPrefix(filepath.Clean(target)+string(filepath.Separator), store) && filepath.Clean(target) != filepath.Clean(s.p.canonicalDir())
}

func readlink(p string) string {
	t, err := os.Readlink(p)
	if err != nil {
		return ""
	}
	return t
}

func isBroken(p string) bool {
	_, err := os.Stat(p)
	return err != nil
}

// linkTarget is relative so links keep working when the repository moves.
func linkTarget(src, dst string) string {
	rel, err := filepath.Rel(filepath.Dir(dst), src)
	if err != nil {
		return src
	}
	return rel
}
