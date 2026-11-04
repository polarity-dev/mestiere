// Package naming maps registry namespaces and skills to installed names.
//
// A skill at skills/<namespace>/<namespace>-<skill> in a registry installs as
// <namespace>-<skill>. Namespaces and skills may both contain hyphens, so an
// installed name alone doesn't say where the namespace ends: code that starts
// from one uses the lock path, or Candidates checked against what exists.
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

var nameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func ValidateNamespace(ns string) error {
	if !nameRe.MatchString(ns) {
		return fmt.Errorf("invalid namespace %q: use lowercase letters, digits and single hyphens", ns)
	}
	return nil
}

func ValidateSkill(skill string) error {
	if !nameRe.MatchString(skill) {
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

// RegistryPath is skills/<ns> or skills/<ns>/<ns>-<skill>. The skill folder
// carries the prefix because the Agent Skills spec requires a skill's folder
// to match the name in its SKILL.md.
func (s Spec) RegistryPath() string {
	if s.Skill == "" {
		return "skills/" + s.Namespace
	}
	return "skills/" + s.Namespace + "/" + s.InstalledName()
}

// V1Path is skills/<ns>/<skill>, where mestiere 1.0 kept a skill. Registries
// can't use it any more, but 1.0 locks and installs still come from it.
func (s Spec) V1Path() string { return "skills/" + s.Namespace + "/" + s.Skill }

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

// ParseSpec accepts the manifest forms "ns" and "ns/skill".
func ParseSpec(raw string) (Spec, error) {
	s := strings.TrimSuffix(strings.TrimSpace(raw), "/")
	ns, skill, _ := strings.Cut(s, "/")
	spec := Spec{Namespace: ns, Skill: skill}
	if strings.Contains(s, "/") && skill == "" {
		return Spec{}, fmt.Errorf("invalid skill %q: expected <namespace>/<skill>", raw)
	}
	if err := spec.validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// Candidates lists what a command-line argument can mean. "ns/skill" and "ns/"
// mean one thing; a name without "/" can be a namespace or, split at any
// hyphen, the installed name of a skill. The namespace reading comes first.
func Candidates(raw string) ([]Spec, error) {
	s := strings.TrimSpace(raw)
	if strings.Contains(s, "/") {
		spec, err := ParseSpec(s)
		if err != nil {
			return nil, err
		}
		return []Spec{spec}, nil
	}
	var out []Spec
	if ValidateNamespace(s) == nil {
		out = append(out, Spec{Namespace: s})
	}
	for i, c := range s {
		if c != '-' {
			continue
		}
		if spec := (Spec{Namespace: s[:i], Skill: s[i+1:]}); spec.validate() == nil {
			out = append(out, spec)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("invalid name %q: use <namespace>, <namespace>/<skill> or an installed name", raw)
	}
	return out, nil
}

// ParsePath maps a registry path to its spec: skills/<ns>/<ns>-<skill>, or
// the 1.0 skills/<ns>/<skill> when v1 is set.
func ParsePath(path string, v1 bool) (Spec, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "skills" {
		return Spec{}, false
	}
	skill, ok := strings.CutPrefix(parts[2], parts[1]+"-")
	if v1 {
		skill, ok = parts[2], true
	}
	spec := Spec{Namespace: parts[1], Skill: skill}
	if !ok || spec.validate() != nil {
		return Spec{}, false
	}
	return spec, true
}
