package cli

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/project"
	"github.com/polarity-dev/mestiere/internal/registry"
)

var agentLabels = map[string]string{"claude": "Claude Code", "kiro": "Kiro"}

// initWizard walks a person through init: agents, a registry, skills. Each
// step has a flag-driven equivalent, which agents and scripts use instead.
func (a *app) initWizard(link string) error {
	a.printBanner("Let's set up agent skills in this repository.")
	root, err := filepath.Abs(a.dir)
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()

	var options []huh.Option[string]
	for _, h := range agents.Harnesses() {
		label := fmt.Sprintf("%-12s %s", agentLabels[h.ID], h.Dir)
		options = append(options, huh.NewOption(label, h.ID).Selected(hasAgentDir(root, h) || hasAgentDir(home, h)))
	}
	var ids []string
	if err := huh.NewMultiSelect[string]().
		Title("Which agents does the team use?").
		Description("Skills always go to .agents/skills, which Codex, Cursor, Gemini CLI and Copilot read.\nThese agents get links to it. Preselected: agents found here or in your home folder.").
		Options(options...).Value(&ids).Run(); err != nil {
		return err
	}
	if ids == nil {
		ids = []string{}
	}
	p, r, err := project.Init(a.dir, ids, link)
	if err := a.report(r, err); err != nil {
		return err
	}

	reg, err := a.wizardRegistry(p)
	if err != nil || reg == "" {
		a.nextSteps(false)
		return err
	}
	picked, err := a.pick(p, reg)
	if errors.Is(err, errNoSelection) {
		a.nextSteps(false)
		return nil
	}
	if err != nil {
		return err
	}
	install := true
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Install %d selected from %s?", len(picked), reg)).
		Description(strings.Join(picked, ", ")).
		Affirmative("Install").Negative("Not now").Value(&install).Run(); err != nil {
		return err
	}
	if install {
		if err := a.report(a.working("Installing skills", func() (*project.Report, error) { return p.Add(reg, picked, false) })); err != nil {
			return err
		}
	}
	a.nextSteps(install)
	return nil
}

func hasAgentDir(base string, h agents.Agent) bool {
	if base == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(base, filepath.FromSlash(path.Dir(h.Dir))))
	return err == nil && info.IsDir()
}

// wizardRegistry returns the registry to pick skills from, asking for one when
// none is configured. An empty name means the person skipped this step.
func (a *app) wizardRegistry(p *project.Project) (string, error) {
	switch regs := p.Manifest.Registries; len(regs) {
	case 0:
	case 1:
		return regs[0].Name, nil
	default:
		var name string
		var options []huh.Option[string]
		for _, reg := range regs {
			options = append(options, huh.NewOption(reg.Name+"  "+reg.URL, reg.Name))
		}
		err := huh.NewSelect[string]().Title("Pick skills from which registry?").Options(options...).Value(&name).Run()
		return name, err
	}
	var url, name string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Registry git URL").
			Description("The repository your team keeps skills in. Leave empty to skip.").
			Placeholder("git@github.com:acme/skills.git").Value(&url),
	)).Run()
	if err != nil || strings.TrimSpace(url) == "" {
		return "", err
	}
	url = strings.TrimSpace(url)
	name = registryName(url)
	if err := huh.NewInput().Title("Name for this registry").Value(&name).Run(); err != nil {
		return "", err
	}
	if err := a.busy("Checking access to "+url, func() error { return registry.Reachable(p.Root, url, "") }); err != nil {
		return "", fmt.Errorf("can't reach %s with your git credentials: %w", url, err)
	}
	if err := p.AddRegistry(name, url, ""); err != nil {
		return "", err
	}
	a.printf("%s\n", a.action(fmt.Sprintf("add registry %s (%s)", name, url)))
	return name, nil
}

var nonName = regexp.MustCompile(`[^a-z0-9_-]+`)

// registryName suggests a registry name from the last segment of its URL.
func registryName(url string) string {
	base := strings.TrimSuffix(strings.TrimRight(url, "/"), ".git")
	if i := strings.LastIndexAny(base, "/:"); i >= 0 {
		base = base[i+1:]
	}
	if name := strings.Trim(nonName.ReplaceAllString(strings.ToLower(base), "-"), "-_"); name != "" {
		return name
	}
	return "skills"
}

func (a *app) nextSteps(installed bool) {
	steps := []string{
		"git add skills.yaml .gitignore && git commit -m \"chore: add agent skills\"",
		"teammates run   mestiere sync",
		"CI runs         mestiere sync --frozen && mestiere doctor --strict",
	}
	if !installed {
		steps = append([]string{"mestiere add        pick skills any time"}, steps...)
	}
	st := a.style()
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, st.box.Render(st.bold.Render("Next steps")+"\n"+strings.Join(steps, "\n")))
}

var errNoSelection = errors.New("nothing selected")
