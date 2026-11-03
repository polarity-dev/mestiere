package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/polarity-dev/mestiere/internal/authoring"
)

// repoRoot is the repository root, which is also the bundled registry.
const repoRoot = "../.."

func TestBundledRegistryLints(t *testing.T) {
	issues, err := authoring.Lint(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range issues {
		t.Errorf("%s", i)
	}
}

var (
	invocationRe = regexp.MustCompile("mestiere((?: [a-z][a-z-]*)+)")
	flagRe       = regexp.MustCompile(`(?:^|\s)--([a-z][a-z-]*)`)
	inlineRe     = regexp.MustCompile("`([^`\n]+)`")
	codeBlockRe  = regexp.MustCompile("(?s)```[a-z]*\n(.*?)```")
)

// TestBundledSkillsUseRealCommands keeps the bundled skills in step with the
// CLI: every `mestiere ...` command and flag written in their code must exist.
func TestBundledSkillsUseRealCommands(t *testing.T) {
	root := NewRootCmd(os.Stdout, os.Stderr, false)
	files, _ := filepath.Glob(filepath.Join(repoRoot, "skills", "*", "*", "SKILL.md"))
	shared, _ := filepath.Glob(filepath.Join(repoRoot, "skills", "*", "_shared", "references", "*.md"))
	if len(files) == 0 {
		t.Fatal("no bundled skills found")
	}
	for _, file := range append(files, shared...) {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, snippet := range codeSnippets(string(data)) {
			for _, line := range strings.Split(snippet, "\n") {
				for _, m := range invocationRe.FindAllStringSubmatch(line, -1) {
					cmd := resolve(root, strings.Fields(m[1]))
					if cmd == root {
						t.Errorf("%s: unknown command %q", file, m[0])
						continue
					}
					for _, flag := range flagRe.FindAllStringSubmatch(line[strings.Index(line, m[0]):], -1) {
						if cmd.Flags().Lookup(flag[1]) == nil && cmd.InheritedFlags().Lookup(flag[1]) == nil {
							t.Errorf("%s: %q has no flag --%s", file, cmd.CommandPath(), flag[1])
						}
					}
				}
			}
		}
	}
}

// codeSnippets returns fenced block contents and inline code spans, which is
// where the skills write commands.
func codeSnippets(md string) []string {
	var out []string
	for _, m := range codeBlockRe.FindAllStringSubmatch(md, -1) {
		out = append(out, m[1])
	}
	for _, m := range inlineRe.FindAllStringSubmatch(codeBlockRe.ReplaceAllString(md, ""), -1) {
		out = append(out, m[1])
	}
	return out
}

// resolve walks args down the command tree and returns the deepest match.
func resolve(root *cobra.Command, args []string) *cobra.Command {
	cmd := root
	for _, a := range args {
		var next *cobra.Command
		for _, c := range cmd.Commands() {
			if c.Name() == a {
				next = c
			}
		}
		if next == nil {
			return cmd
		}
		cmd = next
	}
	return cmd
}
