package naming

import (
	"reflect"
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
		{in: "front-end/x", want: Spec{Namespace: "front-end", Skill: "x"}},
		{in: "frontend-forms", want: Spec{Namespace: "frontend-forms"}},
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

func TestCandidates(t *testing.T) {
	tests := []struct {
		in   string
		want []Spec
	}{
		{in: "frontend", want: []Spec{{Namespace: "frontend"}}},
		{in: "ai-tools/", want: []Spec{{Namespace: "ai-tools"}}},
		{in: "ai-tools/review", want: []Spec{{Namespace: "ai-tools", Skill: "review"}}},
		{in: "ai-code-review", want: []Spec{
			{Namespace: "ai-code-review"},
			{Namespace: "ai", Skill: "code-review"},
			{Namespace: "ai-code", Skill: "review"},
		}},
	}
	for _, tt := range tests {
		got, err := Candidates(tt.in)
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Candidates(%q) = %+v, %v; want %+v", tt.in, got, err, tt.want)
		}
	}
	if _, err := Candidates("-x"); err == nil {
		t.Error("Candidates(-x) succeeded")
	}
}

func TestParsePath(t *testing.T) {
	spec := Spec{Namespace: "ai-code", Skill: "db-migrations"}
	if got, ok := ParsePath(spec.RegistryPath(), false); !ok || got != spec {
		t.Fatalf("ParsePath(%q) = %+v, %v", spec.RegistryPath(), got, ok)
	}
	if got, ok := ParsePath(spec.V1Path(), true); !ok || got != spec {
		t.Fatalf("ParsePath(%q, v1) = %+v, %v", spec.V1Path(), got, ok)
	}
	if _, ok := ParsePath("skills/ai/forms", false); ok {
		t.Fatal("unprefixed folder parsed as current layout")
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
