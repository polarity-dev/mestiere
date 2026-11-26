package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/project"
)

type completeFunc = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

const noFiles = cobra.ShellCompDirectiveNoFileComp

// completionProject opens the project offline, so pressing Tab never fetches.
func (a *app) completionProject() *project.Project {
	p, err := a.open()
	if err != nil {
		return nil
	}
	p.Offline = true
	return p
}

// matching keeps the candidates that start with toComplete and aren't already
// on the command line. A candidate may carry a "\tdescription" suffix.
func matching(cands, args []string, toComplete string) []string {
	given := map[string]bool{}
	for _, arg := range args {
		given[arg] = true
	}
	var out []string
	for _, c := range cands {
		value, _, _ := strings.Cut(c, "\t")
		if strings.HasPrefix(value, toComplete) && !given[value] {
			out = append(out, c)
		}
	}
	return out
}

// completeOffered lists what the cached registries offer, with descriptions.
func (a *app) completeOffered(regName *string) completeFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		p := a.completionProject()
		if p == nil {
			return nil, noFiles
		}
		var cands []string
		for _, reg := range p.Manifest.Registries {
			if *regName != "" && reg.Name != *regName {
				continue
			}
			l, err := p.Browse(reg.Name)
			if err != nil {
				continue
			}
			for _, n := range l.Namespaces {
				if n.Name != "" {
					cands = append(cands, n.Name+"\t"+summary(n.Description, 60))
				}
				for _, s := range n.Skills {
					cands = append(cands, s.Spec.String()+"\t"+summary(s.Description, 60))
				}
			}
		}
		return matching(cands, args, toComplete), noFiles
	}
}

func (a *app) completeInstalled(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	p := a.completionProject()
	if p == nil {
		return nil, noFiles
	}
	return matching(p.Installed(), args, toComplete), noFiles
}

// completeLocked lists installed skills by their installed name.
func (a *app) completeLocked(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	p := a.completionProject()
	if p == nil {
		return nil, noFiles
	}
	var names []string
	for _, reg := range p.Manifest.Registries {
		for name := range reg.Lock {
			names = append(names, name)
		}
	}
	return matching(names, args, toComplete), noFiles
}

// completeSkillFolders lists installed and local skills, which track and untrack accept.
func (a *app) completeSkillFolders(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	names, _ := a.completeLocked(cmd, args, toComplete)
	if p := a.completionProject(); p != nil {
		if st, err := p.Status(); err == nil {
			var local []string
			for _, l := range st.Local {
				local = append(local, l.Name)
			}
			names = append(names, matching(local, append(args, names...), toComplete)...)
		}
	}
	return names, noFiles
}

// completeUnmanaged lists skills mestiere doesn't manage, for adopt and prune.
func (a *app) completeUnmanaged(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	p := a.completionProject()
	if p == nil {
		return nil, noFiles
	}
	return matching(p.UnmanagedNames(), args, toComplete), noFiles
}

// completeRegistries lists the configured registries, for --registry.
func (a *app) completeRegistries(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	p := a.completionProject()
	if p == nil {
		return nil, noFiles
	}
	var names []string
	for _, reg := range p.Manifest.Registries {
		names = append(names, reg.Name+"\t"+reg.URL)
	}
	return matching(names, nil, toComplete), noFiles
}

// completeRegistryArg completes the single registry name a command takes.
func (a *app) completeRegistryArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, noFiles
	}
	return a.completeRegistries(cmd, args, toComplete)
}

// completeAgents completes the last item of a comma-separated agent list.
func completeAgents(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	done, last := "", toComplete
	if i := strings.LastIndex(toComplete, ","); i >= 0 {
		done, last = toComplete[:i+1], toComplete[i+1:]
	}
	var out []string
	for _, id := range agents.HarnessIDs() {
		if strings.HasPrefix(id, last) && !strings.Contains(","+done, ","+id+",") {
			out = append(out, done+id)
		}
	}
	return out, noFiles | cobra.ShellCompDirectiveNoSpace
}
