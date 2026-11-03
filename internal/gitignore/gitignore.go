// Package gitignore manages the default ignore rule for installed skills and the
// per-skill exceptions that commit a skill.
//
// Rule ignores only the direct children of any hidden skills/ directory. That
// keeps a single "!<dir>/skills/<name>" line enough to re-include a whole skill,
// which would not be true for a "skills/**" pattern.
package gitignore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/polarity-dev/mestiere/internal/gitx"
)

const Rule = "**/.*/skills/*"

const header = "# mestiere: skills are ignored by default; add !<dir>/skills/<name> to commit one"

func InRepo(root string) bool {
	out, err := gitx.Run(root, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// IsIgnored evaluates ignore rules for rel, whether or not the path exists or is tracked.
func IsIgnored(root, rel string) (bool, error) {
	_, err := gitx.Run(root, "check-ignore", "-q", "--no-index", "--", rel)
	if err == nil {
		return true, nil
	}
	if gitx.ExitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

// EnsureRule appends Rule unless every probe path is already ignored, so that
// equivalent rules written by hand are respected.
func EnsureRule(root string, probes []string) (added bool, warnings []string, err error) {
	lines, err := read(root)
	if err != nil {
		return false, nil, err
	}
	inRepo := InRepo(root)
	if inRepo {
		ignored, err := allIgnored(root, probes)
		if err != nil {
			return false, nil, err
		}
		if ignored {
			return false, nil, nil
		}
	}
	if !contains(lines, Rule) {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, header, Rule)
		if err := write(root, lines); err != nil {
			return false, nil, err
		}
		added = true
	}
	if inRepo {
		for _, p := range probes {
			ok, err := IsIgnored(root, p)
			if err != nil {
				return added, warnings, err
			}
			if !ok {
				warnings = append(warnings, filepath.ToSlash(filepath.Dir(p))+" is still not ignored; a later rule in .gitignore re-includes it")
			}
		}
	}
	return added, warnings, nil
}

func allIgnored(root string, probes []string) (bool, error) {
	for _, p := range probes {
		ok, err := IsIgnored(root, p)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

// Track adds an exception for each project-relative path.
func Track(root string, rels []string) (added, warnings []string, err error) {
	lines, err := read(root)
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range rels {
		line := "!" + filepath.ToSlash(rel)
		if contains(lines, line) {
			continue
		}
		lines = append(lines, line)
		added = append(added, line)
	}
	if len(added) > 0 {
		if err := write(root, lines); err != nil {
			return nil, nil, err
		}
	}
	if InRepo(root) {
		for _, rel := range rels {
			ignored, err := IsIgnored(root, rel)
			if err != nil {
				return added, warnings, err
			}
			if ignored {
				warnings = append(warnings, rel+" is still ignored; a later rule or an ignored parent directory overrides the exception")
			}
		}
	}
	return added, warnings, nil
}

// Untrack removes the exceptions added by Track.
func Untrack(root string, rels []string) (removed []string, err error) {
	lines, err := read(root)
	if err != nil {
		return nil, err
	}
	drop := map[string]bool{}
	for _, rel := range rels {
		drop["!"+filepath.ToSlash(rel)] = true
	}
	kept := lines[:0]
	for _, l := range lines {
		if drop[strings.TrimSpace(l)] {
			removed = append(removed, strings.TrimSpace(l))
			continue
		}
		kept = append(kept, l)
	}
	if len(removed) > 0 {
		return removed, write(root, kept)
	}
	return nil, nil
}

func contains(lines []string, line string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

func read(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func write(root string, lines []string) error {
	return os.WriteFile(filepath.Join(root, ".gitignore"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
