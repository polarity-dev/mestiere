package naming

import (
	"strings"
	"testing"
)

func TestParseSpec(t *testing.T) {
	tests := []struct {
		in      string
		want    Spec
		wantErr bool
	}{
		{in: "frontend", want: Spec{Namespace: "frontend"}},
		{in: "frontend/", want: Spec{Namespace: "frontend"}},
		{in: "frontend/react-patterns", want: Spec{Namespace: "frontend", Skill: "react-patterns"}},
		{in: "frontend-react-patterns", want: Spec{Namespace: "frontend", Skill: "react-patterns"}},
		{in: "front-end/x", wantErr: true},
		{in: "Frontend", wantErr: true},
		{in: "frontend/react--patterns", wantErr: true},
		{in: "frontend/-x", wantErr: true},
		{in: "frontend/a/b", wantErr: true},
		{in: "frontend/" + strings.Repeat("a", 60), wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseSpec(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSpec(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSpec(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestInstalledNameRoundTrip(t *testing.T) {
	spec := Spec{Namespace: "backend", Skill: "db-migrations"}
	ns, skill, ok := Split(spec.InstalledName())
	if !ok || ns != spec.Namespace || skill != spec.Skill {
		t.Fatalf("Split(%q) = %q, %q, %v", spec.InstalledName(), ns, skill, ok)
	}
	if got, ok := ParsePath(spec.RegistryPath()); !ok || got != spec {
		t.Fatalf("ParsePath(%q) = %+v, %v", spec.RegistryPath(), got, ok)
	}
}

func TestSplitRejectsLocalNames(t *testing.T) {
	for _, name := range []string{"myskill", "-x", "x-", "My-Skill"} {
		if _, _, ok := Split(name); ok {
			t.Errorf("Split(%q) succeeded", name)
		}
	}
}

func TestCovers(t *testing.T) {
	ns := Spec{Namespace: "frontend"}
	skill := Spec{Namespace: "frontend", Skill: "forms"}
	other := Spec{Namespace: "backend", Skill: "forms"}
	if !ns.Covers(skill) || !skill.Covers(skill) || ns.Covers(other) || skill.Covers(ns) {
		t.Fatal("unexpected Covers result")
	}
}
