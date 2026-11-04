// Package authoring scaffolds and lints registry repositories.
package authoring

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/skillmd"
)

const (
	RegistryFile  = "registry.yaml"
	NamespaceFile = "namespace.yaml"
	SchemaVersion = 1
)

type registryDoc struct {
	SchemaVersion int    `yaml:"schemaVersion"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description,omitempty"`
}

type namespaceDoc struct {
	Description string   `yaml:"description,omitempty"`
	Owners      []string `yaml:"owners,omitempty"`
}

func Init(dir, name, description string) error {
	if name == "" {
		name = filepath.Base(dir)
	}
	p := filepath.Join(dir, RegistryFile)
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("%s already exists", p)
	}
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0o755); err != nil {
		return err
	}
	// git does not track empty directories.
	if err := os.WriteFile(filepath.Join(dir, "skills", ".gitkeep"), nil, 0o644); err != nil {
		return err
	}
	return writeYAML(p, registryDoc{SchemaVersion: SchemaVersion, Name: name, Description: description})
}

func NewNamespace(dir, ns, description string, owners []string) error {
	if err := naming.ValidateNamespace(ns); err != nil {
		return err
	}
	if err := requireRegistry(dir); err != nil {
		return err
	}
	nsDir := filepath.Join(dir, "skills", ns)
	if _, err := os.Stat(nsDir); err == nil {
		return fmt.Errorf("namespace %q already exists", ns)
	}
	if err := os.MkdirAll(nsDir, 0o755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(nsDir, NamespaceFile), namespaceDoc{Description: description, Owners: owners})
}

func NewSkill(dir string, spec naming.Spec, description string) error {
	if spec.IsNamespace() {
		return errors.New("expected <namespace>/<skill>")
	}
	if err := requireRegistry(dir); err != nil {
		return err
	}
	nsDir := filepath.Join(dir, "skills", spec.Namespace)
	if _, err := os.Stat(nsDir); err != nil {
		return fmt.Errorf("namespace %q doesn't exist; create it with `mestiere registry new-namespace %s`", spec.Namespace, spec.Namespace)
	}
	skillDir := filepath.Join(dir, filepath.FromSlash(spec.RegistryPath()))
	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill %s already exists", spec)
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillmd.Template(spec.InstalledName(), description)), 0o644)
}

func requireRegistry(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, RegistryFile)); err != nil {
		return fmt.Errorf("%s isn't a registry (missing %s); run `mestiere registry init`", dir, RegistryFile)
	}
	return nil
}

func writeYAML(p string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Severity Severity
	Path     string
	Message  string
}

func (i Issue) String() string { return fmt.Sprintf("%s: %s: %s", i.Severity, i.Path, i.Message) }

// Lint checks a registry working tree against the rules installs depend on.
func Lint(dir string) ([]Issue, error) {
	l := &linter{root: dir}
	l.registryFile()
	skillsDir := filepath.Join(dir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if errors.Is(err, fs.ErrNotExist) {
		l.add(SeverityError, "skills", "missing skills/ directory")
		return l.issues, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		rel := path.Join("skills", e.Name())
		switch {
		case fsx.IsHidden(e.Name()):
		case !e.IsDir():
			l.add(SeverityWarning, rel, "unexpected file; only namespace directories belong in skills/")
		default:
			l.namespace(e.Name())
		}
	}
	sort.SliceStable(l.issues, func(i, j int) bool { return l.issues[i].Path < l.issues[j].Path })
	return l.issues, nil
}

type linter struct {
	root   string
	issues []Issue
	// used holds the registry paths inside _shared that skills link to.
	used map[string]bool
	// names maps installed names to the folder that claimed them first.
	names map[string]string
}

func (l *linter) add(sev Severity, rel, msg string, args ...any) {
	l.issues = append(l.issues, Issue{Severity: sev, Path: rel, Message: fmt.Sprintf(msg, args...)})
}

func (l *linter) registryFile() {
	data, err := os.ReadFile(filepath.Join(l.root, RegistryFile))
	if err != nil {
		l.add(SeverityError, RegistryFile, "missing registry manifest")
		return
	}
	var doc registryDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		l.add(SeverityError, RegistryFile, "invalid YAML: %v", err)
		return
	}
	if doc.SchemaVersion != SchemaVersion {
		l.add(SeverityError, RegistryFile, "schemaVersion must be %d", SchemaVersion)
	}
	if doc.Name == "" {
		l.add(SeverityError, RegistryFile, "name is required")
	}
}

func (l *linter) namespace(ns string) {
	rel := path.Join("skills", ns)
	if err := naming.ValidateNamespace(ns); err != nil {
		l.add(SeverityError, rel, "%v", err)
		return
	}
	nsDir := filepath.Join(l.root, "skills", ns)
	if data, err := os.ReadFile(filepath.Join(nsDir, NamespaceFile)); err == nil {
		var doc namespaceDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			l.add(SeverityError, path.Join(rel, NamespaceFile), "invalid YAML: %v", err)
		}
	}
	entries, err := os.ReadDir(nsDir)
	if err != nil {
		l.add(SeverityError, rel, "%v", err)
		return
	}
	for _, e := range entries {
		childRel := path.Join(rel, e.Name())
		switch {
		case e.Name() == NamespaceFile, fsx.IsHidden(e.Name()):
		case !e.IsDir():
			l.add(SeverityWarning, childRel, "unexpected file; only skill directories, %s and %s/ belong in a namespace", NamespaceFile, naming.SharedDir)
		case e.Name() == naming.SharedDir:
		default:
			skill, ok := strings.CutPrefix(e.Name(), ns+"-")
			if !ok {
				l.add(SeverityError, childRel, "skill folders are named %s-<skill>, the skill's installed name; rename it to %s-%s", ns, ns, e.Name())
				continue
			}
			spec := naming.Spec{Namespace: ns, Skill: skill}
			if other, ok := l.names[spec.InstalledName()]; ok {
				l.add(SeverityError, childRel, "installs as %s, like %s; rename one of them", spec.InstalledName(), other)
			}
			if l.names == nil {
				l.names = map[string]string{}
			}
			l.names[spec.InstalledName()] = childRel
			l.skill(spec)
		}
	}
	l.shared(ns)
}

// shared checks a namespace's _shared folder and flags files no skill links to.
func (l *linter) shared(ns string) {
	rel := naming.SharedPath(ns)
	dir := filepath.Join(l.root, filepath.FromSlash(rel))
	if info, err := os.Lstat(dir); err != nil {
		return
	} else if !info.IsDir() {
		l.add(SeverityError, rel, "must be a directory")
		return
	}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return err
		}
		sub, _ := filepath.Rel(dir, p)
		entryRel := path.Join(rel, filepath.ToSlash(sub))
		switch {
		case fsx.IsHidden(d.Name()):
			l.add(SeverityError, entryRel, "hidden files aren't installed; remove it from %s", naming.SharedDir)
			if d.IsDir() {
				return filepath.SkipDir
			}
		case d.Type()&fs.ModeSymlink != 0:
			l.add(SeverityError, entryRel, "links aren't allowed inside %s", naming.SharedDir)
		case d.Type().IsRegular() && !l.isUsed(entryRel):
			l.add(SeverityWarning, entryRel, "no skill links to this file")
		}
		return nil
	})
}

func (l *linter) isUsed(rel string) bool {
	for used := range l.used {
		if rel == used || strings.HasPrefix(rel, used+"/") {
			return true
		}
	}
	return false
}

func (l *linter) skill(spec naming.Spec) {
	rel := spec.RegistryPath()
	if err := naming.ValidateSkill(spec.Skill); err != nil {
		l.add(SeverityError, rel, "%v", err)
		return
	}
	if n := len(spec.InstalledName()); n > naming.MaxNameLen {
		l.add(SeverityError, rel, "installed name %q is %d characters; the limit is %d", spec.InstalledName(), n, naming.MaxNameLen)
	}
	skillDir := filepath.Join(l.root, filepath.FromSlash(rel))
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		l.add(SeverityError, rel, "missing SKILL.md")
	} else if fm, err := skillmd.Parse(data); err != nil {
		l.add(SeverityError, path.Join(rel, "SKILL.md"), "%v", err)
	} else {
		if fm.Name != spec.InstalledName() {
			l.add(SeverityError, path.Join(rel, "SKILL.md"), "name must be %q (the folder name), got %q", spec.InstalledName(), fm.Name)
		}
		switch d := strings.TrimSpace(fm.Description); {
		case d == "":
			l.add(SeverityError, path.Join(rel, "SKILL.md"), "description is required")
		case len(d) > skillmd.MaxDescriptionLen:
			l.add(SeverityError, path.Join(rel, "SKILL.md"), "description is %d characters; the limit is %d", len(d), skillmd.MaxDescriptionLen)
		}
	}
	l.contents(spec, skillDir, rel)
}

func (l *linter) contents(spec naming.Spec, skillDir, rel string) {
	_ = filepath.WalkDir(skillDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			l.add(SeverityError, rel, "%v", err)
			return nil
		}
		sub, _ := filepath.Rel(skillDir, p)
		entryRel := path.Join(rel, filepath.ToSlash(sub))
		if p != skillDir && fsx.IsHidden(d.Name()) {
			l.add(SeverityError, entryRel, "hidden files aren't installed; remove it from the skill")
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		target, err := os.Readlink(p)
		if err != nil {
			l.add(SeverityError, entryRel, "%v", err)
			return nil
		}
		if filepath.IsAbs(target) {
			l.add(SeverityError, entryRel, "links must be relative")
			return nil
		}
		l.link(spec, entryRel, path.Join(path.Dir(entryRel), filepath.ToSlash(target)))
		return nil
	})
}

// link checks a symlink at entryRel resolving to the registry path resolved.
// Links may stay inside the skill or point into the namespace's _shared folder,
// where they are replaced by the target files on install.
func (l *linter) link(spec naming.Spec, entryRel, resolved string) {
	sharedRel := naming.SharedPath(spec.Namespace)
	inside := func(base string) bool { return resolved == base || strings.HasPrefix(resolved, base+"/") }
	switch {
	case inside(spec.RegistryPath()):
		if _, err := os.Stat(filepath.Join(l.root, filepath.FromSlash(resolved))); err != nil {
			l.add(SeverityError, entryRel, "broken link to %s", resolved)
		}
	case inside(sharedRel):
		target := filepath.Join(l.root, filepath.FromSlash(resolved))
		info, err := os.Lstat(target)
		switch {
		case err != nil:
			l.add(SeverityError, entryRel, "links to %s, which doesn't exist", resolved)
		case info.Mode()&fs.ModeSymlink != 0:
			l.add(SeverityError, entryRel, "links to %s, which is itself a link", resolved)
		default:
			if l.used == nil {
				l.used = map[string]bool{}
			}
			l.used[resolved] = true
		}
	default:
		l.add(SeverityError, entryRel, "links outside the skill and %s/", sharedRel)
	}
}

// HasErrors reports whether any issue is an error.
func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}
