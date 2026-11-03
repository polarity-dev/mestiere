package gitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polarity-dev/mestiere/internal/testutil"
)

var probes = []string{".agents/skills/mestiere-probe", ".claude/skills/mestiere-probe"}

func repo(t *testing.T) string {
	t.Helper()
	testutil.Isolate(t)
	dir := t.TempDir()
	testutil.InitRepo(t, dir)
	return dir
}

func ignored(t *testing.T, root, rel string) bool {
	t.Helper()
	ok, err := IsIgnored(root, rel)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestEnsureRule(t *testing.T) {
	root := repo(t)
	testutil.WriteFile(t, filepath.Join(root, ".gitignore"), "node_modules/", 0o644)
	added, warnings, err := EnsureRule(root, probes)
	if err != nil || !added || len(warnings) > 0 {
		t.Fatalf("EnsureRule = %v, %v, %v", added, warnings, err)
	}
	for _, rel := range []string{".agents/skills/x", ".claude/skills/x/SKILL.md", "packages/app/.kiro/skills/x"} {
		if !ignored(t, root, rel) {
			t.Errorf("%s isn't ignored", rel)
		}
	}
	for _, rel := range []string{".claude/settings.json", ".claude/agents/a.md"} {
		if ignored(t, root, rel) {
			t.Errorf("%s is ignored", rel)
		}
	}
	added, _, _ = EnsureRule(root, probes)
	if added {
		t.Fatal("rule added twice")
	}
	data, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !strings.HasPrefix(string(data), "node_modules/\n") {
		t.Fatalf("existing rules not preserved:\n%s", data)
	}
}

func TestEnsureRuleRespectsEquivalentRules(t *testing.T) {
	root := repo(t)
	testutil.WriteFile(t, filepath.Join(root, ".gitignore"), ".agents/\n.claude/\n", 0o644)
	added, _, err := EnsureRule(root, probes)
	if err != nil || added {
		t.Fatalf("EnsureRule = %v, %v", added, err)
	}
}

func TestEnsureRuleWarnsOnNegation(t *testing.T) {
	root := repo(t)
	testutil.WriteFile(t, filepath.Join(root, ".gitignore"), Rule+"\n!.claude/skills/mestiere-probe\n", 0o644)
	_, warnings, err := EnsureRule(root, probes)
	if err != nil || len(warnings) != 1 {
		t.Fatalf("warnings = %v, err = %v", warnings, err)
	}
}

func TestTrackAndUntrack(t *testing.T) {
	root := repo(t)
	if _, _, err := EnsureRule(root, probes); err != nil {
		t.Fatal(err)
	}
	rels := []string{".agents/skills/mine", ".claude/skills/mine"}
	added, warnings, err := Track(root, rels)
	if err != nil || len(added) != 2 || len(warnings) > 0 {
		t.Fatalf("Track = %v, %v, %v", added, warnings, err)
	}
	if ignored(t, root, ".agents/skills/mine/scripts/run.sh") {
		t.Fatal("tracked skill contents are still ignored")
	}
	if !ignored(t, root, ".agents/skills/other") {
		t.Fatal("other skills are no longer ignored")
	}
	if again, _, _ := Track(root, rels); len(again) != 0 {
		t.Fatal("exception added twice")
	}
	removed, err := Untrack(root, rels)
	if err != nil || len(removed) != 2 {
		t.Fatalf("Untrack = %v, %v", removed, err)
	}
	if !ignored(t, root, ".agents/skills/mine") {
		t.Fatal("untracked skill isn't ignored")
	}
}

func TestOutsideRepo(t *testing.T) {
	testutil.Isolate(t)
	root := t.TempDir()
	added, _, err := EnsureRule(root, probes)
	if err != nil || !added {
		t.Fatalf("EnsureRule = %v, %v", added, err)
	}
}
