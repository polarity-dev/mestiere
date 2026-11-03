// Package naming maps registry namespaces and skills to installed names.
//
// A skill at skills/<namespace>/<skill> in a registry installs as <namespace>-<skill>.
// Namespaces cannot contain hyphens, so the first hyphen of an installed name always
// separates namespace from skill and the mapping is reversible.
package naming

import (
	"fmt"
	"regexp"
	"strings"
)

// MaxNameLen is the Agent Skills spec limit for a skill name.
const MaxNameLen = 64

// SharedDir is the namespace folder holding files that several skills link to.
// Skill names can't contain underscores, so it never collides with a skill.
const SharedDir = "_shared"

// SharedPath returns the registry path of a namespace's shared folder.
func SharedPath(ns string) string { return "skills/" + ns + "/" + SharedDir }

var (
	namespaceRe = regexp.MustCompile(`^[a-z0-9]+$`)
	skillRe     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

func ValidateNamespace(ns string) error {
	if !namespaceRe.MatchString(ns) {
		return fmt.Errorf("invalid namespace %q: use lowercase letters and digits only (no hyphens)", ns)
	}
	return nil
}

func ValidateSkill(skill string) error {
	if !skillRe.MatchString(skill) {
		return fmt.Errorf("invalid skill name %q: use lowercase letters, digits and single hyphens", skill)
	}
	return nil
}

// Spec identifies a whole namespace (Skill empty) or a single skill.
type Spec struct {
	Namespace string
	Skill     string
}

func (s Spec) IsNamespace() bool { return s.Skill == "" }

// String returns the manifest form: "ns" or "ns/skill".
func (s Spec) String() string {
	if s.Skill == "" {
		return s.Namespace
	}
	return s.Namespace + "/" + s.Skill
}

func (s Spec) InstalledName() string { return s.Namespace + "-" + s.Skill }

func (s Spec) RegistryPath() string {
	if s.Skill == "" {
		return "skills/" + s.Namespace
	}
	return "skills/" + s.Namespace + "/" + s.Skill
}

// Covers reports whether the spec includes the given single-skill spec.
func (s Spec) Covers(other Spec) bool {
	if s.Namespace != other.Namespace {
		return false
	}
	return s.IsNamespace() || s.Skill == other.Skill
}

func (s Spec) validate() error {
	if err := ValidateNamespace(s.Namespace); err != nil {
		return err
	}
	if s.Skill == "" {
		return nil
	}
	if err := ValidateSkill(s.Skill); err != nil {
		return err
	}
	if n := len(s.InstalledName()); n > MaxNameLen {
		return fmt.Errorf("installed name %q is %d characters; the limit is %d", s.InstalledName(), n, MaxNameLen)
	}
	return nil
}

// ParseSpec accepts "ns", "ns/skill" or an installed name "ns-skill".
func ParseSpec(raw string) (Spec, error) {
	s := strings.TrimSuffix(strings.TrimSpace(raw), "/")
	var spec Spec
	switch {
	case strings.Contains(s, "/"):
		ns, skill, _ := strings.Cut(s, "/")
		spec = Spec{Namespace: ns, Skill: skill}
	case strings.Contains(s, "-"):
		ns, skill, ok := Split(s)
		if !ok {
			return Spec{}, fmt.Errorf("invalid skill %q: expected <namespace>-<skill>", raw)
		}
		spec = Spec{Namespace: ns, Skill: skill}
	default:
		spec = Spec{Namespace: s}
	}
	if err := spec.validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// ParsePath maps a registry path skills/<ns>/<skill> to its spec.
func ParsePath(path string) (Spec, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "skills" {
		return Spec{}, false
	}
	spec := Spec{Namespace: parts[1], Skill: parts[2]}
	if spec.validate() != nil {
		return Spec{}, false
	}
	return spec, true
}

// Split maps an installed name back to its namespace and skill.
func Split(installed string) (ns, skill string, ok bool) {
	ns, skill, found := strings.Cut(installed, "-")
	if !found || ValidateNamespace(ns) != nil || ValidateSkill(skill) != nil {
		return "", "", false
	}
	return ns, skill, true
}
