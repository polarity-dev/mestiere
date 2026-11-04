package registry

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/gitx"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/skillmd"
)

const (
	modeFile    = "100644"
	modeExec    = "100755"
	modeSymlink = "120000"
	modeGitlink = "160000"
)

// Content is a skill as it is installed: links into the namespace's _shared
// folder are replaced by the files they point to.
type Content struct {
	// Tree is the git tree ID of the installed directory. It equals the
	// skill's own tree ID when the skill links nothing from _shared.
	Tree    string
	Entries []fsx.Entry
	// Legacy is the same version with its README.md, as mestiere 1.0 installed
	// it. It's nil when the skill has no README.md. Locks and installs written
	// by 1.0 still refer to it.
	Legacy *Content
}

// IsReadme reports whether p is a skill's top-level README.md, in any case.
// It documents the skill for people browsing the registry and is never
// installed, so it doesn't reach the agent's context.
func IsReadme(p string) bool { return strings.EqualFold(p, "README.md") }

// ContentError means a skill version can't be installed as published.
type ContentError struct {
	Skill  naming.Spec
	Reason string
}

func (e *ContentError) Error() string { return fmt.Sprintf("%s: %s", e.Skill, e.Reason) }

func contentErr(spec naming.Spec, format string, args ...any) error {
	return &ContentError{Skill: spec, Reason: fmt.Sprintf(format, args...)}
}

type rawEntry struct {
	mode, kind, oid, path string
}

func (m *Mirror) listTree(tree string) ([]rawEntry, error) {
	out, err := gitx.RunInput(m.Dir, nil, "ls-tree", "-r", "-z", tree)
	if err != nil {
		return nil, err
	}
	var entries []rawEntry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("unexpected ls-tree output %q", rec)
		}
		entries = append(entries, rawEntry{mode: fields[0], kind: fields[1], oid: fields[2], path: p})
	}
	return entries, nil
}

// readBlobs returns the content of each object ID.
func (m *Mirror) readBlobs(oids []string) (map[string][]byte, error) {
	out := map[string][]byte{}
	if len(oids) == 0 {
		return out, nil
	}
	var query bytes.Buffer
	for _, oid := range oids {
		query.WriteString(oid + "\n")
	}
	data, err := gitx.RunInput(m.Dir, &query, "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(bytes.NewReader(data))
	for range oids {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading cat-file output: %w", err)
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return nil, fmt.Errorf("object %s: %w", strings.TrimSpace(header), ErrNotFound)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+1) // content plus the trailing newline
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		out[fields[0]] = buf[:size]
	}
	return out, nil
}

func within(p, base string) bool { return p == base || strings.HasPrefix(p, base+"/") }

// Materialize returns the installed form of spec at commit.
func (m *Mirror) Materialize(commit string, spec naming.Spec) (*Content, error) {
	skillTree, err := m.TreeAt(commit, spec.RegistryPath())
	if err != nil {
		return nil, err
	}
	sharedTree, err := m.TreeAt(commit, naming.SharedPath(spec.Namespace))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return m.materialize(spec, skillTree, sharedTree)
}

func (m *Mirror) materialize(spec naming.Spec, skillTree, sharedTree string) (*Content, error) {
	key := [2]string{skillTree, sharedTree}
	if c, ok := m.contents[key]; ok {
		return c, nil
	}
	entries, err := m.listTree(skillTree)
	if err != nil {
		return nil, err
	}
	var links []string
	for _, e := range entries {
		if e.mode == modeSymlink {
			links = append(links, e.oid)
		}
	}
	targets, err := m.readBlobs(links)
	if err != nil {
		return nil, err
	}

	skillPath, sharedPath := spec.RegistryPath(), naming.SharedPath(spec.Namespace)
	var shared map[string]rawEntry
	loadShared := func() error {
		if shared != nil {
			return nil
		}
		shared = map[string]rawEntry{}
		if sharedTree == "" {
			return nil
		}
		list, err := m.listTree(sharedTree)
		if err != nil {
			return err
		}
		for _, e := range list {
			shared[e.path] = e
		}
		return nil
	}

	var out []fsx.Entry
	linked := false
	for _, e := range entries {
		switch e.mode {
		case modeGitlink:
			return nil, contentErr(spec, "%s is a git submodule; submodules can't be installed", e.path)
		case modeFile, modeExec:
			out = append(out, fsx.Entry{Path: e.path, Mode: e.mode, OID: e.oid})
			continue
		case modeSymlink:
		default:
			return nil, contentErr(spec, "%s has unsupported mode %s", e.path, e.mode)
		}
		target := string(targets[e.oid])
		if path.IsAbs(target) {
			return nil, contentErr(spec, "%s links to absolute path %s", e.path, target)
		}
		resolved := path.Join(skillPath, path.Dir(e.path), target)
		switch {
		case within(resolved, skillPath):
			out = append(out, fsx.Entry{Path: e.path, Mode: e.mode, OID: e.oid})
		case within(resolved, sharedPath):
			if err := loadShared(); err != nil {
				return nil, err
			}
			expanded, err := expandShared(spec, e.path, strings.TrimPrefix(strings.TrimPrefix(resolved, sharedPath), "/"), shared)
			if err != nil {
				return nil, err
			}
			out = append(out, expanded...)
			linked = true
		default:
			return nil, contentErr(spec, "%s links outside the skill and %s", e.path, sharedPath)
		}
	}
	for _, e := range out {
		for _, part := range strings.Split(e.Path, "/") {
			if fsx.IsHidden(part) {
				return nil, contentErr(spec, "contains hidden file %s; hidden files can't be installed", e.Path)
			}
		}
	}
	c := &Content{Tree: skillTree, Entries: out}
	if linked {
		if c.Tree, err = fsx.HashEntries(fsx.FormatOf(skillTree), out); err != nil {
			return nil, contentErr(spec, "%v", err)
		}
	}
	var installed []fsx.Entry
	for _, e := range out {
		if !IsReadme(e.Path) {
			installed = append(installed, e)
		}
	}
	if len(installed) < len(out) {
		c = &Content{Entries: installed, Legacy: c}
		if c.Tree, err = fsx.HashEntries(fsx.FormatOf(skillTree), installed); err != nil {
			return nil, contentErr(spec, "%v", err)
		}
	}
	if m.contents == nil {
		m.contents = map[[2]string]*Content{}
	}
	m.contents[key] = c
	return c, nil
}

// expandShared returns the entries a link into _shared stands for: the file
// itself, or every file under a linked directory.
func expandShared(spec naming.Spec, at, rel string, shared map[string]rawEntry) ([]fsx.Entry, error) {
	if e, ok := shared[rel]; ok {
		if e.mode == modeSymlink {
			return nil, contentErr(spec, "%s links to %s/%s, which is itself a link", at, naming.SharedDir, rel)
		}
		return []fsx.Entry{{Path: at, Mode: e.mode, OID: e.oid}}, nil
	}
	var out []fsx.Entry
	for p, e := range shared {
		suffix := p
		if rel != "" {
			var ok bool
			if suffix, ok = strings.CutPrefix(p, rel+"/"); !ok {
				continue
			}
		}
		if e.mode == modeSymlink {
			return nil, contentErr(spec, "%s links to %s/%s, which contains link %s", at, naming.SharedDir, rel, p)
		}
		out = append(out, fsx.Entry{Path: at + "/" + suffix, Mode: e.mode, OID: e.oid})
	}
	if len(out) == 0 {
		return nil, contentErr(spec, "%s links to %s/%s, which doesn't exist", at, naming.SharedDir, rel)
	}
	return out, nil
}

// Descriptions returns the SKILL.md description of each spec at commit.
// Skills whose SKILL.md is missing or unparsable are left out.
func (m *Mirror) Descriptions(commit string, specs []naming.Spec) (map[naming.Spec]string, error) {
	out := map[naming.Spec]string{}
	if len(specs) == 0 {
		return out, nil
	}
	var query bytes.Buffer
	for _, s := range specs {
		fmt.Fprintf(&query, "%s:%s/SKILL.md\n", commit, s.RegistryPath())
	}
	data, err := gitx.RunInput(m.Dir, &query, "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(bytes.NewReader(data))
	for _, s := range specs {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading cat-file output: %w", err)
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			continue // "<query> missing"
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+1)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		if fm, err := skillmd.Parse(buf[:size]); err == nil {
			out[s] = strings.Join(strings.Fields(fm.Description), " ")
		}
	}
	return out, nil
}

// WriteContent writes c into dest, which must not exist.
func (m *Mirror) WriteContent(c *Content, dest string) error {
	var oids []string
	seen := map[string]bool{}
	for _, e := range c.Entries {
		if !seen[e.OID] {
			seen[e.OID] = true
			oids = append(oids, e.OID)
		}
	}
	blobs, err := m.readBlobs(oids)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, e := range c.Entries {
		if err := writeEntry(dest, e, blobs[e.OID]); err != nil {
			return err
		}
	}
	return nil
}

func writeEntry(dest string, e fsx.Entry, data []byte) error {
	name := path.Clean(e.Path)
	if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
		return fmt.Errorf("entry %q escapes the skill directory", e.Path)
	}
	target := filepath.Join(dest, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	switch e.Mode {
	case modeSymlink:
		link := string(data)
		resolved := path.Join(path.Dir(name), link)
		if path.IsAbs(link) || resolved == ".." || strings.HasPrefix(resolved, "../") {
			return fmt.Errorf("symlink %q points outside the skill directory", e.Path)
		}
		return os.Symlink(link, target)
	case modeExec:
		return os.WriteFile(target, data, 0o755)
	default:
		return os.WriteFile(target, data, 0o644)
	}
}

// HistoricalTrees returns the installed tree ID of every version spec has had
// on any branch or tag. A directory whose hash is in this set is identical to
// a published version, which is what makes it safe to replace or delete.
func (m *Mirror) HistoricalTrees(spec naming.Spec) (map[string]bool, error) {
	if trees, ok := m.history[spec]; ok {
		return trees, nil
	}
	skillPath, sharedPath := spec.RegistryPath(), naming.SharedPath(spec.Namespace)
	// --full-history keeps commits on side branches that default history
	// simplification would prune, since a ref may have installed from them.
	out, err := gitx.Run(m.Dir, "rev-list", "--all", "--full-history", "--", skillPath, sharedPath)
	if err != nil {
		return nil, err
	}
	trees := map[string]bool{}
	commits := strings.Fields(out)
	if len(commits) > 0 {
		var query bytes.Buffer
		for _, c := range commits {
			fmt.Fprintf(&query, "%s:%s\n%s:%s\n", c, skillPath, c, sharedPath)
		}
		res, err := gitx.RunInput(m.Dir, &query, "cat-file", "--batch-check")
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.TrimRight(string(res), "\n"), "\n")
		treeOf := func(line string) string {
			if f := strings.Fields(line); len(f) == 3 && f[1] == "tree" {
				return f[0]
			}
			return ""
		}
		seen := map[[2]string]bool{}
		for i := 0; i+1 < len(lines); i += 2 {
			pair := [2]string{treeOf(lines[i]), treeOf(lines[i+1])}
			if pair[0] == "" || seen[pair] {
				continue
			}
			seen[pair] = true
			c, err := m.materialize(spec, pair[0], pair[1])
			var contentErr *ContentError
			if errors.As(err, &contentErr) {
				continue // never installable, so nothing on disk can match it
			}
			if err != nil {
				return nil, err
			}
			trees[c.Tree] = true
			if c.Legacy != nil {
				trees[c.Legacy.Tree] = true
			}
		}
	}
	if m.history == nil {
		m.history = map[naming.Spec]map[string]bool{}
	}
	m.history[spec] = trees
	return trees, nil
}
