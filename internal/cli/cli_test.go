package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polarity-dev/mestiere/internal/testutil"
)

type run struct {
	t   *testing.T
	dir string
}

func (r run) exec(args ...string) (string, string, error) {
	r.t.Helper()
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut, false)
	cmd.SetArgs(append([]string{"-C", r.dir}, args...))
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func (r run) ok(args ...string) string {
	r.t.Helper()
	out, errOut, err := r.exec(args...)
	if err != nil {
		r.t.Fatalf("mestiere %s: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, out, errOut)
	}
	return out
}

func TestEndToEnd(t *testing.T) {
	testutil.Isolate(t)
	regDir := filepath.Join(t.TempDir(), "registry")
	author := run{t: t, dir: regDir}
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatal(err)
	}
	author.ok("registry", "init", "--name", "polarity")
	author.ok("registry", "new-namespace", "frontend")
	author.ok("registry", "new-skill", "frontend/forms", "--description", "Build accessible forms.")
	if out := author.ok("registry", "lint"); !strings.Contains(out, "registry is valid") {
		t.Fatalf("lint output: %s", out)
	}
	testutil.InitRepo(t, regDir)
	testutil.Git(t, regDir, "add", "-A")
	testutil.Git(t, regDir, "commit", "--quiet", "-m", "init")

	projectDir := filepath.Join(t.TempDir(), "app")
	testutil.InitRepo(t, projectDir)
	app := run{t: t, dir: projectDir}
	app.ok("init", "--agents", "claude,kiro")
	app.ok("registry", "add", "polarity", regDir)
	app.ok("add", "frontend")
	if _, err := os.Readlink(filepath.Join(projectDir, ".claude/skills/frontend-forms")); err != nil {
		t.Fatalf("link missing: %v", err)
	}
	if out := app.ok("status"); !strings.Contains(out, "frontend-forms") || !strings.Contains(out, "claude:ok") {
		t.Fatalf("status output:\n%s", out)
	}
	if out := app.ok("outdated"); !strings.Contains(out, "up to date") {
		t.Fatalf("outdated output:\n%s", out)
	}
	if out := app.ok("doctor"); !strings.Contains(out, "no problems found") {
		t.Fatalf("doctor output:\n%s", out)
	}
	if out := app.ok("cache", "dir"); strings.TrimSpace(out) != os.Getenv("MESTIERE_CACHE_DIR") {
		t.Fatalf("cache dir = %q", out)
	}
	if out := app.ok("cache", "info"); !strings.Contains(out, "MB") {
		t.Fatalf("cache info = %q", out)
	}
	app.ok("eject", "frontend-forms", "--track")
	if data, _ := os.ReadFile(filepath.Join(projectDir, ".gitignore")); !strings.Contains(string(data), "!.agents/skills/frontend-forms") {
		t.Fatalf(".gitignore:\n%s", data)
	}
	app.ok("sync", "--frozen")

	testutil.WriteFile(t, filepath.Join(projectDir, ".agents/skills/frontend-forms/SKILL.md"), "edited", 0o644)
	app.ok("sync")

	_, errOut, err := app.exec("remove", "backend")
	if err == nil || !strings.Contains(err.Error(), "isn't installed") {
		t.Fatalf("remove err = %v, stderr = %s", err, errOut)
	}
}

func TestSyncFailureExitsWithProblems(t *testing.T) {
	testutil.Isolate(t)
	reg := testutil.NewRegistry(t)
	reg.Skill("frontend", "forms", "v1")
	reg.Commit("v1")
	projectDir := filepath.Join(t.TempDir(), "app")
	testutil.InitRepo(t, projectDir)
	app := run{t: t, dir: projectDir}
	app.ok("init", "--agents", "claude")
	app.ok("registry", "add", "polarity", reg.Dir)
	app.ok("add", "frontend")
	testutil.WriteFile(t, filepath.Join(projectDir, ".agents/skills/frontend-forms/SKILL.md"), "edited", 0o644)
	_, errOut, err := app.exec("sync")
	if !errors.Is(err, ErrFailed) || !strings.Contains(errOut, "has local changes") {
		t.Fatalf("err = %v, stderr = %s", err, errOut)
	}
	app.ok("sync", "--force")
}

func TestMissingManifest(t *testing.T) {
	testutil.Isolate(t)
	_, _, err := run{t: t, dir: t.TempDir()}.exec("sync")
	if err == nil || !strings.Contains(err.Error(), "mestiere init") {
		t.Fatalf("err = %v", err)
	}
}

func TestStatusWithoutManifestShowsLocalAndGlobalSkills(t *testing.T) {
	testutil.Isolate(t)
	home := os.Getenv("HOME")
	repo := filepath.Join(t.TempDir(), "plain")
	testutil.InitRepo(t, repo)
	sub := filepath.Join(repo, "pkg", "deep")
	for _, dir := range []string{
		filepath.Join(home, ".claude/skills/personal"),
		filepath.Join(repo, ".kiro/skills/house-style"),
		sub,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out := run{t: t, dir: sub}.ok("status")
	for _, want := range []string{"no skills.yaml found", ".kiro/skills/house-style                 committed", "global skills:", "~/.claude/skills/personal"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "mestiere track") {
		t.Fatalf("track hint needs a manifest:\n%s", out)
	}
}

func TestGlobalAdd(t *testing.T) {
	testutil.Isolate(t)
	home := os.Getenv("HOME")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	regDir := filepath.Join(t.TempDir(), "registry")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatal(err)
	}
	author := run{t: t, dir: regDir}
	author.ok("registry", "init", "--name", "polarity")
	author.ok("registry", "new-namespace", "frontend")
	author.ok("registry", "new-skill", "frontend/forms", "--description", "Build accessible forms.")
	testutil.InitRepo(t, regDir)
	testutil.Git(t, regDir, "add", "-A")
	testutil.Git(t, regDir, "commit", "--quiet", "-m", "init")

	elsewhere := run{t: t, dir: t.TempDir()}
	if out := elsewhere.ok("registry", "list", "-g"); !strings.Contains(out, "github.com/polarity-dev/mestiere") {
		t.Fatalf("new global manifest lacks the built-in registry:\n%s", out)
	}
	elsewhere.ok("registry", "add", "-g", "polarity", regDir)
	elsewhere.ok("add", "-g", "-r", "polarity", "frontend")

	if _, err := os.Stat(filepath.Join(home, ".config/mestiere/skills.yaml")); err != nil {
		t.Fatalf("global manifest: %v", err)
	}
	if _, err := os.Readlink(filepath.Join(home, ".claude/skills/frontend-forms")); err != nil {
		t.Fatalf("claude link: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".kiro")); err == nil {
		t.Fatal("global install created a folder for an agent the user doesn't have")
	}
	if out := elsewhere.ok("status", "-g"); !strings.Contains(out, "frontend-forms") || !strings.Contains(out, "claude:ok") {
		t.Fatalf("status -g:\n%s", out)
	}
	if out := elsewhere.ok("doctor", "-g"); strings.Contains(out, "git") {
		t.Fatalf("doctor -g mentions git checks:\n%s", out)
	}
	if _, _, err := elsewhere.exec("track", "-g", "frontend-forms"); err == nil {
		t.Fatal("track -g succeeded")
	}
	elsewhere.ok("remove", "-g", "frontend")
	if _, err := os.Lstat(filepath.Join(home, ".claude/skills/frontend-forms")); err == nil {
		t.Fatal("remove -g left the link behind")
	}
}
