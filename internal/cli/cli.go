// Package cli wires mestiere's commands.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/authoring"
	"github.com/polarity-dev/mestiere/internal/gitx"
	"github.com/polarity-dev/mestiere/internal/manifest"
	"github.com/polarity-dev/mestiere/internal/naming"
	"github.com/polarity-dev/mestiere/internal/project"
	"github.com/polarity-dev/mestiere/internal/registry"
)

// Version is set at build time with -ldflags "-X .../cli.Version=...".
var Version = "dev"

// version falls back to the module version recorded by `go install
// module@version`, which doesn't pass ldflags.
func version() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return Version
}

// ErrFailed signals that problems were already printed.
var ErrFailed = errors.New("failed")

type app struct {
	dir     string
	quiet   bool
	global  bool
	noInput bool
	out     io.Writer
	err     io.Writer
	// interactive is false in tests, without a terminal, in CI and in agent shells.
	interactive bool
	// spinners is true when stderr is a terminal that can redraw a line.
	spinners bool
	st       *styles
}

// human reports whether a person is at the terminal: only then does mestiere
// prompt, or shorten output to fit the screen.
func (a *app) human() bool { return a.interactive && !a.noInput }

// globalCommands are the commands that accept -g, listed in flag errors.
var globalCommands = "add, remove, sync, update, outdated, status, doctor, eject, adopt, prune and the registry add, remove, list and show subcommands"

// scoped gives cmd the -g flag. It is per command, not on the root, so commands
// without a global meaning reject it instead of ignoring it.
func (a *app) scoped(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().BoolVarP(&a.global, "global", "g", false, "act on the skills installed for every project in your home directory")
	return cmd
}

func NewRootCmd(out, errOut io.Writer, interactive bool) *cobra.Command {
	return newApp(out, errOut, interactive).rootCmd()
}

func newApp(out, errOut io.Writer, interactive bool) *app {
	a := &app{out: out, err: errOut, interactive: interactive}
	if f, ok := errOut.(*os.File); ok {
		a.spinners = isTerminal(f)
	}
	return a
}

func (a *app) rootCmd() *cobra.Command {
	out, errOut := a.out, a.err
	root := &cobra.Command{
		Use:           "mestiere",
		Short:         "Install agent skills from git registries",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version(),
		PersistentPreRun: func(*cobra.Command, []string) {
			if !a.human() {
				// Git would otherwise ask for a password or passphrase on the terminal and block agents.
				setDefaultEnv("GIT_TERMINAL_PROMPT", "0")
				// GIT_SSH_COMMAND outranks core.sshCommand, so only set it when the user has neither.
				if cmd, _ := gitx.Run(a.dir, "config", "--get", "core.sshCommand"); cmd == "" {
					setDefaultEnv("GIT_SSH_COMMAND", "ssh -o BatchMode=yes")
				}
			}
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.printBanner("Agent skills from git registries, locked like dependencies.")
			return cmd.Help()
		},
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVarP(&a.dir, "dir", "C", ".", "run as if started in this directory")
	root.PersistentFlags().BoolVarP(&a.quiet, "quiet", "q", false, "only print warnings and errors")
	root.PersistentFlags().BoolVar(&a.noInput, "no-input", false, "never prompt; fail with a hint when an argument is missing (for agents and scripts)")
	root.SetFlagErrorFunc(flagError)
	root.AddCommand(
		a.initCmd(), a.registryCmd(), a.scoped(a.addCmd()), a.scoped(a.removeCmd()), a.scoped(a.syncCmd()),
		a.scoped(a.updateCmd()), a.scoped(a.outdatedCmd()), a.scoped(a.statusCmd()), a.trackCmd(), a.untrackCmd(),
		a.scoped(a.ejectCmd()), a.scoped(a.adoptCmd()), a.scoped(a.pruneCmd()), a.scoped(a.doctorCmd()), a.cacheCmd(),
	)
	return root
}

// flagError adds a suggestion to pflag's unknown-flag errors, which Cobra
// doesn't make on its own.
func flagError(cmd *cobra.Command, err error) error {
	msg := err.Error()
	name, long := strings.CutPrefix(msg, "unknown flag: --")
	if !long {
		short, ok := strings.CutPrefix(msg, "unknown shorthand flag: '")
		if !ok || short == "" {
			return err
		}
		name = short[:1]
	}
	if name == "global" || name == "g" {
		return fmt.Errorf("`%s` has no global form; -g works with %s", cmd.CommandPath(), globalCommands)
	}
	if !long {
		return err
	}
	var names []string
	for _, fs := range []*pflag.FlagSet{cmd.Flags(), cmd.InheritedFlags()} {
		fs.VisitAll(func(f *pflag.Flag) { names = append(names, f.Name) })
	}
	if c, ok := naming.Closest(name, names); ok {
		return fmt.Errorf("%s; did you mean --%s?", msg, c)
	}
	return err
}

// noInputEnv turns prompts off for the whole process.
const noInputEnv = "MESTIERE_NO_INPUT"

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	notify := startUpdateCheck(version(), os.Stderr)
	defer notify()
	interactive := isTerminal(os.Stdin) && isTerminal(os.Stdout) &&
		os.Getenv("CI") == "" && os.Getenv(noInputEnv) == "" && !agents.InAgentShell(os.Getenv)
	a := newApp(os.Stdout, os.Stderr, interactive)
	if err := a.rootCmd().Execute(); err != nil {
		switch {
		case errors.Is(err, ErrFailed):
		case errors.Is(err, huh.ErrUserAborted):
			fmt.Fprintln(os.Stderr, "cancelled")
		case a.human():
			fmt.Fprintln(os.Stderr, a.style().bad.Render("✗ "+err.Error()))
		default:
			fmt.Fprintf(os.Stderr, "mestiere: %v\n", err)
		}
		return 1
	}
	return 0
}

func setDefaultEnv(key, value string) {
	if _, ok := os.LookupEnv(key); !ok {
		_ = os.Setenv(key, value)
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (a *app) open() (*project.Project, error) {
	if a.global {
		return project.OpenGlobal()
	}
	return project.Open(a.dir)
}

func (a *app) printf(format string, args ...any) {
	if !a.quiet {
		fmt.Fprintf(a.out, format, args...)
	}
}

// report prints r and returns ErrFailed when it contains problems.
func (a *app) report(r *project.Report, err error) error {
	if r != nil {
		for _, s := range r.Actions {
			a.printf("%s\n", a.action(s))
		}
		for _, s := range r.Warnings {
			fmt.Fprintln(a.err, a.warnLine(s))
		}
		for _, s := range r.Problems {
			fmt.Fprintln(a.err, a.errorLine(s))
		}
		for _, s := range r.Hints {
			a.printf("%s\n", a.hintLine(s))
		}
	}
	if err != nil {
		return err
	}
	if r != nil && r.Failed() {
		return ErrFailed
	}
	return nil
}

func (a *app) initCmd() *cobra.Command {
	var agentList, link string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create skills.yaml and the .gitignore rule",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var ids []string
			switch {
			case cmd.Flags().Changed("agents"):
				ids = splitList(agentList)
				if ids == nil {
					ids = []string{}
				}
			case a.human():
				return a.initWizard(link)
			}
			_, r, err := project.Init(a.dir, ids, link)
			return a.report(r, err)
		},
	}
	cmd.Flags().StringVar(&agentList, "agents", "", fmt.Sprintf("comma-separated agents (%s)", strings.Join(agents.HarnessIDs(), ", ")))
	cmd.Flags().StringVar(&link, "link", "", "how agents get skills: symlink (default) or copy")
	_ = cmd.RegisterFlagCompletionFunc("agents", completeAgents)
	_ = cmd.RegisterFlagCompletionFunc("link", cobra.FixedCompletions([]string{project.LinkSymlink, project.LinkCopy}, cobra.ShellCompDirectiveNoFileComp))
	return cmd
}

func (a *app) registryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "registry", Short: "Manage and author registries"}
	var ref string
	add := &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Add a registry to this project",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			if err := p.AddRegistry(args[0], args[1], ref); err != nil {
				return err
			}
			a.printf("  add registry %s (%s)\n", args[0], args[1])
			return nil
		},
	}
	add.Flags().StringVar(&ref, "ref", "", "branch, tag or commit to follow (default: the registry's default branch)")

	var force bool
	remove := &cobra.Command{
		Use:   "remove <name>",
		Short: "Uninstall a registry's skills and remove it",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(a.working("Removing the registry", func() (*project.Report, error) { return p.RemoveRegistry(args[0], force) }))
		},
	}
	remove.Flags().BoolVar(&force, "force", false, "also delete skills with local changes")

	list := &cobra.Command{
		Use:   "list",
		Short: "List this project's registries",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			for _, reg := range p.Manifest.Registries {
				ref := reg.Ref
				if ref == "" {
					ref = "(default branch)"
				}
				fmt.Fprintf(a.out, "%s\t%s\t%s\t%d skills\n", reg.Name, reg.URL, ref, len(reg.Lock))
			}
			return nil
		},
	}

	var showJSON bool
	show := &cobra.Command{
		Use:   "show [name]",
		Short: "List the namespaces and skills a registry offers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			var l *project.Listing
			if err := a.busy("Reading the registry", func() (err error) { l, err = p.Browse(name); return }); err != nil {
				return err
			}
			if showJSON {
				return a.printJSON(l)
			}
			a.showListing(l)
			return nil
		},
	}
	show.Flags().BoolVar(&showJSON, "json", false, "print the listing as JSON")

	var regName, description string
	initReg := &cobra.Command{
		Use:   "init [dir]",
		Short: "Scaffold a new registry repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := a.authorDir(args)
			if err := authoring.Init(dir, regName, description); err != nil {
				return err
			}
			a.printf("  create %s\n", filepath.Join(dir, authoring.RegistryFile))
			return nil
		},
	}
	initReg.Flags().StringVar(&regName, "name", "", "registry name (default: directory name)")
	initReg.Flags().StringVar(&description, "description", "", "registry description")

	lint := &cobra.Command{
		Use:   "lint [dir]",
		Short: "Check a registry against the naming and SKILL.md rules",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			issues, err := authoring.Lint(a.authorDir(args))
			if err != nil {
				return err
			}
			for _, i := range issues {
				fmt.Fprintln(a.err, i)
			}
			if authoring.HasErrors(issues) {
				return ErrFailed
			}
			a.printf("registry is valid\n")
			return nil
		},
	}

	var nsDescription, owners string
	newNamespace := &cobra.Command{
		Use:   "new-namespace <namespace>",
		Short: "Create a namespace in a registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := authoring.NewNamespace(a.dir, args[0], nsDescription, splitList(owners)); err != nil {
				return err
			}
			a.printf("  create %s\n", filepath.Join(a.dir, "skills", args[0]))
			return nil
		},
	}
	newNamespace.Flags().StringVar(&nsDescription, "description", "", "namespace description")
	newNamespace.Flags().StringVar(&owners, "owners", "", "comma-separated owners")

	var skillDescription string
	newSkill := &cobra.Command{
		Use:   "new-skill <namespace>/<skill>",
		Short: "Create a skill with a SKILL.md template",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			spec, err := naming.ParseSpec(args[0])
			if err != nil {
				return err
			}
			if err := authoring.NewSkill(a.dir, spec, skillDescription); err != nil {
				return err
			}
			a.printf("  create %s\n", filepath.Join(a.dir, filepath.FromSlash(spec.RegistryPath()), "SKILL.md"))
			return nil
		},
	}
	newSkill.Flags().StringVar(&skillDescription, "description", "", "skill description")

	remove.ValidArgsFunction = a.completeRegistryArg
	show.ValidArgsFunction = a.completeRegistryArg
	cmd.AddCommand(a.scoped(add), a.scoped(remove), a.scoped(list), a.scoped(show), initReg, lint, newNamespace, newSkill)
	return cmd
}

func (a *app) showListing(l *project.Listing) {
	st := a.style()
	if a.human() {
		fmt.Fprintf(a.out, "%s %s\n", st.bold.Render(l.Registry), st.faint.Render("at "+registry.Short(l.Commit)))
	} else {
		fmt.Fprintf(a.out, "%s at %s\n", l.Registry, registry.Short(l.Commit))
	}
	if len(l.Namespaces) == 0 {
		fmt.Fprintln(a.out, "  no skills")
	}
	row := func(name string, namespace, installed bool, desc string) {
		if !a.human() {
			// Agents need the whole description to pick skills.
			mark := ""
			if installed {
				mark = "installed"
			}
			fmt.Fprintf(a.out, "%-42s %-10s %s\n", name, mark, desc)
			return
		}
		mark := " "
		if installed {
			mark = "✓"
		}
		label := fmt.Sprintf("%-40s", name)
		if namespace {
			label = st.bold.Render(label)
		}
		fmt.Fprintf(a.out, "%s %s %s\n", mark, label, st.faint.Render(summary(desc, 72)))
	}
	for _, n := range l.Namespaces {
		indent := ""
		// Plain skills, from a registry with the vercel/skills layout, have no namespace row.
		if n.Name != "" {
			row(n.Name, true, n.Installed, n.Description)
			indent = "  "
		}
		for _, s := range n.Skills {
			row(indent+s.Spec.String(), false, s.Installed, s.Description)
		}
	}
}

// authorDir resolves the registry directory for authoring commands, which
// work on a registry checkout rather than a project.
func (a *app) authorDir(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return a.dir
}

func (a *app) addCmd() *cobra.Command {
	var regName string
	var force bool
	cmd := &cobra.Command{
		Use:   "add [namespace | namespace/skill]...",
		Short: "Install namespaces or skills from a registry",
		Example: `  mestiere add frontend
  mestiere add backend/db-migrations
  mestiere add -r polarity frontend backend/db-migrations`,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				if !a.human() {
					show := "mestiere registry show"
					if a.global {
						show += " -g"
					}
					return fmt.Errorf("name at least one namespace or skill to add; `%s` lists them", show)
				}
				if args, err = a.pick(p, regName); err != nil {
					return err
				}
			}
			return a.report(a.working("Installing skills", func() (*project.Report, error) { return p.Add(regName, args, force) }))
		},
	}
	cmd.Flags().StringVarP(&regName, "registry", "r", "", "registry to install from (required when several are configured)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing directories with the same name")
	cmd.ValidArgsFunction = a.completeOffered(&regName)
	_ = cmd.RegisterFlagCompletionFunc("registry", a.completeRegistries)
	return cmd
}

func (a *app) pick(p *project.Project, regName string) ([]string, error) {
	var l *project.Listing
	err := a.busy("Reading the registry", func() (err error) { l, err = p.Browse(regName); return })
	if err != nil {
		return nil, err
	}
	var options []huh.Option[string]
	label := func(name string, installed bool, desc string) string {
		if installed {
			desc = "(installed) " + desc
		}
		return fmt.Sprintf("%-30s %s", name, summary(desc, 72))
	}
	for _, n := range l.Namespaces {
		indent := ""
		if n.Name != "" {
			options = append(options, huh.NewOption(label(n.Name, n.Installed, n.Description), n.Name))
			indent = "    "
		}
		for _, s := range n.Skills {
			options = append(options, huh.NewOption(label(indent+s.Spec.Skill, s.Installed, s.Description), s.Spec.String()))
		}
	}
	if len(options) == 0 {
		return nil, errors.New("the registry has no skills")
	}
	var picked []string
	err = huh.NewMultiSelect[string]().
		Title("Select skills to install from " + l.Registry).
		Description("A namespace installs all of it and follows it on update.\nspace selects · / filters · enter installs").
		Options(options...).Value(&picked).Filterable(true).Height(min(len(options)+4, 20)).Run()
	if err != nil {
		return nil, err
	}
	if len(picked) == 0 {
		return nil, errNoSelection
	}
	return picked, nil
}

func (a *app) removeCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:               "remove <namespace | namespace/skill>...",
		Aliases:           []string{"rm", "uninstall"},
		Short:             "Uninstall namespaces or skills",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: a.completeInstalled,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(a.working("Removing skills", func() (*project.Report, error) { return p.Remove(args, force) }))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "also delete skills with local changes")
	return cmd
}

func (a *app) syncCmd() *cobra.Command {
	var opts project.SyncOptions
	cmd := &cobra.Command{
		Use:     "sync",
		Aliases: []string{"install"},
		Short:   "Make installed skills match skills.yaml",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(a.working("Syncing skills", func() (*project.Report, error) { return p.Sync(opts) }))
		},
	}
	cmd.Flags().BoolVar(&opts.Frozen, "frozen", false, "fail if install and lock disagree instead of re-resolving (for CI)")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite local changes")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "print what would change")
	return cmd
}

func (a *app) updateCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:               "update [target]...",
		Aliases:           []string{"upgrade"},
		ValidArgsFunction: a.completeInstalled,
		Short:             "Move locked skills to the latest registry version",
		Long:              "Targets are namespaces, skills, or registry-qualified forms like polarity:frontend. Without targets everything is updated.",
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			var changes []project.Change
			r, err := a.working("Updating skills", func() (r *project.Report, err error) {
				changes, r, err = p.Update(args, force)
				return
			})
			for _, c := range changes {
				a.printf("  %s\n", c)
			}
			if err == nil && len(changes) == 0 {
				a.printf("everything is up to date\n")
			}
			return a.report(r, err)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite local changes")
	return cmd
}

func (a *app) outdatedCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Show skills whose content changed in their registry",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			var changes []project.Change
			if err := a.busy("Checking registries", func() (err error) { changes, err = p.Outdated(); return }); err != nil {
				return err
			}
			if asJSON {
				return a.printJSON(nonNil(changes))
			}
			if len(changes) == 0 {
				fmt.Fprintln(a.out, "everything is up to date")
			}
			for _, c := range changes {
				fmt.Fprintln(a.out, c)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the changes as JSON")
	return cmd
}

// statusJSON is the --json form of status.
type statusJSON struct {
	Root    string `json:"root"`
	Managed bool   `json:"managed"`
	Global  bool   `json:"global"`
	*project.Status
}

func (a *app) statusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"ls", "list"},
		Short:   "Show installed, local and global skills",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			managed := err == nil
			if errors.Is(err, manifest.ErrNotFound) {
				p, err = project.Unmanaged(a.dir)
			}
			if err != nil {
				return err
			}
			st, err := p.Status()
			if err != nil {
				return err
			}
			if asJSON {
				return a.printJSON(statusJSON{Root: p.Root, Managed: managed, Global: p.Global, Status: st})
			}
			a.showStatus(p, st, managed)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the status as JSON")
	return cmd
}

func (a *app) showStatus(p *project.Project, st *project.Status, managed bool) {
	stl := a.style()
	for _, d := range st.Drift {
		line := fmt.Sprintf("drift: %s (run `mestiere sync`)", d)
		if a.human() {
			line = stl.warn.Render("⚠ " + line)
		}
		fmt.Fprintln(a.out, line)
	}
	switch {
	case !managed:
		fmt.Fprintf(a.out, "no %s found (run `mestiere init` to manage skills in %s)\n", manifest.FileName, p.Root)
	case len(st.Skills) == 0:
		fmt.Fprintln(a.out, "no registry skills installed")
	}
	if len(st.Skills) > 0 && a.human() {
		fmt.Fprintln(a.out, stl.bold.Render("Registry skills"))
	}
	for _, s := range st.Skills {
		var links []string
		for _, id := range sortedKeys(s.Links) {
			links = append(links, id+":"+string(s.Links[id]))
		}
		committed := ""
		if s.Committed {
			committed = "  committed"
		}
		if !a.human() {
			fmt.Fprintf(a.out, "%-40s %-10s %s@%s  %s%s\n", s.Name, s.State, s.Registry, registry.Short(s.Commit), strings.Join(links, " "), committed)
			continue
		}
		mark, state := stl.ok.Render("✓"), fmt.Sprintf("%-10s", s.State)
		if s.State != project.StateOK {
			mark, state = stl.warn.Render("⚠"), stl.warn.Render(state)
		}
		fmt.Fprintf(a.out, "  %s %-40s %s %s\n", mark, s.Name, state,
			stl.faint.Render(fmt.Sprintf("%s@%s  %s%s", s.Registry, registry.Short(s.Commit), strings.Join(links, " "), committed)))
	}
	if len(st.Shadowed) > 0 {
		fmt.Fprintln(a.out)
		a.heading("also installed globally:", "Also installed globally")
		for _, sh := range st.Shadowed {
			loads := map[agents.Precedence]string{
				agents.GlobalWins:  "loads the global copy",
				agents.ProjectWins: "loads the project copy",
				agents.Unknown:     "may load either copy",
			}[sh.Loads]
			line := fmt.Sprintf("  %-40s %s %s (%s)", sh.Name, sh.Agent, loads, sh.Path)
			if a.human() && sh.Loads != agents.ProjectWins {
				line = stl.warn.Render(line)
			}
			fmt.Fprintln(a.out, line)
		}
	}
	if len(st.Local) > 0 {
		fmt.Fprintln(a.out)
		a.heading("local skills:", "Local skills")
		for _, l := range st.Local {
			if p.Global {
				origin := l.Origin
				if l.Target != "" {
					origin += " → " + l.Target
				}
				if a.human() {
					origin = stl.faint.Render(origin)
				}
				fmt.Fprintf(a.out, "  %-40s %s\n", "~/"+l.Path, origin)
				continue
			}
			state := "not committed"
			if managed {
				state += " (run `mestiere track " + l.Name + "`)"
			}
			if l.Committed {
				state = "committed"
			}
			if a.human() {
				state = stl.faint.Render(state)
			}
			fmt.Fprintf(a.out, "  %-40s %s\n", l.Path, state)
		}
	}
	if len(st.Global) > 0 {
		fmt.Fprintln(a.out)
		a.heading("global skills:", "Global skills")
		for _, g := range st.Global {
			fmt.Fprintf(a.out, "  %s\n", g.Path)
		}
	}
}

func (a *app) trackCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "track <skill>...",
		Short:             "Commit a skill by adding a .gitignore exception",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: a.completeSkillFolders,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(p.Track(args))
		},
	}
}

func (a *app) untrackCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "untrack <skill>...",
		Short:             "Remove a skill's .gitignore exception",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: a.completeSkillFolders,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(p.Untrack(args))
		},
	}
}

func (a *app) ejectCmd() *cobra.Command {
	var track bool
	cmd := &cobra.Command{
		Use:               "eject <skill>...",
		Short:             "Keep an installed skill as a local skill that mestiere no longer manages",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: a.completeLocked,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(p.Eject(args, track))
		},
	}
	cmd.Flags().BoolVar(&track, "track", false, "also commit the ejected skills")
	return cmd
}

func (a *app) adoptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "adopt <skill>...",
		Short: "Manage a skill you installed another way, when a registry publishes the same content",
		Long: "adopt looks for a configured registry that publishes a skill with the same name and content as the folder on disk, " +
			"then installs it from there. The folder becomes the managed copy, and links replace other copies.",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: a.completeUnmanaged,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(a.working("Looking for the publisher", func() (*project.Report, error) { return p.Adopt(args) }))
		},
	}
}

func (a *app) pruneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune -g [skill]...",
		Short: "Move global skills that mestiere doesn't manage to the trash",
		Long: "prune moves every copy and link of the named global skills to a dated folder under " +
			"~/.local/share/mestiere/trash. Nothing is deleted: move a folder back to restore it. " +
			"Run `mestiere doctor -g` first to see what is unmanaged.",
		ValidArgsFunction: a.completeUnmanaged,
		RunE: func(_ *cobra.Command, args []string) error {
			if !a.global {
				return errors.New("prune cleans up global skills; run `mestiere prune -g <skill>`")
			}
			p, err := a.open()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				names := p.UnmanagedNames()
				if len(names) == 0 {
					a.printf("every global skill is managed by mestiere\n")
					return nil
				}
				if !a.human() {
					return fmt.Errorf("name the skills to prune; unmanaged global skills: %s", strings.Join(names, ", "))
				}
				if args, err = a.pickUnmanaged(names); err != nil {
					return err
				}
			}
			return a.report(p.Prune(args))
		},
	}
	return cmd
}

// pickUnmanaged lets a person choose global skills to prune, then confirm.
func (a *app) pickUnmanaged(names []string) ([]string, error) {
	var picked []string
	err := huh.NewMultiSelect[string]().
		Title("Move which global skills to the trash?").
		Description("They aren't managed by mestiere. Nothing is deleted; each can be moved back.\nspace selects · / filters · enter continues").
		Options(huh.NewOptions(names...)...).Value(&picked).Filterable(true).Run()
	if err != nil {
		return nil, err
	}
	if len(picked) == 0 {
		return nil, errNoSelection
	}
	ok := false
	if err := huh.NewConfirm().Title(fmt.Sprintf("Move %d skills to the trash?", len(picked))).
		Description(strings.Join(picked, ", ")).Affirmative("Move").Negative("Cancel").Value(&ok).Run(); err != nil {
		return nil, err
	}
	if !ok {
		return nil, huh.ErrUserAborted
	}
	return picked, nil
}

type doctorJSON struct {
	OK       bool     `json:"ok"`
	Problems []string `json:"problems"`
	Warnings []string `json:"warnings"`
}

func (a *app) doctorCmd() *cobra.Command {
	var opts project.DoctorOptions
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the project without changing anything",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			r, err := a.working("Checking the project", func() (*project.Report, error) { return p.Doctor(opts) })
			if asJSON && err == nil {
				if err := a.printJSON(doctorJSON{OK: !r.Failed(), Problems: nonNil(r.Problems), Warnings: nonNil(r.Warnings)}); err != nil {
					return err
				}
				if r.Failed() {
					return ErrFailed
				}
				return nil
			}
			if err := a.report(r, err); err != nil {
				return err
			}
			if a.human() {
				a.printf("%s\n", a.style().ok.Render("✓ no problems found"))
			} else {
				a.printf("no problems found\n")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the findings as JSON; the exit status still reports problems")
	cmd.Flags().BoolVar(&opts.Strict, "strict", false, "fail on local skills that aren't committed")
	cmd.Flags().BoolVar(&opts.Offline, "offline", false, "skip network checks")
	return cmd
}

func (a *app) cacheCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cache", Short: "Inspect or clear the registry cache"}
	dir := &cobra.Command{
		Use:   "dir",
		Short: "Print the cache directory",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			d, err := registry.CacheDir()
			if err != nil {
				return err
			}
			fmt.Fprintln(a.out, d)
			return nil
		},
	}
	info := &cobra.Command{
		Use:   "info",
		Short: "Show the cache directory and its size",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			d, err := registry.CacheDir()
			if err != nil {
				return err
			}
			size, err := registry.CacheSize()
			if err != nil {
				return err
			}
			fmt.Fprintf(a.out, "%s\t%.1f MB\n", d, float64(size)/(1<<20))
			return nil
		},
	}
	clean := &cobra.Command{
		Use:   "clean",
		Short: "Delete all registry mirrors; they are re-fetched when needed",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			removed, err := registry.CleanCache()
			for _, r := range removed {
				a.printf("  remove %s\n", r)
			}
			if err == nil && len(removed) == 0 {
				a.printf("cache is empty\n")
			}
			return err
		},
	}
	cmd.AddCommand(dir, info, clean)
	return cmd
}

// working runs a report-returning operation behind a spinner.
func (a *app) working(title string, fn func() (*project.Report, error)) (r *project.Report, err error) {
	busyErr := a.busy(title, func() error { r, err = fn(); return nil })
	if err == nil {
		err = busyErr
	}
	return r, err
}

// summary cuts a description to max runes.
func summary(desc string, max int) string {
	if r := []rune(desc); len(r) > max {
		desc = strings.TrimSpace(string(r[:max-1])) + "…"
	}
	return desc
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// nonNil makes an empty result print as [] rather than null in JSON.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
