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

// Spec identifies a whole namespace (Skill empty) or a single skill. A skill
// with no namespace is a plain skill from a registry with the vercel/skills
// layout, stored at skills/<skill> and installed under its own name.
type Spec struct {
	Namespace string
	Skill     string
}

func (s Spec) IsNamespace() bool { return s.Skill == "" }

// IsPlain reports whether s is a skill without a namespace.
func (s Spec) IsPlain() bool { return s.Namespace == "" && s.Skill != "" }

// String returns the manifest form: "ns", "ns/skill" or a plain "skill".
func (s Spec) String() string {
	switch {
	case s.Skill == "":
		return s.Namespace
	case s.Namespace == "":
		return s.Skill
	}
	return s.Namespace + "/" + s.Skill
}

func (s Spec) InstalledName() string {
	if s.Namespace == "" {
		return s.Skill
	}
	return s.Namespace + "-" + s.Skill
}

// RegistryPath is skills/<ns> or skills/<ns>/<ns>-<skill>. The skill folder
// carries the prefix because the Agent Skills spec requires a skill's folder
// to match the name in its SKILL.md.
func (s Spec) RegistryPath() string {
	if s.IsPlain() {
		return "skills/" + s.Skill
	}
	if s.Skill == "" {
		return "skills/" + s.Namespace
	}
	return "skills/" + s.Namespace + "/" + s.InstalledName()
}

// V1Path is skills/<ns>/<skill>, where mestiere 1.0 kept a skill. Registries
// can't use it any more, but 1.0 locks and installs still come from it.
func (s Spec) V1Path() string {
	if s.IsPlain() {
		return s.RegistryPath()
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

// validate checks a namespaced spec; plain skills are checked by ParsePlain.
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
		if c != '-' || i == 0 {
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

// Closest returns the candidate nearest to s by edit distance, if it is close
// enough to be a likely typo: at most 2 edits, or a quarter of s's length.
func Closest(s string, candidates []string) (string, bool) {
	best, bestDist := "", -1
	for _, c := range candidates {
		if d := distance(s, c); bestDist < 0 || d < bestDist || (d == bestDist && c < best) {
			best, bestDist = c, d
		}
	}
	if bestDist < 0 || bestDist > max(2, len([]rune(s))/4) {
		return "", false
	}
	return best, true
}

// distance is the Levenshtein distance between a and b, counted in runes.
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// ParsePlain accepts the name of a plain skill.
func ParsePlain(raw string) (Spec, error) {
	name := strings.TrimSpace(raw)
	if err := ValidateSkill(name); err != nil {
		return Spec{}, err
	}
	if n := len(name); n > MaxNameLen {
		return Spec{}, fmt.Errorf("skill name %q is %d characters; the limit is %d", name, n, MaxNameLen)
	}
	return Spec{Skill: name}, nil
}

// ParsePlainPath maps skills/<skill>, a plain skill's registry path, to its spec.
func ParsePlainPath(path string) (Spec, bool) {
	name, ok := strings.CutPrefix(path, "skills/")
	if !ok || strings.Contains(name, "/") {
		return Spec{}, false
	}
	spec, err := ParsePlain(name)
	return spec, err == nil
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
