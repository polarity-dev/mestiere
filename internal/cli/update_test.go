package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	for _, tt := range []struct {
		latest, current string
		want            bool
	}{
		{"1.2.0", "1.1.0", true},
		{"1.10.0", "1.9.3", true},
		{"2.0.0", "1.99.99", true},
		{"1.1.0", "1.1.0", false},
		{"1.0.9", "1.1.0", false},
		{"1.2.0-rc.1", "1.1.0", false},
		{"", "1.1.0", false},
		{"1.2.0", "dev", false},
	} {
		if got := newer(tt.latest, tt.current); got != tt.want {
			t.Errorf("newer(%q, %q) = %v", tt.latest, tt.current, got)
		}
	}
}

func TestFetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Redirect(w, r, "/releases", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/polarity-dev/mestiere/releases/tag/v1.4.2", http.StatusFound)
	}))
	defer srv.Close()
	got, err := fetchLatest(context.Background(), http.DefaultTransport, srv.URL+"/latest")
	if err != nil || got != "1.4.2" {
		t.Fatalf("fetchLatest = %q, %v", got, err)
	}
	if _, err := fetchLatest(context.Background(), http.DefaultTransport, srv.URL+"/bad"); err == nil {
		t.Fatal("redirect without a tag accepted")
	}
}

func TestUpdateNotice(t *testing.T) {
	got := updateNotice("1.1.0", "1.2.0", "brew upgrade --cask polarity-dev/tap/mestiere", false)
	lines := strings.Split(strings.Trim(got, "\n"), "\n")
	if len(lines) != 7 {
		t.Fatalf("got %d lines:\n%s", len(lines), got)
	}
	width := len([]rune(lines[0]))
	for _, l := range lines {
		if len([]rune(l)) != width {
			t.Fatalf("ragged box:\n%s", got)
		}
	}
	for _, want := range []string{"Update available: 1.1.0 → 1.2.0", "releases/tag/v1.2.0", "Run brew upgrade --cask polarity-dev/tap/mestiere to update"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if colored := updateNotice("1.1.0", "1.2.0", "", true); !strings.Contains(colored, "\x1b[32m1.2.0") || !strings.Contains(colored, "Download it from") {
		t.Fatalf("colored notice:\n%q", colored)
	}
}
