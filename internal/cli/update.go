package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/polarity-dev/mestiere/internal/registry"
)

const (
	releasesURL   = "https://github.com/polarity-dev/mestiere/releases"
	checkInterval = 24 * time.Hour
	// fetchWait is how long a finished command waits for a check in flight.
	// A slower answer is dropped and the next run checks again.
	fetchWait = 300 * time.Millisecond
)

type updateState struct {
	Checked time.Time `json:"checked"`
	Latest  string    `json:"latest,omitempty"`
}

// startUpdateCheck looks for a newer release at most once a day, in the
// background, and returns a function that prints a notice to w when one exists.
// It does nothing for development builds, outside a terminal, in CI, or when
// MESTIERE_NO_UPDATE_CHECK or NO_UPDATE_NOTIFIER is set.
func startUpdateCheck(current string, w *os.File) func() {
	// Shell completion runs mestiere __complete with stderr on the user's prompt.
	completing := len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "__complete")
	if _, ok := parseVersion(current); !ok || completing || !isTerminal(w) || os.Getenv("CI") != "" ||
		os.Getenv("MESTIERE_NO_UPDATE_CHECK") != "" || os.Getenv("NO_UPDATE_NOTIFIER") != "" {
		return func() {}
	}
	cache, err := registry.CacheDir()
	if err != nil {
		return func() {}
	}
	statePath := filepath.Join(cache, "update-check.json")
	var state updateState
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &state)
	}
	var fresh chan string
	if time.Since(state.Checked) > checkInterval {
		fresh = make(chan string, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			latest, err := fetchLatest(ctx, http.DefaultTransport, releasesURL+"/latest")
			next := updateState{Checked: time.Now(), Latest: state.Latest}
			if err == nil {
				next.Latest = latest
			}
			if data, err := json.Marshal(next); err == nil && os.MkdirAll(cache, 0o755) == nil {
				_ = os.WriteFile(statePath, data, 0o644)
			}
			fresh <- next.Latest
		}()
	}
	return func() {
		latest := state.Latest
		if fresh != nil {
			select {
			case l := <-fresh:
				latest = l
			case <-time.After(fetchWait):
			}
		}
		if newer(latest, current) {
			fmt.Fprint(w, updateNotice(current, latest, upgradeCommand(), os.Getenv("NO_COLOR") == ""))
		}
	}
}

// fetchLatest reads the latest release tag from the redirect GitHub answers
// /releases/latest with, which unlike the REST API has no rate limit.
func fetchLatest(ctx context.Context, rt http.RoundTripper, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	tag := strings.TrimPrefix(path.Base(resp.Header.Get("Location")), "v")
	if resp.StatusCode/100 != 3 {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}
	if _, ok := parseVersion(tag); !ok {
		return "", fmt.Errorf("unexpected release %q", tag)
	}
	return tag, nil
}

// parseVersion accepts release versions only: X.Y.Z, no pre-release suffix.
func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func newer(latest, current string) bool {
	l, ok1 := parseVersion(latest)
	c, ok2 := parseVersion(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// upgradeCommand matches how this binary was installed.
func upgradeCommand() string {
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil && strings.Contains(real, "/Caskroom/") {
			return "brew upgrade --cask polarity-dev/tap/mestiere"
		}
	}
	if Version == "dev" {
		return "go install github.com/polarity-dev/mestiere/cmd/mestiere@latest"
	}
	return ""
}

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

// updateNotice draws the notice in a rounded box, like pnpm and npm do.
func updateNotice(current, latest, command string, color bool) string {
	paint := func(code, s string) string {
		if !color {
			return s
		}
		return code + s + ansiReset
	}
	type line struct{ plain, styled string }
	lines := []line{{
		"Update available: " + current + " → " + latest,
		"Update available: " + paint(ansiRed, current) + " → " + paint(ansiGreen, latest),
	}, {
		"Changelog: " + releasesURL + "/tag/v" + latest,
		"Changelog: " + paint(ansiCyan, releasesURL+"/tag/v"+latest),
	}}
	if command != "" {
		lines = append(lines, line{"Run " + command + " to update", "Run " + paint(ansiCyan, command) + " to update"})
	} else {
		lines = append(lines, line{"Download it from " + releasesURL + "/latest", "Download it from " + paint(ansiCyan, releasesURL+"/latest")})
	}
	width := 0
	for _, l := range lines {
		width = max(width, utf8.RuneCountInString(l.plain))
	}
	width += 6
	var b strings.Builder
	border := func(s string) { b.WriteString(paint(ansiYellow, s)) }
	row := func(content string, pad int) {
		b.WriteString("   ")
		border("│")
		left := pad / 2
		b.WriteString(strings.Repeat(" ", left) + content + strings.Repeat(" ", pad-left))
		border("│")
		b.WriteString("\n")
	}
	b.WriteString("\n   ")
	border("╭" + strings.Repeat("─", width) + "╮")
	b.WriteString("\n")
	row("", width)
	for _, l := range lines {
		row(l.styled, width-utf8.RuneCountInString(l.plain))
	}
	row("", width)
	b.WriteString("   ")
	border("╰" + strings.Repeat("─", width) + "╯")
	b.WriteString("\n\n")
	return b.String()
}
