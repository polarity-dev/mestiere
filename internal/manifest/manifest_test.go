package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `# Project skills
version: 1
agents: [claude, kiro]
registries:
  polarity:
    url: git@github.com:polarity-dev/skills.git # company registry
    ref: main
    install:
      - frontend # whole namespace
      - backend/db-migrations
    lock:
      frontend-forms:
        path: skills/frontend/frontend-forms
        commit: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
        tree: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad(t *testing.T) {
	m, err := Load(write(t, sample))
	if err != nil {
		t.Fatal(err)
	}
	reg := m.Registry("polarity")
	if reg == nil || reg.Ref != "main" || len(reg.Install) != 2 || reg.Lock["frontend-forms"].Path != "skills/frontend/frontend-forms" {
		t.Fatalf("unexpected manifest: %+v", reg)
	}
	if strings.Join(m.Agents, ",") != "claude,kiro" {
		t.Fatalf("agents = %v", m.Agents)
	}
}

func TestSavePreservesComments(t *testing.T) {
	p := write(t, sample)
	m, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	reg := m.Registry("polarity")
	reg.Install = append(reg.Install, "docs")
	reg.Lock["backend-db-migrations"] = LockEntry{Path: "skills/backend/backend-db-migrations", Commit: "c1", Tree: "t1"}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	out := string(data)
	for _, want := range []string{"# Project skills", "# company registry", "- frontend # whole namespace", "- docs", "backend-db-migrations:", "agents: [claude, kiro]"} {
		if !strings.Contains(out, want) {
			t.Errorf("saved manifest is missing %q:\n%s", want, out)
		}
	}
	again, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Registry("polarity").Lock) != 2 {
		t.Fatalf("lock not round-tripped:\n%s", out)
	}
}

func TestSaveNewManifest(t *testing.T) {
	p := filepath.Join(t.TempDir(), FileName)
	m := New(p)
	m.Agents = []string{"claude"}
	m.Registries = append(m.Registries, &Registry{
		Name: "polarity", URL: "https://example.com/r.git",
		Install: []string{"frontend"},
		Lock:    map[string]LockEntry{"frontend-forms": {Path: "skills/frontend/frontend-forms", Commit: "c", Tree: "t"}},
	})
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	out := string(data)
	if !strings.Contains(out, "lock: # written by mestiere") {
		t.Errorf("missing lock comment:\n%s", out)
	}
	if strings.Contains(out, "{") {
		t.Errorf("expected block style:\n%s", out)
	}
	if _, err := Load(p); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyCollectionsGrowIntoBlockStyle(t *testing.T) {
	p := filepath.Join(t.TempDir(), FileName)
	m := New(p)
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	steps := []func(*Manifest){
		func(m *Manifest) {
			m.Registries = append(m.Registries, &Registry{Name: "polarity", URL: "/r", Lock: map[string]LockEntry{}})
		},
		func(m *Manifest) {
			reg := m.Registry("polarity")
			reg.Install = []string{"frontend"}
			reg.Lock["frontend-forms"] = LockEntry{Path: "skills/frontend/frontend-forms", Commit: "c", Tree: "t"}
		},
	}
	for _, step := range steps {
		loaded, err := Load(p)
		if err != nil {
			t.Fatal(err)
		}
		step(loaded)
		if err := loaded.Save(); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(p)
	if strings.Contains(string(data), "{") {
		t.Fatalf("flow mapping written:\n%s", data)
	}
	if _, err := Load(p); err != nil {
		t.Fatalf("%v\n%s", err, data)
	}
}

func TestEmptyLockIsOmitted(t *testing.T) {
	p := write(t, sample)
	m, _ := Load(p)
	m.Registry("polarity").Lock = map[string]LockEntry{}
	m.Registry("polarity").Exclude = nil
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	if strings.Contains(string(data), "lock:") || strings.Contains(string(data), "exclude:") {
		t.Fatalf("empty sections were written:\n%s", data)
	}
}

func TestLoadRejectsUnknownKeysAndVersions(t *testing.T) {
	if _, err := Load(write(t, "version: 1\nagent: [claude]\n")); err == nil {
		t.Error("unknown key accepted")
	}
	if _, err := Load(write(t, "version: 2\n")); err == nil {
		t.Error("unsupported version accepted")
	}
}

func TestFind(t *testing.T) {
	p := write(t, sample)
	nested := filepath.Join(filepath.Dir(p), "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(nested)
	if err != nil || got != p {
		t.Fatalf("Find = %q, %v; want %q", got, err, p)
	}
}
