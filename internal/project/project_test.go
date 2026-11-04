package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/gitignore"
	"github.com/polarity-dev/mestiere/internal/manifest"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/testutil"
)

type fixture struct {
	t   *testing.T
	reg *testutil.Registry
	dir string
	p   *Project
}

// newFixture creates a registry with frontend/{forms,react-patterns} and
// backend/db-migrations, and a project configured for claude and kiro.
func newFixture(t *testing.T, link string) *fixture {
	t.Helper()
	testutil.Isolate(t)
	reg := testutil.NewRegistry(t)
	reg.Skill("frontend", "forms", "forms v1")
	reg.Skill("frontend", "react-patterns", "react v1")
	reg.Skill("backend", "db-migrations", "db v1")
	testutil.WriteFile(t, filepath.Join(reg.Dir, "skills/backend/backend-db-migrations/scripts/migrate.sh"), "#!/bin/sh\n", 0o755)
	reg.Commit("initial skills")

	dir := filepath.Join(t.TempDir(), "project")
	testutil.InitRepo(t, dir)
	p, r, err := Init(dir, []string{"claude", "kiro"}, link)
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	if err := p.AddRegistry("polarity", reg.Dir, ""); err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, reg: reg, dir: dir, p: p}
}

// reopen reloads the project from disk, as a new CLI invocation would.
func (f *fixture) reopen() *Project {
	f.t.Helper()
	p, err := Open(f.dir)
	if err != nil {
		f.t.Fatal(err)
	}
	f.p = p
	return p
}

func (f *fixture) path(rel string) string { return filepath.Join(f.dir, filepath.FromSlash(rel)) }

func (f *fixture) exists(rel string) bool {
	_, err := os.Lstat(f.path(rel))
	return err == nil
}

func (f *fixture) read(rel string) string {
	f.t.Helper()
	data, err := os.ReadFile(f.path(rel))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

func (f *fixture) sync(opts SyncOptions) *Report {
	f.t.Helper()
	r, err := f.reopen().Sync(opts)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}

func (f *fixture) lockNames() []string {
	var names []string
	for name := range f.reopen().Manifest.Registry("polarity").Lock {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func requireClean(t *testing.T, r *Report) {
	t.Helper()
	if r != nil && (len(r.Problems) > 0 || len(r.Warnings) > 0) {
		t.Fatalf("problems: %v\nwarnings: %v", r.Problems, r.Warnings)
	}
}

func requireNoActions(t *testing.T, r *Report) {
	t.Helper()
	requireClean(t, r)
	if len(r.Actions) > 0 {
		t.Fatalf("expected no actions, got %v", r.Actions)
	}
}

func requireProblem(t *testing.T, r *Report, substr string) {
	t.Helper()
	for _, p := range r.Problems {
		if strings.Contains(p, substr) {
			return
		}
	}
	t.Fatalf("no problem containing %q in %v", substr, r.Problems)
}

func (f *fixture) add(args ...string) *Report {
	f.t.Helper()
	r, err := f.reopen().Add("", args, false)
	if err != nil {
		f.t.Fatal(err)
	}
	requireClean(f.t, r)
	return r
}

func TestAddNamespaceInstallsAndLinks(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")

	if got := strings.Join(f.lockNames(), ","); got != "frontend-forms,frontend-react-patterns" {
		t.Fatalf("lock = %s", got)
	}
	if !strings.Contains(f.read(".agents/skills/frontend-forms/SKILL.md"), "forms v1") {
		t.Fatal("skill content not installed")
	}
	for _, dir := range []string{".claude/skills", ".kiro/skills"} {
		target, err := os.Readlink(f.path(dir + "/frontend-forms"))
		if err != nil || target != "../../.agents/skills/frontend-forms" {
			t.Fatalf("%s link = %q, %v", dir, target, err)
		}
	}
	if f.exists(".agents/skills/backend-db-migrations") {
		t.Fatal("unrequested namespace installed")
	}
	for _, rel := range []string{".agents/skills/frontend-forms", ".claude/skills/frontend-forms"} {
		if ok, _ := gitignore.IsIgnored(f.dir, rel); !ok {
			t.Fatalf("%s isn't ignored", rel)
		}
	}
	requireNoActions(t, f.sync(SyncOptions{}))
}

func TestAddSingleSkillKeepsExecutableBit(t *testing.T) {
	f := newFixture(t, "")
	f.add("backend/db-migrations")
	info, err := os.Stat(f.path(".agents/skills/backend-db-migrations/scripts/migrate.sh"))
	if err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("script not executable: %v %v", info, err)
	}
	if got := f.reopen().Manifest.Registry("polarity").Install; strings.Join(got, ",") != "backend/db-migrations" {
		t.Fatalf("install = %v", got)
	}
}

func TestSyncRestoresDeletedFilesAndLinks(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	os.RemoveAll(f.path(".agents/skills/frontend-forms"))
	os.Remove(f.path(".kiro/skills/frontend-react-patterns"))
	requireClean(t, f.sync(SyncOptions{}))
	if !f.exists(".agents/skills/frontend-forms/SKILL.md") || !f.exists(".kiro/skills/frontend-react-patterns") {
		t.Fatal("not restored")
	}
}

func TestSyncFrozenRestoresFreshClone(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	for _, dir := range []string{".agents", ".claude", ".kiro"} {
		os.RemoveAll(f.path(dir))
	}
	requireClean(t, f.sync(SyncOptions{Frozen: true}))
	if !f.exists(".claude/skills/frontend-forms/SKILL.md") {
		t.Fatal("frozen sync didn't install")
	}
}

func TestSyncRefusesLocalEdits(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	testutil.WriteFile(t, f.path(".agents/skills/frontend-forms/SKILL.md"), "edited", 0o644)
	r := f.sync(SyncOptions{})
	requireProblem(t, r, ".agents/skills/frontend-forms has local changes")
	if f.read(".agents/skills/frontend-forms/SKILL.md") != "edited" {
		t.Fatal("local edit overwritten without --force")
	}
	requireClean(t, f.sync(SyncOptions{Force: true}))
	if !strings.Contains(f.read(".agents/skills/frontend-forms/SKILL.md"), "forms v1") {
		t.Fatal("--force didn't restore content")
	}
}

func TestOutdatedAndUpdate(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	before := f.reopen().Manifest.Registry("polarity").Lock

	f.reg.Skill("frontend", "forms", "forms v2")
	head := f.reg.Commit("forms v2")

	changes, err := f.reopen().Outdated()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Name != "frontend-forms" || changes[0].Kind != Updated {
		t.Fatalf("Outdated = %v", changes)
	}
	if f.reopen().Manifest.Registry("polarity").Lock["frontend-forms"].Commit != before["frontend-forms"].Commit {
		t.Fatal("Outdated wrote the lock")
	}

	changes, r, err := f.reopen().Update(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	if len(changes) != 1 {
		t.Fatalf("Update changes = %v", changes)
	}
	lock := f.reopen().Manifest.Registry("polarity").Lock
	if lock["frontend-forms"].Commit != head || lock["frontend-forms"].Tree != f.reg.Tree("skills/frontend/frontend-forms") {
		t.Fatalf("lock not moved: %+v", lock["frontend-forms"])
	}
	if lock["frontend-react-patterns"].Tree != before["frontend-react-patterns"].Tree {
		t.Fatal("unchanged skill got a new tree")
	}
	if !strings.Contains(f.read(".claude/skills/frontend-forms/SKILL.md"), "forms v2") {
		t.Fatal("new content not installed")
	}
}

func TestUpdateTargetsOnlyNamedSkill(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	f.reg.Skill("frontend", "forms", "forms v2")
	f.reg.Skill("frontend", "react-patterns", "react v2")
	f.reg.Commit("both v2")
	changes, _, err := f.reopen().Update([]string{"frontend-forms"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Name != "frontend-forms" {
		t.Fatalf("changes = %v", changes)
	}
	if !strings.Contains(f.read(".agents/skills/frontend-react-patterns/SKILL.md"), "react v1") {
		t.Fatal("untargeted skill was updated")
	}
	if _, _, err := f.reopen().Update([]string{"backend"}, false); err == nil {
		t.Fatal("updating an uninstalled namespace succeeded")
	}
}

func TestUpdateFollowsNamespace(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	f.reg.Remove("skills/frontend/frontend-forms")
	f.reg.Skill("frontend", "tables", "tables v1")
	f.reg.Commit("reshape frontend")
	changes, r, err := f.reopen().Update(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	kinds := map[string]ChangeKind{}
	for _, c := range changes {
		kinds[c.Name] = c.Kind
	}
	if kinds["frontend-forms"] != Removed || kinds["frontend-tables"] != Added {
		t.Fatalf("changes = %v", changes)
	}
	if f.exists(".agents/skills/frontend-forms") || f.exists(".claude/skills/frontend-forms") {
		t.Fatal("skill deleted upstream is still installed")
	}
	if !f.exists(".kiro/skills/frontend-tables/SKILL.md") {
		t.Fatal("new namespace skill not installed")
	}
}

func TestUpdateFailsWhenSingleSkillDeletedUpstream(t *testing.T) {
	f := newFixture(t, "")
	f.add("backend/db-migrations")
	f.reg.Remove("skills/backend/backend-db-migrations")
	f.reg.Commit("drop db-migrations")
	_, _, err := f.reopen().Update(nil, false)
	if err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("err = %v", err)
	}
	if !f.exists(".agents/skills/backend-db-migrations") {
		t.Fatal("failed update removed files")
	}
}

// A teammate updated the lock and the change arrived with a pull: the old
// version on disk is published content, so sync replaces it silently.
func TestSyncUpgradesPublishedVersionFromPulledLock(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	f.reg.Skill("frontend", "forms", "forms v2")
	head := f.reg.Commit("forms v2")

	p := f.reopen()
	p.Manifest.Registry("polarity").Lock["frontend-forms"] = manifest.LockEntry{
		Path: "skills/frontend/frontend-forms", Commit: head, Tree: f.reg.Tree("skills/frontend/frontend-forms"),
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	r := f.sync(SyncOptions{Frozen: true})
	requireClean(t, r)
	if !strings.Contains(f.read(".agents/skills/frontend-forms/SKILL.md"), "forms v2") {
		t.Fatal("pulled lock not applied")
	}
}

func TestReadmeIsNotInstalledAndOldLocksStillSync(t *testing.T) {
	f := newFixture(t, "")
	f.reg.Write("skills/frontend/frontend-forms/README.md", "for people")
	head := f.reg.Commit("forms readme")
	f.add("frontend")
	if f.exists(".agents/skills/frontend-forms/README.md") {
		t.Fatal("README.md installed")
	}

	// A lock written by 1.0 holds the tree with README.md; a fresh clone installs exactly that.
	os.RemoveAll(f.path(".agents/skills/frontend-forms"))
	p := f.reopen()
	p.Manifest.Registry("polarity").Lock["frontend-forms"] = manifest.LockEntry{
		Path: "skills/frontend/frontend-forms", Commit: head, Tree: f.reg.Tree("skills/frontend/frontend-forms"),
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{Frozen: true}))
	if !f.exists(".agents/skills/frontend-forms/README.md") {
		t.Fatal("old lock not installed as locked")
	}
	requireNoActions(t, f.sync(SyncOptions{Frozen: true}))

	// Relocking drops README.md, and the old install counts as published.
	if _, _, err := f.reopen().Update(nil, false); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{}))
	if f.exists(".agents/skills/frontend-forms/README.md") {
		t.Fatal("README.md kept after update")
	}
}

// Locks written by 1.0 point at skills/<ns>/<skill>: they still sync from the
// locked commit, and update relocks them once the registry renames the folder.
func TestOldLayoutLockSyncsAndRelocks(t *testing.T) {
	f := newFixture(t, "")
	f.reg.Write("skills/frontend/old/SKILL.md", "---\nname: frontend-old\ndescription: d\n---\n\nold v1\n")
	legacy := f.reg.Commit("1.0 layout")
	f.add("frontend")
	p := f.reopen()
	p.Manifest.Registry("polarity").Lock["frontend-old"] = manifest.LockEntry{
		Path: "skills/frontend/old", Commit: legacy, Tree: f.reg.Tree("skills/frontend/old"),
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{Frozen: true}))
	if !strings.Contains(f.read(".agents/skills/frontend-old/SKILL.md"), "old v1") {
		t.Fatal("old lock not installed")
	}

	f.reg.Remove("skills/frontend/old")
	f.reg.Skill("frontend", "old", "old v2")
	f.reg.Commit("rename to current layout")
	if _, _, err := f.reopen().Update(nil, false); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{}))
	if e := f.reopen().Manifest.Registry("polarity").Lock["frontend-old"]; e.Path != "skills/frontend/frontend-old" {
		t.Fatalf("lock not moved to the current path: %+v", e)
	}
	if !strings.Contains(f.read(".agents/skills/frontend-old/SKILL.md"), "old v2") {
		t.Fatal("relocked version not installed")
	}
}

func TestAddNamespaceAfterOneOfItsSkills(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend/forms")
	pinned := f.reopen().Manifest.Registry("polarity").Lock["frontend-forms"]
	f.reg.Skill("frontend", "forms", "forms v2")
	f.reg.Commit("forms v2")
	f.add("frontend")
	if !f.exists(".agents/skills/frontend-react-patterns/SKILL.md") {
		t.Fatal("rest of the namespace not installed")
	}
	if got := f.reopen().Manifest.Registry("polarity").Lock["frontend-forms"]; got != pinned {
		t.Fatalf("already locked skill moved: %+v", got)
	}
}

func TestHyphenatedNamespace(t *testing.T) {
	f := newFixture(t, "")
	f.reg.Skill("ai-code", "review", "review v1")
	f.reg.Skill("ai-code", "lint", "lint v1")
	f.reg.Commit("ai-code namespace")

	f.add("ai-code-review")
	if !strings.Contains(f.read(".agents/skills/ai-code-review/SKILL.md"), "review v1") {
		t.Fatal("installed name didn't resolve to ai-code/review")
	}
	if got := f.reopen().Manifest.Registry("polarity").Install; len(got) != 1 || got[0] != "ai-code/review" {
		t.Fatalf("install = %v", got)
	}
	f.add("ai-code-lint")
	if !f.exists(".agents/skills/ai-code-lint/SKILL.md") {
		t.Fatal("second skill not installed")
	}

	f.reg.Skill("ai-code", "review", "review v2")
	f.reg.Commit("review v2")
	if _, _, err := f.reopen().Update([]string{"ai-code-review"}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.read(".agents/skills/ai-code-review/SKILL.md"), "review v2") {
		t.Fatal("update by installed name didn't apply")
	}

	// ai/code-review would install over ai-code/review, so it is refused.
	f.reg.Skill("ai", "code-review", "clash")
	f.reg.Commit("clash")
	if _, _, err := f.reopen().Update(nil, false); err == nil || !strings.Contains(err.Error(), "which install as ai-code-review") {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.reopen().Eject([]string{"ai-code-revie"}, false); err == nil || !strings.Contains(err.Error(), "matches no namespace or skill") {
		t.Fatalf("eject err = %v", err)
	}
}

func TestPickSpecAmbiguous(t *testing.T) {
	_, err := pickSpec("ai-tools", "anywhere", func(naming.Spec) bool { return true })
	if err == nil || !strings.Contains(err.Error(), "write it as one of: ai-tools/, ai/tools") {
		t.Fatalf("err = %v", err)
	}
}

// A teammate removed a namespace and the change arrived with a pull.
func TestSyncRemovesSkillsDroppedFromManifest(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend", "backend")
	p := f.reopen()
	reg := p.Manifest.Registry("polarity")
	reg.Install = []string{"backend"}
	for name := range reg.Lock {
		if strings.HasPrefix(name, "frontend-") {
			delete(reg.Lock, name)
		}
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{Frozen: true}))
	for _, rel := range []string{".agents/skills/frontend-forms", ".claude/skills/frontend-forms", ".kiro/skills/frontend-react-patterns"} {
		if f.exists(rel) {
			t.Fatalf("%s still exists", rel)
		}
	}
	if !f.exists(".claude/skills/backend-db-migrations/SKILL.md") {
		t.Fatal("kept skill was removed")
	}
}

func TestSyncNeverTouchesUserSkills(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	testutil.WriteFile(t, f.path(".agents/skills/my-skill/SKILL.md"), "mine", 0o644)
	testutil.WriteFile(t, f.path(".claude/skills/claude-only/SKILL.md"), "mine", 0o644)
	if err := fsx.CopyDir(f.path(".agents/skills/frontend-forms"), f.path(".agents/skills/frontend-forms-copy")); err != nil {
		t.Fatal(err)
	}
	if err := fsx.CopyDir(f.path(".agents/skills/frontend-forms"), f.path(".agents/skills/myforms")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.reopen().Remove([]string{"frontend"}, false); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{}))
	for _, rel := range []string{".agents/skills/my-skill", ".claude/skills/claude-only", ".agents/skills/frontend-forms-copy", ".agents/skills/myforms"} {
		if !f.exists(rel) {
			t.Fatalf("user skill %s was removed", rel)
		}
	}
	if f.exists(".agents/skills/frontend-forms") {
		t.Fatal("removed skill still installed")
	}
}

func TestHiddenFilesInInstalledSkillAreIgnored(t *testing.T) {
	f := newFixture(t, LinkCopy)
	f.add("frontend")
	testutil.WriteFile(t, f.path(".agents/skills/frontend-forms/.DS_Store"), "junk", 0o644)
	testutil.WriteFile(t, f.path(".agents/skills/frontend-forms/.idea/workspace.xml"), "x", 0o644)
	testutil.WriteFile(t, f.path(".claude/skills/frontend-forms/.SKILL.md.swp"), "x", 0o644)
	requireNoActions(t, f.sync(SyncOptions{}))
	if _, err := f.reopen().Remove([]string{"frontend"}, false); err != nil {
		t.Fatal(err)
	}
	if f.exists(".agents/skills/frontend-forms") || f.exists(".claude/skills/frontend-forms") {
		t.Fatal("hidden files made an unmodified skill look edited")
	}
}

func TestRegistrySkillWithHiddenFilesIsRejected(t *testing.T) {
	f := newFixture(t, "")
	f.reg.Write("skills/frontend/frontend-forms/.env.example", "TOKEN=")
	f.reg.Commit("add hidden file")
	_, err := f.reopen().Add("", []string{"frontend/forms"}, false)
	if err == nil || !strings.Contains(err.Error(), "contains hidden file .env.example") {
		t.Fatalf("err = %v", err)
	}
	if f.exists(".agents/skills/frontend-forms") {
		t.Fatal("skill with hidden files installed")
	}
}

func TestSharedFilesInstallAndUpdate(t *testing.T) {
	f := newFixture(t, "")
	testutil.WriteFile(t, filepath.Join(f.reg.Dir, "skills/frontend/_shared/scripts/lint.sh"), "#!/bin/sh\necho v1\n", 0o755)
	f.reg.Link("skills/frontend/frontend-forms/scripts/lint.sh", "../../_shared/scripts/lint.sh")
	f.reg.Commit("share lint script")
	f.add("frontend")

	info, err := os.Lstat(f.path(".agents/skills/frontend-forms/scripts/lint.sh"))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("shared script not installed as an executable file: %v %v", info, err)
	}
	if f.exists(".agents/skills/frontend-_shared") {
		t.Fatal("_shared installed as a skill")
	}
	requireNoActions(t, f.sync(SyncOptions{}))

	testutil.WriteFile(t, filepath.Join(f.reg.Dir, "skills/frontend/_shared/scripts/lint.sh"), "#!/bin/sh\necho v2\n", 0o755)
	f.reg.Commit("lint v2")
	changes, err := f.reopen().Outdated()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Name != "frontend-forms" {
		t.Fatalf("Outdated = %v; only the linking skill should change", changes)
	}
	if _, r, err := f.reopen().Update(nil, false); err != nil {
		t.Fatal(err)
	} else {
		requireClean(t, r)
	}
	if !strings.Contains(f.read(".claude/skills/frontend-forms/scripts/lint.sh"), "v2") {
		t.Fatal("shared change not installed")
	}
}

// The previous version of a skill with shared files is still recognized as
// published content, so a pulled lock replaces it without --force.
func TestSharedFilesPulledLockUpgrade(t *testing.T) {
	f := newFixture(t, "")
	f.reg.Write("skills/backend/_shared/env.md", "env v1")
	f.reg.Link("skills/backend/backend-db-migrations/env.md", "../_shared/env.md")
	f.reg.Commit("share env")
	f.add("backend")
	f.reg.Write("skills/backend/_shared/env.md", "env v2")
	f.reg.Commit("env v2")

	changes, err := f.reopen().Outdated()
	if err != nil || len(changes) != 1 {
		t.Fatalf("Outdated = %v, %v", changes, err)
	}
	p := f.reopen()
	if _, err := p.resolve(resolveOptions{all: true}); err != nil {
		t.Fatal(err)
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{Frozen: true}))
	if f.read(".agents/skills/backend-db-migrations/env.md") != "env v2" {
		t.Fatal("pulled lock not applied")
	}
}

func TestAddAcceptsFolderName(t *testing.T) {
	f := newFixture(t, "")
	f.add("backend/backend-db-migrations")
	if got := f.reopen().Manifest.Registry("polarity").Install; strings.Join(got, ",") != "backend/db-migrations" {
		t.Fatalf("install = %v", got)
	}
	if !f.exists(".agents/skills/backend-db-migrations/SKILL.md") {
		t.Fatal("skill not installed")
	}
}

func TestAddMissingSkillSuggests(t *testing.T) {
	f := newFixture(t, "")
	_, err := f.reopen().Add("", []string{"backend-db/migrations"}, false)
	if err == nil || err.Error() != "registry polarity has no skill backend-db/migrations; did you mean backend/db-migrations?" {
		t.Fatalf("err = %v", err)
	}
	_, err = f.reopen().Add("", []string{"backend/nope"}, false)
	if err == nil || err.Error() != "registry polarity has no skill backend/nope" {
		t.Fatalf("err = %v", err)
	}
}

func TestAddInstalledSkillDeletedUpstreamWarns(t *testing.T) {
	f := newFixture(t, "")
	f.add("backend/db-migrations")
	f.reg.Remove("skills/backend/backend-db-migrations")
	f.reg.Commit("drop db-migrations")
	r, err := f.reopen().Add("", []string{"backend/db-migrations"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fmt.Sprint(r.Warnings), "already installed") {
		t.Fatalf("warnings = %v", r.Warnings)
	}
}

func TestBrowse(t *testing.T) {
	f := newFixture(t, "")
	f.reg.Write("skills/frontend/namespace.yaml", "description: >\n  UI work\n  in React.\n")
	f.reg.Commit("describe frontend")
	f.add("backend")
	l, err := f.reopen().Browse("")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range l.Namespaces {
		got = append(got, fmt.Sprintf("%s|%v|%s", n.Name, n.Installed, n.Description))
		for _, s := range n.Skills {
			got = append(got, fmt.Sprintf("  %s|%v|%s", s.Spec, s.Installed, s.Description))
		}
	}
	want := "backend|true|,  backend/db-migrations|true|Test skill db-migrations.," +
		"frontend|false|UI work in React.,  frontend/forms|false|Test skill forms.,  frontend/react-patterns|false|Test skill react-patterns."
	if strings.Join(got, ",") != want {
		t.Fatalf("Browse = %v", got)
	}
}

func TestFrozenFailsOnDrift(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	p := f.reopen()
	reg := p.Manifest.Registry("polarity")
	reg.Install = append(reg.Install, "backend")
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.reopen().Sync(SyncOptions{Frozen: true}); err == nil || !strings.Contains(err.Error(), "backend is requested but not locked") {
		t.Fatalf("err = %v", err)
	}
	requireClean(t, f.sync(SyncOptions{}))
	if !f.exists(".agents/skills/backend-db-migrations") {
		t.Fatal("drift not resolved")
	}
}

func TestRemoveAgentUnlinks(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	testutil.WriteFile(t, f.path(".kiro/skills/kiro-own/SKILL.md"), "mine", 0o644)
	p := f.reopen()
	p.Manifest.Agents = []string{"claude"}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{}))
	if f.exists(".kiro/skills/frontend-forms") {
		t.Fatal("link for unconfigured agent kept")
	}
	if !f.exists(".kiro/skills/kiro-own") || !f.exists(".claude/skills/frontend-forms") {
		t.Fatal("unrelated entries removed")
	}
}

func TestRemoveNamespaceMemberExcludesIt(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	r, err := f.reopen().Remove([]string{"frontend/forms"}, false)
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	reg := f.reopen().Manifest.Registry("polarity")
	if strings.Join(reg.Exclude, ",") != "frontend/forms" || strings.Join(reg.Install, ",") != "frontend" {
		t.Fatalf("install = %v, exclude = %v", reg.Install, reg.Exclude)
	}
	if f.exists(".agents/skills/frontend-forms") || f.exists(".claude/skills/frontend-forms") {
		t.Fatal("excluded skill still installed")
	}
	f.reg.Skill("frontend", "forms", "forms v2")
	f.reg.Commit("forms v2")
	if _, _, err := f.reopen().Update(nil, false); err != nil {
		t.Fatal(err)
	}
	if f.exists(".agents/skills/frontend-forms") {
		t.Fatal("update reinstalled an excluded skill")
	}
	f.add("frontend/forms")
	if !f.exists(".agents/skills/frontend-forms") || len(f.reopen().Manifest.Registry("polarity").Exclude) != 0 {
		t.Fatal("re-adding didn't lift the exclusion")
	}
}

func TestRemoveKeepsModifiedSkill(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	testutil.WriteFile(t, f.path(".agents/skills/frontend-forms/SKILL.md"), "edited", 0o644)
	r, err := f.reopen().Remove([]string{"frontend"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "kept .agents/skills/frontend-forms") {
		t.Fatalf("warnings = %v", r.Warnings)
	}
	if !f.exists(".agents/skills/frontend-forms") || !f.exists(".claude/skills/frontend-forms") {
		t.Fatal("modified skill or its link was removed")
	}
	if f.exists(".agents/skills/frontend-react-patterns") {
		t.Fatal("unmodified skill kept")
	}
	if _, err := f.reopen().Remove([]string{"backend"}, false); err == nil {
		t.Fatal("removing an uninstalled namespace succeeded")
	}
}

func TestEjectKeepsFilesAndTrackCommitsThem(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	r, err := f.reopen().Eject([]string{"frontend-forms"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hints) != 1 {
		t.Fatalf("hints = %v", r.Hints)
	}
	requireClean(t, f.sync(SyncOptions{}))
	if !f.exists(".agents/skills/frontend-forms/SKILL.md") || !f.exists(".claude/skills/frontend-forms") {
		t.Fatal("ejected skill or its links removed by sync")
	}
	if got := strings.Join(f.lockNames(), ","); got != "frontend-react-patterns" {
		t.Fatalf("lock = %s", got)
	}

	st, err := f.reopen().Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Local) != 3 || st.Local[0].Committed {
		t.Fatalf("local = %+v", st.Local)
	}

	r, err = f.reopen().Track([]string{"frontend-forms"})
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	for _, rel := range []string{".agents/skills/frontend-forms/SKILL.md", ".claude/skills/frontend-forms", ".kiro/skills/frontend-forms"} {
		if ok, _ := gitignore.IsIgnored(f.dir, rel); ok {
			t.Fatalf("%s is still ignored", rel)
		}
	}
	r, err = f.reopen().Untrack([]string{"frontend-forms"})
	if err != nil || len(r.Actions) != 3 {
		t.Fatalf("Untrack = %v, %v", r, err)
	}
	if _, err := f.reopen().Eject([]string{"backend-db-migrations"}, false); err == nil {
		t.Fatal("ejecting an uninstalled skill succeeded")
	}
}

func TestEjectingWholeNamespaceStaysInSync(t *testing.T) {
	f := newFixture(t, "")
	f.add("backend")
	if _, err := f.reopen().Eject([]string{"backend-db-migrations"}, false); err != nil {
		t.Fatal(err)
	}
	requireNoActions(t, f.sync(SyncOptions{Frozen: true}))
	if !f.exists(".agents/skills/backend-db-migrations") {
		t.Fatal("ejected skill removed")
	}
}

func TestTrackRequiresExistingSkill(t *testing.T) {
	f := newFixture(t, "")
	if _, err := f.reopen().Track([]string{"nothing-here"}); err == nil {
		t.Fatal("tracked a missing skill")
	}
	if _, err := f.reopen().Track([]string{"../escape"}); err == nil {
		t.Fatal("accepted a path")
	}
}

func TestAddRefusesExistingForeignFolder(t *testing.T) {
	f := newFixture(t, "")
	testutil.WriteFile(t, f.path(".agents/skills/backend-db-migrations/SKILL.md"), "my own", 0o644)
	if _, err := f.reopen().Add("", []string{"backend"}, false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	if len(f.reopen().Manifest.Registry("polarity").Install) != 0 {
		t.Fatal("failed add was saved")
	}
	r, err := f.reopen().Add("", []string{"backend"}, true)
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	if !strings.Contains(f.read(".agents/skills/backend-db-migrations/SKILL.md"), "db v1") {
		t.Fatal("--force didn't install")
	}
}

func TestCollisionAcrossRegistries(t *testing.T) {
	f := newFixture(t, "")
	other := testutil.NewRegistry(t)
	other.Skill("frontend", "forms", "other forms")
	other.Commit("other")
	f.add("frontend")
	if err := f.reopen().AddRegistry("other", other.Dir, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.reopen().Add("other", []string{"frontend/forms"}, false); err == nil || !strings.Contains(err.Error(), "comes from both") {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.reopen().Add("", []string{"frontend"}, false); err == nil {
		t.Fatal("ambiguous registry accepted")
	}
}

func TestCopyLinkMode(t *testing.T) {
	f := newFixture(t, LinkCopy)
	f.add("frontend")
	info, err := os.Lstat(f.path(".claude/skills/frontend-forms"))
	if err != nil || !info.IsDir() {
		t.Fatalf("expected a copied directory: %v %v", info, err)
	}
	requireNoActions(t, f.sync(SyncOptions{}))

	f.reg.Skill("frontend", "forms", "forms v2")
	f.reg.Commit("forms v2")
	if _, r, err := f.reopen().Update(nil, false); err != nil {
		t.Fatal(err)
	} else {
		requireClean(t, r)
	}
	if !strings.Contains(f.read(".kiro/skills/frontend-forms/SKILL.md"), "forms v2") {
		t.Fatal("copy not refreshed")
	}

	testutil.WriteFile(t, f.path(".claude/skills/frontend-forms/SKILL.md"), "edited copy", 0o644)
	requireProblem(t, f.sync(SyncOptions{}), ".claude/skills/frontend-forms has local changes")

	if _, err := f.reopen().Remove([]string{"frontend/react-patterns"}, false); err != nil {
		t.Fatal(err)
	}
	if f.exists(".claude/skills/frontend-react-patterns") {
		t.Fatal("copy of removed skill kept")
	}
}

func TestSwitchingCopyToSymlinkReplacesCopies(t *testing.T) {
	f := newFixture(t, LinkCopy)
	f.add("frontend")
	p := f.reopen()
	p.Manifest.Link = LinkSymlink
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	requireClean(t, f.sync(SyncOptions{}))
	if target, err := os.Readlink(f.path(".claude/skills/frontend-forms")); err != nil || target != "../../.agents/skills/frontend-forms" {
		t.Fatalf("link = %q, %v", target, err)
	}
}

func TestSyncWorksOfflineFromCache(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	os.RemoveAll(f.reg.Dir)
	os.RemoveAll(f.path(".agents"))
	requireClean(t, f.sync(SyncOptions{Frozen: true}))
	if !f.exists(".agents/skills/frontend-forms/SKILL.md") {
		t.Fatal("not restored from cache")
	}
}

func TestStatus(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	testutil.WriteFile(t, f.path(".agents/skills/frontend-forms/SKILL.md"), "edited", 0o644)
	os.Remove(f.path(".kiro/skills/frontend-react-patterns"))
	st, err := f.reopen().Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Skills) != 2 {
		t.Fatalf("skills = %+v", st.Skills)
	}
	forms, react := st.Skills[0], st.Skills[1]
	if forms.State != StateModified || forms.Links["claude"] != StateOK {
		t.Fatalf("forms = %+v", forms)
	}
	if react.State != StateOK || react.Links["kiro"] != StateMissing {
		t.Fatalf("react = %+v", react)
	}
}

func TestDoctor(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	r, err := f.reopen().Doctor(DoctorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)

	os.Remove(f.path(".claude/skills/frontend-forms"))
	testutil.WriteFile(t, f.path(".agents/skills/my-skill/SKILL.md"), "mine", 0o644)
	r, err = f.reopen().Doctor(DoctorOptions{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	requireProblem(t, r, "out of sync: would link .claude/skills/frontend-forms")
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "mestiere track my-skill") {
		t.Fatalf("warnings = %v", r.Warnings)
	}
	if f.exists(".claude/skills/frontend-forms") {
		t.Fatal("doctor changed files")
	}
	r, _ = f.reopen().Doctor(DoctorOptions{Offline: true, Strict: true})
	requireProblem(t, r, "local skill .agents/skills/my-skill isn't committed")
}

func TestDoctorReportsCommittedInstalls(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	testutil.Git(t, f.dir, "add", "-f", ".agents/skills/frontend-forms")
	r, err := f.reopen().Doctor(DoctorOptions{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "is committed but ignored") {
		t.Fatalf("warnings = %v", r.Warnings)
	}
}

func TestDoctorUnreachableRegistry(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	os.RemoveAll(f.reg.Dir)
	r, err := f.reopen().Doctor(DoctorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	requireProblem(t, r, "registry polarity isn't reachable")
}

func TestRemoveRegistry(t *testing.T) {
	f := newFixture(t, "")
	f.add("frontend")
	r, err := f.reopen().RemoveRegistry("polarity", false)
	if err != nil {
		t.Fatal(err)
	}
	requireClean(t, r)
	if f.reopen().Manifest.Registry("polarity") != nil {
		t.Fatal("registry kept")
	}
	if f.exists(".agents/skills/frontend-forms") || f.exists(".claude/skills/frontend-forms") {
		t.Fatal("registry skills kept")
	}
}

func TestPinnedRef(t *testing.T) {
	testutil.Isolate(t)
	reg := testutil.NewRegistry(t)
	reg.Skill("frontend", "forms", "forms v1")
	reg.Commit("v1")
	testutil.Git(t, reg.Dir, "tag", "v1")
	reg.Skill("frontend", "forms", "forms v2")
	reg.Commit("v2")

	dir := t.TempDir()
	testutil.InitRepo(t, dir)
	p, _, err := Init(dir, []string{"claude"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.AddRegistry("polarity", reg.Dir, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Add("", []string{"frontend"}, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".agents/skills/frontend-forms/SKILL.md"))
	if !strings.Contains(string(data), "forms v1") {
		t.Fatalf("pinned ref not honored:\n%s", data)
	}
}

func TestInitValidation(t *testing.T) {
	testutil.Isolate(t)
	if _, _, err := Init(t.TempDir(), []string{"cursor"}, ""); err == nil {
		t.Error("unknown agent accepted")
	}
	if _, _, err := Init(t.TempDir(), nil, "hardlink"); err == nil {
		t.Error("unknown link mode accepted")
	}
	dir := t.TempDir()
	if _, _, err := Init(dir, []string{"claude"}, ""); err != nil {
		t.Fatal(err)
	}
	p, _, err := Init(dir, nil, "")
	if err != nil || strings.Join(p.Manifest.Agents, ",") != "claude" {
		t.Fatalf("re-init changed agents: %v, %v", p.Manifest.Agents, err)
	}
}
