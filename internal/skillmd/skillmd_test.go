package skillmd

import "testing"

func TestParse(t *testing.T) {
	fm, err := Parse([]byte("\ufeff---\r\nname: a-b\r\ndescription: \"Does: things\"\r\n---\r\n# Body\n"))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "a-b" || fm.Description != "Does: things" {
		t.Fatalf("got %+v", fm)
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"# no frontmatter", "---\nname: x\n", "---\nname: [\n---\n"} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("Parse(%q) succeeded", in)
		}
	}
}

func TestTemplateParses(t *testing.T) {
	fm, err := Parse([]byte(Template("ns-skill", "Use when: things happen")))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "ns-skill" || fm.Description != "Use when: things happen" {
		t.Fatalf("got %+v", fm)
	}
}
