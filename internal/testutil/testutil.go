// Package testutil builds throwaway git repositories for tests.
package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Isolate points git and mestiere at per-test locations so tests never read
// the developer's global git config (signing, hooksPath), registry cache or
// home-level skills.
func Isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	globalConfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(globalConfig, []byte("[init]\n\tdefaultBranch = main\n[protocol \"file\"]\n\tallow = always\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	t.Setenv("MESTIERE_CACHE_DIR", filepath.Join(home, "cache"))
}

func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func InitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, dir, "init", "--quiet", "-b", "main")
}

func WriteFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// Registry is a registry repository under construction.
type Registry struct {
	t   *testing.T
	Dir string
}

func NewRegistry(t *testing.T) *Registry {
	t.Helper()
	r := &Registry{t: t, Dir: filepath.Join(t.TempDir(), "registry")}
	InitRepo(t, r.Dir)
	r.Write("registry.yaml", "schemaVersion: 1\nname: test\n")
	return r
}

func (r *Registry) Write(rel, content string) {
	WriteFile(r.t, filepath.Join(r.Dir, filepath.FromSlash(rel)), content, 0o644)
}

// Skill writes a valid SKILL.md for ns/skill with body as its instructions.
func (r *Registry) Skill(ns, skill, body string) {
	r.Write(fmt.Sprintf("skills/%s/%s-%s/SKILL.md", ns, ns, skill),
		fmt.Sprintf("---\nname: %s-%s\ndescription: Test skill %s.\n---\n\n%s\n", ns, skill, skill, body))
}

func (r *Registry) Remove(rel string) {
	if err := os.RemoveAll(filepath.Join(r.Dir, filepath.FromSlash(rel))); err != nil {
		r.t.Fatal(err)
	}
}

// Link creates a symlink at rel pointing to target.
func (r *Registry) Link(rel, target string) {
	p := filepath.Join(r.Dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.Symlink(target, p); err != nil {
		r.t.Fatal(err)
	}
}

// Commit commits everything and returns the new HEAD.
func (r *Registry) Commit(msg string) string {
	Git(r.t, r.Dir, "add", "-A")
	Git(r.t, r.Dir, "commit", "--quiet", "--allow-empty", "-m", msg)
	return Git(r.t, r.Dir, "rev-parse", "HEAD")
}

// Tree returns the tree ID of rel at HEAD.
func (r *Registry) Tree(rel string) string {
	return Git(r.t, r.Dir, "rev-parse", "HEAD:"+rel)
}
