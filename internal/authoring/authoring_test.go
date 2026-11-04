package authoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/testutil"
)

func scaffold(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := Init(dir, "polarity", ""); err != nil {
		t.Fatal(err)
	}
	if err := NewNamespace(dir, "frontend", "Frontend skills", []string{"web-team"}); err != nil {
		t.Fatal(err)
	}
	if err := NewSkill(dir, naming.Spec{Namespace: "frontend", Skill: "react-patterns"}, "Use for React work."); err != nil {
		t.Fatal(err)
	}
	return dir
}

func messages(issues []Issue) string {
	var out []string
	for _, i := range issues {
		out = append(out, i.String())
	}
	return strings.Join(out, "\n")
}

func TestScaffoldLintsClean(t *testing.T) {
	issues, err := Lint(scaffold(t))
	if err != nil || len(issues) > 0 {
		t.Fatalf("Lint = %v\n%s", err, messages(issues))
	}
}

func TestScaffoldErrors(t *testing.T) {
	dir := scaffold(t)
	if err := Init(dir, "", ""); err == nil {
		t.Error("Init over an existing registry succeeded")
	}
	if err := NewNamespace(dir, "front-end", "", nil); err == nil {
		t.Error("hyphenated namespace accepted")
	}
	if err := NewSkill(dir, naming.Spec{Namespace: "backend", Skill: "x"}, ""); err == nil {
		t.Error("skill in a missing namespace accepted")
	}
	if err := NewNamespace(t.TempDir(), "x", "", nil); err == nil {
		t.Error("namespace outside a registry accepted")
	}
}

func TestLintFindsProblems(t *testing.T) {
	dir := scaffold(t)
	skills := filepath.Join(dir, "skills")
	testutil.WriteFile(t, filepath.Join(skills, "front-end", "x", "SKILL.md"), "---\nname: x\n---\n", 0o644)
	testutil.WriteFile(t, filepath.Join(skills, "frontend", "frontend-wrong-name", "SKILL.md"), "---\nname: other\ndescription: d\n---\n", 0o644)
	testutil.WriteFile(t, filepath.Join(skills, "frontend", "frontend-no-description", "SKILL.md"), "---\nname: frontend-no-description\n---\n", 0o644)
	testutil.WriteFile(t, filepath.Join(skills, "frontend", "unprefixed", "SKILL.md"), "---\nname: frontend-unprefixed\ndescription: d\n---\n", 0o644)
	if err := os.MkdirAll(filepath.Join(skills, "frontend", "frontend-empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(skills, "frontend", "frontend-react-patterns", ".DS_Store"), "", 0o644)
	testutil.WriteFile(t, filepath.Join(skills, "frontend", "frontend-react-patterns", "scripts", ".env.example"), "", 0o644)
	testutil.WriteFile(t, filepath.Join(skills, "frontend", "frontend-react-patterns", ".vscode", "settings.json"), "{}", 0o644)
	testutil.WriteFile(t, filepath.Join(skills, "frontend", ".DS_Store"), "", 0o644)
	if err := os.Symlink("../../../../registry.yaml", filepath.Join(skills, "frontend", "frontend-react-patterns", "escape")); err != nil {
		t.Fatal(err)
	}
	issues, err := Lint(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := messages(issues)
	for _, want := range []string{
		"skills/front-end: invalid namespace",
		"error: skills/frontend/unprefixed: skill folders are named frontend-<skill>, the skill's installed name; rename it to frontend-unprefixed",
		`skills/frontend/frontend-wrong-name/SKILL.md: name must be "frontend-wrong-name"`,
		"skills/frontend/frontend-no-description/SKILL.md: description is required",
		"skills/frontend/frontend-empty: missing SKILL.md",
		"skills/frontend/frontend-react-patterns/.DS_Store: hidden files aren't installed",
		"skills/frontend/frontend-react-patterns/scripts/.env.example: hidden files aren't installed",
		"skills/frontend/frontend-react-patterns/.vscode: hidden files aren't installed",
		"skills/frontend/frontend-react-patterns/escape: links outside the skill and skills/frontend/_shared/",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing issue %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, ".vscode/settings.json") || strings.Contains(got, "skills/frontend/.DS_Store") {
		t.Errorf("unexpected issue in:\n%s", got)
	}
	if !HasErrors(issues) {
		t.Fatal("HasErrors = false")
	}
}

func TestLintShared(t *testing.T) {
	dir := scaffold(t)
	if err := NewSkill(dir, naming.Spec{Namespace: "frontend", Skill: "tables"}, "Tables."); err != nil {
		t.Fatal(err)
	}
	ns := filepath.Join(dir, "skills", "frontend")
	testutil.WriteFile(t, filepath.Join(ns, "_shared", "scripts", "lint.sh"), "#!/bin/sh\n", 0o755)
	testutil.WriteFile(t, filepath.Join(ns, "_shared", "references", "tokens.md"), "t", 0o644)
	testutil.WriteFile(t, filepath.Join(ns, "_shared", "unused.md"), "u", 0o644)
	link := func(at, target string) {
		t.Helper()
		p := filepath.Join(ns, filepath.FromSlash(at))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, p); err != nil {
			t.Fatal(err)
		}
	}
	link("frontend-react-patterns/references/tokens.md", "../../_shared/references/tokens.md")
	link("frontend-tables/scripts", "../_shared/scripts")
	issues, err := Lint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if HasErrors(issues) {
		t.Fatalf("valid shared layout has errors:\n%s", messages(issues))
	}
	if got := messages(issues); !strings.Contains(got, "warning: skills/frontend/_shared/unused.md: no skill links to this file") || strings.Contains(got, "lint.sh") {
		t.Fatalf("unexpected issues:\n%s", got)
	}

	link("_shared/alias.md", "references/tokens.md")
	link("frontend-react-patterns/alias.md", "../_shared/alias.md")
	link("frontend-react-patterns/missing.md", "../_shared/nope.md")
	link("frontend-react-patterns/broken.md", "nope.md")
	testutil.WriteFile(t, filepath.Join(ns, "_shared", ".env"), "x", 0o644)
	issues, _ = Lint(dir)
	got := messages(issues)
	for _, want := range []string{
		"skills/frontend/_shared/alias.md: links aren't allowed inside _shared",
		"skills/frontend/frontend-react-patterns/alias.md: links to skills/frontend/_shared/alias.md, which is itself a link",
		"skills/frontend/frontend-react-patterns/missing.md: links to skills/frontend/_shared/nope.md, which doesn't exist",
		"skills/frontend/frontend-react-patterns/broken.md: broken link",
		"skills/frontend/_shared/.env: hidden files aren't installed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing issue %q in:\n%s", want, got)
		}
	}
}

func TestLintMissingRegistryFile(t *testing.T) {
	issues, _ := Lint(t.TempDir())
	if !strings.Contains(messages(issues), "missing registry manifest") {
		t.Fatalf("got:\n%s", messages(issues))
	}
}
