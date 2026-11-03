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

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/authoring"
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
	dir    string
	quiet  bool
	global bool
	out    io.Writer
	err    io.Writer
	// interactive is false in tests and when stdin isn't a terminal.
	interactive bool
}

func NewRootCmd(out, errOut io.Writer, interactive bool) *cobra.Command {
	a := &app{out: out, err: errOut, interactive: interactive}
	root := &cobra.Command{
		Use:           "mestiere",
		Short:         "Install agent skills from git registries",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version(),
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVarP(&a.dir, "dir", "C", ".", "run as if started in this directory")
	root.PersistentFlags().BoolVarP(&a.quiet, "quiet", "q", false, "only print warnings and errors")
	root.PersistentFlags().BoolVarP(&a.global, "global", "g", false, "act on the skills installed for every project in your home directory")
	root.AddCommand(
		a.initCmd(), a.registryCmd(), a.addCmd(), a.removeCmd(), a.syncCmd(),
		a.updateCmd(), a.outdatedCmd(), a.statusCmd(), a.trackCmd(), a.untrackCmd(),
		a.ejectCmd(), a.doctorCmd(), a.cacheCmd(),
	)
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	root := NewRootCmd(os.Stdout, os.Stderr, isTerminal(os.Stdin) && isTerminal(os.Stdout))
	if err := root.Execute(); err != nil {
		if !errors.Is(err, ErrFailed) {
			fmt.Fprintf(os.Stderr, "mestiere: %v\n", err)
		}
		return 1
	}
	return 0
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
			a.printf("  %s\n", s)
		}
		for _, s := range r.Warnings {
			fmt.Fprintf(a.err, "warning: %s\n", s)
		}
		for _, s := range r.Problems {
			fmt.Fprintf(a.err, "error: %s\n", s)
		}
		for _, s := range r.Hints {
			a.printf("hint: %s\n", s)
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
			if a.global {
				return errors.New("global skills need no init; run `mestiere add -g <namespace>` directly")
			}
			var ids []string
			switch {
			case cmd.Flags().Changed("agents"):
				ids = splitList(agentList)
				if ids == nil {
					ids = []string{}
				}
			case a.interactive:
				if err := huh.NewMultiSelect[string]().
					Title("Which agents should receive skills? (.agents is always included)").
					Options(huh.NewOptions(agents.HarnessIDs()...)...).
					Value(&ids).Run(); err != nil {
					return err
				}
				if ids == nil {
					ids = []string{}
				}
			}
			_, r, err := project.Init(a.dir, ids, link)
			return a.report(r, err)
		},
	}
	cmd.Flags().StringVar(&agentList, "agents", "", fmt.Sprintf("comma-separated agents (%s)", strings.Join(agents.HarnessIDs(), ", ")))
	cmd.Flags().StringVar(&link, "link", "", "how agents get skills: symlink (default) or copy")
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
			return a.report(p.RemoveRegistry(args[0], force))
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
			l, err := p.Browse(name)
			if err != nil {
				return err
			}
			fmt.Fprintf(a.out, "%s at %s\n", l.Registry, registry.Short(l.Commit))
			if len(l.Skills) == 0 {
				fmt.Fprintln(a.out, "  no skills")
			}
			for _, s := range l.Skills {
				mark := ""
				if s.Installed {
					mark = "installed"
				}
				fmt.Fprintf(a.out, "  %-40s %-10s %s\n", s.Spec, mark, s.Description)
			}
			return nil
		},
	}

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

	cmd.AddCommand(add, remove, list, show, initReg, lint, newNamespace, newSkill)
	return cmd
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
				if !a.interactive {
					return errors.New("name at least one namespace or skill to add")
				}
				if args, err = a.pick(p, regName); err != nil {
					return err
				}
			}
			return a.report(p.Add(regName, args, force))
		},
	}
	cmd.Flags().StringVarP(&regName, "registry", "r", "", "registry to install from (required when several are configured)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing directories with the same name")
	return cmd
}

func (a *app) pick(p *project.Project, regName string) ([]string, error) {
	catalog, err := p.Catalog(regName)
	if err != nil {
		return nil, err
	}
	var namespaces []string
	for ns := range catalog {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	var options []huh.Option[string]
	for _, ns := range namespaces {
		options = append(options, huh.NewOption(ns+" (whole namespace)", ns))
		for _, skill := range catalog[ns] {
			options = append(options, huh.NewOption("  "+ns+"/"+skill, ns+"/"+skill))
		}
	}
	if len(options) == 0 {
		return nil, errors.New("the registry has no skills")
	}
	var picked []string
	if err := huh.NewMultiSelect[string]().Title("Select skills to install").Options(options...).Value(&picked).Run(); err != nil {
		return nil, err
	}
	if len(picked) == 0 {
		return nil, errors.New("nothing selected")
	}
	return picked, nil
}

func (a *app) removeCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <namespace | namespace/skill>...",
		Short: "Uninstall namespaces or skills",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(p.Remove(args, force))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "also delete skills with local changes")
	return cmd
}

func (a *app) syncCmd() *cobra.Command {
	var opts project.SyncOptions
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Make installed skills match skills.yaml",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			return a.report(p.Sync(opts))
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
		Use:   "update [target]...",
		Short: "Move locked skills to the latest registry version",
		Long:  "Targets are namespaces, skills, or registry-qualified forms like polarity:frontend. Without targets everything is updated.",
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			changes, r, err := p.Update(args, force)
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
	return &cobra.Command{
		Use:   "outdated",
		Short: "Show skills whose content changed in their registry",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			changes, err := p.Outdated()
			if err != nil {
				return err
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
}

func (a *app) statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show installed, local and global skills",
		Args:  cobra.NoArgs,
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
			for _, d := range st.Drift {
				fmt.Fprintf(a.out, "drift: %s (run `mestiere sync`)\n", d)
			}
			switch {
			case !managed:
				fmt.Fprintf(a.out, "no %s found (run `mestiere init` to manage skills in %s)\n", manifest.FileName, p.Root)
			case len(st.Skills) == 0:
				fmt.Fprintln(a.out, "no registry skills installed")
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
				fmt.Fprintf(a.out, "%-40s %-10s %s@%s  %s%s\n", s.Name, s.State, s.Registry, registry.Short(s.Commit), strings.Join(links, " "), committed)
			}
			if len(st.Local) > 0 {
				fmt.Fprintln(a.out, "\nlocal skills:")
				for _, l := range st.Local {
					if p.Global {
						fmt.Fprintf(a.out, "  ~/%s\n", l.Path)
						continue
					}
					state := "not committed"
					if managed {
						state += " (run `mestiere track " + l.Name + "`)"
					}
					if l.Committed {
						state = "committed"
					}
					fmt.Fprintf(a.out, "  %-40s %s\n", l.Path, state)
				}
			}
			if len(st.Global) > 0 {
				fmt.Fprintln(a.out, "\nglobal skills:")
				for _, g := range st.Global {
					fmt.Fprintf(a.out, "  %s\n", g.Path)
				}
			}
			return nil
		},
	}
}

func (a *app) trackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "track <skill>...",
		Short: "Commit a skill by adding a .gitignore exception",
		Args:  cobra.MinimumNArgs(1),
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
		Use:   "untrack <skill>...",
		Short: "Remove a skill's .gitignore exception",
		Args:  cobra.MinimumNArgs(1),
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
		Use:   "eject <skill>...",
		Short: "Keep an installed skill as a local skill that mestiere no longer manages",
		Args:  cobra.MinimumNArgs(1),
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

func (a *app) doctorCmd() *cobra.Command {
	var opts project.DoctorOptions
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the project without changing anything",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := a.open()
			if err != nil {
				return err
			}
			if err := a.report(p.Doctor(opts)); err != nil {
				return err
			}
			a.printf("no problems found\n")
			return nil
		},
	}
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

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
