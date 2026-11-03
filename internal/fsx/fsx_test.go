package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polarity-dev/mestiere/internal/testutil"
)

func buildSkillDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "skill")
	testutil.WriteFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: a-b\n---\n", 0o644)
	testutil.WriteFile(t, filepath.Join(dir, "scripts", "run.sh"), "#!/bin/sh\necho hi\n", 0o755)
	testutil.WriteFile(t, filepath.Join(dir, "references", "a.md"), "a", 0o644)
	// git sorts directory "a" as "a/", which places it after "a.md".
	testutil.WriteFile(t, filepath.Join(dir, "a", "x.md"), "x", 0o644)
	testutil.WriteFile(t, filepath.Join(dir, "a.md"), "a", 0o644)
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("references/a.md", filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// gitTree asks git for the tree ID of dir using a separate git dir, so the
// work tree stays free of .git.
func gitTree(t *testing.T, dir, format string) string {
	t.Helper()
	gitDir := filepath.Join(t.TempDir(), "git")
	testutil.Git(t, dir, "--git-dir="+gitDir, "init", "--quiet", "--object-format="+format)
	testutil.Git(t, dir, "--git-dir="+gitDir, "--work-tree="+dir, "add", "-A")
	return testutil.Git(t, dir, "--git-dir="+gitDir, "--work-tree="+dir, "write-tree")
}

func TestTreeHashMatchesGit(t *testing.T) {
	testutil.Isolate(t)
	for _, format := range []ObjectFormat{SHA1, SHA256} {
		t.Run(string(format), func(t *testing.T) {
			dir := buildSkillDir(t)
			got, err := TreeHash(dir, format)
			if err != nil {
				t.Fatal(err)
			}
			if want := gitTree(t, dir, string(format)); got != want {
				t.Fatalf("TreeHash = %s, git write-tree = %s", got, want)
			}
			if FormatOf(got) != format {
				t.Fatalf("FormatOf(%s) = %s", got, FormatOf(got))
			}
		})
	}
}

func TestTreeHashIgnoresHiddenEntries(t *testing.T) {
	dir := buildSkillDir(t)
	before, err := TreeHash(dir, SHA1)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(dir, ".DS_Store"), "junk", 0o644)
	testutil.WriteFile(t, filepath.Join(dir, "scripts", ".run.sh.swp"), "junk", 0o644)
	testutil.WriteFile(t, filepath.Join(dir, ".vscode", "settings.json"), "{}", 0o644)
	after, err := TreeHash(dir, SHA1)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("hidden entries changed the tree hash")
	}
	copied := filepath.Join(t.TempDir(), "copy")
	if err := CopyDir(dir, copied); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".DS_Store", ".vscode", "scripts/.run.sh.swp"} {
		if _, err := os.Lstat(filepath.Join(copied, rel)); !os.IsNotExist(err) {
			t.Errorf("CopyDir copied hidden entry %s", rel)
		}
	}
}

func TestTreeHashDetectsChanges(t *testing.T) {
	dir := buildSkillDir(t)
	before, _ := TreeHash(dir, SHA1)
	if err := os.Chmod(filepath.Join(dir, "scripts", "run.sh"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, _ := TreeHash(dir, SHA1)
	if before == after {
		t.Fatal("dropping the executable bit didn't change the hash")
	}
}

func TestHashEntriesMatchesGit(t *testing.T) {
	testutil.Isolate(t)
	dir := buildSkillDir(t)
	want := gitTree(t, dir, "sha1")
	gitDir := filepath.Join(t.TempDir(), "git")
	testutil.Git(t, dir, "--git-dir="+gitDir, "init", "--quiet")
	testutil.Git(t, dir, "--git-dir="+gitDir, "--work-tree="+dir, "add", "-A")
	tree := testutil.Git(t, dir, "--git-dir="+gitDir, "--work-tree="+dir, "write-tree")
	listing := testutil.Git(t, dir, "--git-dir="+gitDir, "ls-tree", "-r", tree)
	var entries []Entry
	for _, line := range strings.Split(listing, "\n") {
		meta, p, _ := strings.Cut(line, "\t")
		f := strings.Fields(meta)
		entries = append(entries, Entry{Path: p, Mode: f[0], OID: f[2]})
	}
	got, err := HashEntries(SHA1, entries)
	if err != nil || got != want {
		t.Fatalf("HashEntries = %s, %v; want %s", got, err, want)
	}
	if _, err := HashEntries(SHA1, append(entries, Entry{Path: "SKILL.md/x", Mode: "100644", OID: entries[0].OID})); err == nil {
		t.Fatal("file/directory conflict accepted")
	}
}

func TestCopyDirPreservesTree(t *testing.T) {
	src := buildSkillDir(t)
	dst := filepath.Join(t.TempDir(), "copy")
	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	a, _ := TreeHash(src, SHA1)
	b, _ := TreeHash(dst, SHA1)
	if a != b {
		t.Fatalf("copy hash %s != source hash %s", b, a)
	}
	if target, err := os.Readlink(filepath.Join(dst, "link.md")); err != nil || target != "references/a.md" {
		t.Fatalf("symlink not preserved: %q, %v", target, err)
	}
}

func TestReplaceDir(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	testutil.WriteFile(t, filepath.Join(dest, "old.txt"), "old", 0o644)
	staged := filepath.Join(root, "staged")
	testutil.WriteFile(t, filepath.Join(staged, "new.txt"), "new", 0o644)
	if err := ReplaceDir(staged, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "old.txt")); !os.IsNotExist(err) {
		t.Fatal("old content survived")
	}
	if _, err := os.Stat(filepath.Join(dest, "new.txt")); err != nil {
		t.Fatal("new content missing")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("leftover entries: %v", entries)
	}
}
