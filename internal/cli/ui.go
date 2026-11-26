package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// banner is the mestiere wordmark in figlet's "small" letters.
const banner = `                 _    _
 _ __   ___  ___| |_ (_) ___  _ _  ___
| '  \ / -_)(_-<|  _|| |/ -_)| '_|/ -_)
|_|_|_|\___|/__/ \__||_|\___||_|  \___|`

// bannerWidth is the widest banner line; narrower terminals get the name only.
const bannerWidth = 40

// styles render output for people. The brand is black and white, so emphasis
// is weight and dimming; color is kept for warnings and errors.
type styles struct {
	bold, faint, ok, warn, bad, box lipgloss.Style
}

func newStyles(w io.Writer) styles {
	r := lipgloss.NewRenderer(w)
	// No style depends on the background, and setting it skips the terminal query lipgloss would send.
	r.SetHasDarkBackground(true)
	if os.Getenv("NO_COLOR") != "" {
		r.SetColorProfile(0)
	}
	s := r.NewStyle()
	return styles{
		bold:  s.Bold(true),
		faint: s.Faint(true),
		ok:    s.Bold(true),
		warn:  s.Foreground(lipgloss.Color("3")),
		bad:   s.Foreground(lipgloss.Color("1")).Bold(true),
		box:   s.Border(lipgloss.RoundedBorder()).Padding(0, 1),
	}
}

func (a *app) style() styles {
	if a.st == nil {
		st := newStyles(a.out)
		a.st = &st
	}
	return *a.st
}

// action renders one report line such as "install .agents/skills/x": the verb
// in bold, the rest dimmed, behind a check mark.
func (a *app) action(s string) string {
	if !a.human() {
		return "  " + s
	}
	verb, rest, _ := strings.Cut(s, " ")
	st := a.style()
	return fmt.Sprintf("  %s %s %s", st.ok.Render("✓"), st.bold.Render(fmt.Sprintf("%-8s", verb)), st.faint.Render(rest))
}

func (a *app) warnLine(s string) string {
	if !a.human() {
		return "warning: " + s
	}
	return a.style().warn.Render("⚠ " + s)
}

func (a *app) errorLine(s string) string {
	if !a.human() {
		return "error: " + s
	}
	return a.style().bad.Render("✗ " + s)
}

func (a *app) hintLine(s string) string {
	if !a.human() {
		return "hint: " + s
	}
	return a.style().faint.Render("→ " + s)
}

// heading prints a section title for people; plain output keeps its own text.
func (a *app) heading(plain, styled string) {
	if a.human() {
		fmt.Fprintln(a.out, a.style().bold.Render(styled))
		return
	}
	fmt.Fprintln(a.out, plain)
}

// printBanner shows the wordmark to people with a wide enough terminal.
func (a *app) printBanner(tagline string) {
	if !a.human() || a.quiet {
		return
	}
	st := a.style()
	if terminalWidth() >= bannerWidth {
		fmt.Fprintln(a.out, st.bold.Render(banner))
	} else {
		fmt.Fprintln(a.out, st.bold.Render("mestiere"))
	}
	fmt.Fprintf(a.out, "%s\n\n", st.faint.Render(tagline))
}

// busy runs fn behind a spinner on stderr when a person is watching. Agents
// and pipes get no animation, so their output stays a clean log.
func (a *app) busy(title string, fn func() error) error {
	if !a.human() || a.quiet || !a.spinners {
		return fn()
	}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(80 * time.Millisecond)
		defer t.Stop()
		for i := 0; ; i++ {
			fmt.Fprintf(a.err, "\r%s %s", frames[i%len(frames)], title)
			select {
			case <-done:
				fmt.Fprint(a.err, "\r\033[K")
				return
			case <-t.C:
			}
		}
	}()
	err := fn()
	close(done)
	wg.Wait()
	return err
}

func terminalWidth() int {
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		return 0
	}
	return w
}

// printJSON writes v as indented JSON, the stable format for scripts and agents.
func (a *app) printJSON(v any) error {
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
