// Package skillmd parses SKILL.md frontmatter as defined by the Agent Skills spec.
package skillmd

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxDescriptionLen is the Agent Skills spec limit for a description.
const MaxDescriptionLen = 1024

type Frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func Parse(data []byte) (Frontmatter, error) {
	text := strings.ReplaceAll(strings.TrimPrefix(string(data), "\ufeff"), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Frontmatter{}, errors.New("missing frontmatter: SKILL.md must start with ---")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return Frontmatter{}, errors.New("unterminated frontmatter: missing closing ---")
	}
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
		return Frontmatter{}, fmt.Errorf("invalid frontmatter: %w", err)
	}
	return fm, nil
}

// Template returns a minimal SKILL.md for a new skill.
func Template(name, description string) string {
	if description == "" {
		description = "Describe what this skill does and when an agent should use it."
	}
	return fmt.Sprintf(`---
name: %s
description: %s
---

# %s

Write the instructions the agent should follow when this skill applies.
`, name, yamlScalar(description), name)
}

func yamlScalar(s string) string {
	out, err := yaml.Marshal(s)
	if err != nil {
		return s
	}
	return strings.TrimSpace(string(out))
}
