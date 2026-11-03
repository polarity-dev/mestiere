package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/testutil"
)

func TestMirror(t *testing.T) {
	testutil.Isolate(t)
	reg := testutil.NewRegistry(t)
	reg.Skill("frontend", "forms", "v1")
	reg.Skill("frontend", "react-patterns", "v1")
	reg.Skill("Bad-NS", "x", "invalid namespace is skipped")
	reg.Write("skills/frontend/not-a-skill/README.md", "no SKILL.md")
	testutil.WriteFile(t, filepath.Join(reg.Dir, "skills/frontend/forms/run.sh"), "#!/bin/sh\n", 0o755)
	if err := os.Symlink("SKILL.md", filepath.Join(reg.Dir, "skills/frontend/forms/alias.md")); err != nil {
		t.Fatal(err)
	}
	first := reg.Commit("v1")
	v1 := reg.Tree("skills/frontend/forms")

	m, err := Open(reg.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Exists() {
		t.Fatal("mirror exists before fetch")
	}
	if err := m.Fetch(); err != nil {
		t.Fatal(err)
	}
	commit, err := m.Resolve("")
	if err != nil || commit != first {
		t.Fatalf("Resolve = %s, %v; want %s", commit, err, first)
	}
	if _, err := m.Resolve("nope"); err == nil {
		t.Fatal("unknown ref resolved")
	}
	catalog, err := m.Catalog(commit)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string][]string{"frontend": {"forms", "react-patterns"}}; !reflect.DeepEqual(catalog, want) {
		t.Fatalf("Catalog = %v, want %v", catalog, want)
	}
	tree, err := m.TreeAt(commit, "skills/frontend/forms")
	if err != nil || tree != v1 {
		t.Fatalf("TreeAt = %s, %v", tree, err)
	}

	content, err := m.Materialize(commit, naming.Spec{Namespace: "frontend", Skill: "forms"})
	if err != nil || content.Tree != tree {
		t.Fatalf("Materialize = %+v, %v", content, err)
	}
	dest := filepath.Join(t.TempDir(), "forms")
	if err := m.WriteContent(content, dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.TreeHash(dest, m.ObjectFormat()); got != tree {
		t.Fatalf("extracted tree hash %s != %s", got, tree)
	}

	reg.Skill("frontend", "forms", "v2")
	reg.Commit("v2")
	if err := m.Fetch(); err != nil {
		t.Fatal(err)
	}
	m.fetched = false
	if err := m.Fetch(); err != nil {
		t.Fatal(err)
	}
	trees, err := m.HistoricalTrees(naming.Spec{Namespace: "frontend", Skill: "forms"})
	if err != nil {
		t.Fatal(err)
	}
	if len(trees) != 2 || !trees[v1] || !trees[reg.Tree("skills/frontend/forms")] {
		t.Fatalf("HistoricalTrees = %v", trees)
	}
	if trees, _ := m.HistoricalTrees(naming.Spec{Namespace: "frontend", Skill: "missing"}); len(trees) != 0 {
		t.Fatalf("missing path has history: %v", trees)
	}
}

func TestCacheDirPrecedence(t *testing.T) {
	xdg := t.TempDir()
	override := t.TempDir()
	t.Setenv("MESTIERE_CACHE_DIR", "")
	t.Setenv("XDG_CACHE_HOME", xdg)
	if got, _ := CacheDir(); got != filepath.Join(xdg, "mestiere") {
		t.Errorf("with XDG_CACHE_HOME: %s", got)
	}
	t.Setenv("XDG_CACHE_HOME", "relative/path")
	if got, err := CacheDir(); err != nil || got != filepath.Join(platformCacheDir(), "mestiere") {
		t.Errorf("relative XDG_CACHE_HOME must be ignored per the XDG spec: %q, %v", got, err)
	}
	t.Setenv("MESTIERE_CACHE_DIR", override)
	if got, _ := CacheDir(); got != override {
		t.Errorf("with MESTIERE_CACHE_DIR: %s", got)
	}
}

func TestCleanCacheRemovesOnlyMirrors(t *testing.T) {
	testutil.Isolate(t)
	reg := testutil.NewRegistry(t)
	reg.Skill("frontend", "forms", "v1")
	reg.Commit("v1")
	m, _ := Open(reg.Dir)
	if err := m.Fetch(); err != nil {
		t.Fatal(err)
	}
	root, _ := CacheRoot()
	testutil.WriteFile(t, filepath.Join(root, "unrelated", "keep.txt"), "x", 0o644)
	if size, err := CacheSize(); err != nil || size == 0 {
		t.Fatalf("CacheSize = %d, %v", size, err)
	}
	removed, err := CleanCache()
	if err != nil || len(removed) != 1 || m.Exists() {
		t.Fatalf("CleanCache = %v, %v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(root, "unrelated", "keep.txt")); err != nil {
		t.Fatal("unrelated cache content removed")
	}
	again, _ := Open(reg.Dir)
	if err := again.Fetch(); err != nil || !again.Exists() {
		t.Fatalf("mirror not recreated: %v", err)
	}
}

func TestWriteEntryRejectsEscapes(t *testing.T) {
	cases := map[string]struct {
		entry fsx.Entry
		data  string
	}{
		"parent path":   {fsx.Entry{Path: "../evil", Mode: modeFile}, "x"},
		"absolute link": {fsx.Entry{Path: "link", Mode: modeSymlink}, "/etc/passwd"},
		"escaping link": {fsx.Entry{Path: "a/link", Mode: modeSymlink}, "../../outside"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := writeEntry(t.TempDir(), tc.entry, []byte(tc.data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
