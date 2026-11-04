package registry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/testutil"
)

var (
	forms  = naming.Spec{Namespace: "frontend", Skill: "forms"}
	tables = naming.Spec{Namespace: "frontend", Skill: "tables"}
	plain  = naming.Spec{Namespace: "frontend", Skill: "plain"}
)

// sharedRegistry builds frontend/_shared with a file and a directory, a skill
// linking a single file, one linking the whole scripts directory, and one
// linking nothing.
func sharedRegistry(t *testing.T) *testutil.Registry {
	t.Helper()
	testutil.Isolate(t)
	reg := testutil.NewRegistry(t)
	reg.Write("skills/frontend/_shared/references/tokens.md", "tokens v1")
	testutil.WriteFile(t, filepath.Join(reg.Dir, "skills/frontend/_shared/scripts/lint.sh"), "#!/bin/sh\necho lint\n", 0o755)
	reg.Write("skills/frontend/_shared/scripts/lib/util.sh", "util")
	reg.Skill("frontend", "forms", "forms")
	reg.Link("skills/frontend/forms/references/tokens.md", "../../_shared/references/tokens.md")
	reg.Link("skills/frontend/forms/alias.md", "SKILL.md")
	reg.Skill("frontend", "tables", "tables")
	reg.Link("skills/frontend/tables/scripts", "../_shared/scripts")
	reg.Skill("frontend", "plain", "plain")
	return reg
}

func openMirror(t *testing.T, reg *testutil.Registry) *Mirror {
	t.Helper()
	m, err := Open(reg.Dir)
	if err != nil {
		t.Fatal(err)
	}
	m.fetched = false
	if err := m.Fetch(); err != nil {
		t.Fatal(err)
	}
	return m
}

func head(t *testing.T, m *Mirror) string {
	t.Helper()
	c, err := m.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func install(t *testing.T, m *Mirror, spec naming.Spec) (string, *Content) {
	t.Helper()
	c, err := m.Materialize(head(t, m), spec)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), spec.InstalledName())
	if err := m.WriteContent(c, dest); err != nil {
		t.Fatal(err)
	}
	return dest, c
}

func TestMaterializeSharedFile(t *testing.T) {
	reg := sharedRegistry(t)
	reg.Commit("shared")
	m := openMirror(t, reg)
	dest, c := install(t, m, forms)

	info, err := os.Lstat(filepath.Join(dest, "references/tokens.md"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("shared file not materialized: %v %v", info, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dest, "references/tokens.md")); string(data) != "tokens v1" {
		t.Fatalf("content = %q", data)
	}
	if target, err := os.Readlink(filepath.Join(dest, "alias.md")); err != nil || target != "SKILL.md" {
		t.Fatalf("in-skill link not kept: %q, %v", target, err)
	}
	if got, _ := fsx.TreeHash(dest, fsx.SHA1); got != c.Tree {
		t.Fatalf("installed hash %s != content tree %s", got, c.Tree)
	}
	if c.Tree == reg.Tree("skills/frontend/forms") {
		t.Fatal("materialized tree should differ from the skill's own tree")
	}
}

func TestMaterializeSharedDirectory(t *testing.T) {
	reg := sharedRegistry(t)
	reg.Commit("shared")
	m := openMirror(t, reg)
	dest, c := install(t, m, tables)
	info, err := os.Stat(filepath.Join(dest, "scripts/lint.sh"))
	if err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("executable shared script missing: %v %v", info, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "scripts/lib/util.sh")); err != nil {
		t.Fatal("nested shared file missing")
	}
	if got, _ := fsx.TreeHash(dest, fsx.SHA1); got != c.Tree {
		t.Fatalf("installed hash %s != content tree %s", got, c.Tree)
	}
}

func TestMaterializeSkipsReadme(t *testing.T) {
	reg := sharedRegistry(t)
	reg.Write("skills/frontend/plain/Readme.md", "for people")
	reg.Write("skills/frontend/plain/docs/README.md", "nested, installed")
	reg.Commit("readme")
	m := openMirror(t, reg)
	dest, c := install(t, m, plain)

	if _, err := os.Lstat(filepath.Join(dest, "Readme.md")); !os.IsNotExist(err) {
		t.Fatalf("top-level README installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "docs/README.md")); err != nil {
		t.Fatal("nested README.md should be installed")
	}
	if got, _ := fsx.TreeHash(dest, fsx.SHA1); got != c.Tree {
		t.Fatalf("installed hash %s != content tree %s", got, c.Tree)
	}
	if c.Legacy == nil || c.Legacy.Tree != reg.Tree("skills/frontend/plain") {
		t.Fatalf("legacy = %+v, want the registry tree", c.Legacy)
	}
	trees, err := m.HistoricalTrees(plain)
	if err != nil || !trees[c.Tree] || !trees[c.Legacy.Tree] {
		t.Fatalf("HistoricalTrees = %v, %v", trees, err)
	}
}

func TestMaterializeWithoutLinksKeepsGitTree(t *testing.T) {
	reg := sharedRegistry(t)
	reg.Commit("shared")
	m := openMirror(t, reg)
	c, err := m.Materialize(head(t, m), plain)
	if err != nil || c.Tree != reg.Tree("skills/frontend/plain") {
		t.Fatalf("Materialize = %+v, %v", c, err)
	}
}

func TestMaterializeRejectsInvalidLinks(t *testing.T) {
	cases := map[string]func(reg *testutil.Registry){
		"outside namespace": func(reg *testutil.Registry) {
			reg.Link("skills/frontend/plain/x.md", "../../backend/_shared/x.md")
		},
		"other skill": func(reg *testutil.Registry) {
			reg.Link("skills/frontend/plain/x.md", "../forms/SKILL.md")
		},
		"missing shared file": func(reg *testutil.Registry) {
			reg.Link("skills/frontend/plain/x.md", "../_shared/nope.md")
		},
		"chained link": func(reg *testutil.Registry) {
			reg.Link("skills/frontend/_shared/alias.md", "references/tokens.md")
			reg.Link("skills/frontend/plain/x.md", "../_shared/alias.md")
		},
		"hidden shared file": func(reg *testutil.Registry) {
			reg.Write("skills/frontend/_shared/scripts/.env", "x")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			reg := sharedRegistry(t)
			mutate(reg)
			reg.Commit(name)
			m := openMirror(t, reg)
			spec := plain
			if name == "hidden shared file" {
				spec = tables
			}
			_, err := m.Materialize(head(t, m), spec)
			var ce *ContentError
			if !errors.As(err, &ce) {
				t.Fatalf("err = %v, want ContentError", err)
			}
		})
	}
}

func TestHistoricalTreesFollowSharedChanges(t *testing.T) {
	reg := sharedRegistry(t)
	reg.Commit("v1")
	m := openMirror(t, reg)
	_, v1 := install(t, m, forms)
	plainV1, _ := m.Materialize(head(t, m), plain)

	reg.Write("skills/frontend/_shared/references/tokens.md", "tokens v2")
	reg.Commit("tokens v2")
	m = openMirror(t, reg)
	_, v2 := install(t, m, forms)
	if v1.Tree == v2.Tree {
		t.Fatal("shared change didn't change the installed tree")
	}
	trees, err := m.HistoricalTrees(forms)
	if err != nil {
		t.Fatal(err)
	}
	if len(trees) != 2 || !trees[v1.Tree] || !trees[v2.Tree] {
		t.Fatalf("HistoricalTrees = %v", trees)
	}
	plainV2, _ := m.Materialize(head(t, m), plain)
	if plainV1.Tree != plainV2.Tree {
		t.Fatal("shared change altered a skill that links nothing")
	}
	if strings.Contains(strings.Join(keys(trees), ","), plainV1.Tree) {
		t.Fatal("history mixed up skills")
	}
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
