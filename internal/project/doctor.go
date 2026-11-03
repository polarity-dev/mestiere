package project

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/polarity-dev/mestiere/internal/agents"
	"github.com/polarity-dev/mestiere/internal/gitignore"
	"github.com/polarity-dev/mestiere/internal/gitx"
	"github.com/polarity-dev/mestiere/internal/registry"
)

type DoctorOptions struct {
	// Strict reports uncommitted local skills as problems instead of warnings.
	Strict bool
	// Offline skips registry reachability checks and never fetches.
	Offline bool
}

var commitRe = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

// Doctor reports everything that would make sync do work or fail, without
// changing anything.
func (p *Project) Doctor(opts DoctorOptions) (*Report, error) {
	r := &Report{}
	if err := gitx.Available(); err != nil {
		r.Problem("%v", err)
		return r, nil
	}
	if opts.Offline {
		p.Offline = true
	} else {
		for _, reg := range p.Manifest.Registries {
			ref := reg.Ref
			// ls-remote matches ref names only, so a pinned commit can't be checked remotely.
			if commitRe.MatchString(ref) {
				ref = ""
			}
			if err := registry.Reachable(p.Root, p.resolveURL(reg.URL), ref); err != nil {
				r.Problem("registry %s isn't reachable: %v", reg.Name, err)
			}
		}
	}

	inRepo := p.inRepo()
	if inRepo {
		for _, probe := range p.ignoreProbes() {
			if ok, err := gitignore.IsIgnored(p.Root, probe); err == nil && !ok {
				r.Problem("%s isn't ignored; run `mestiere init` or add %q to .gitignore", path.Dir(probe), gitignore.Rule)
			}
		}
		p.checkTrackedInstalls(r)
	} else if !p.Global {
		r.Warn("not inside a git repository; .gitignore checks skipped")
	}

	sr, err := p.Sync(SyncOptions{DryRun: true})
	if err != nil {
		r.Problem("%v", err)
	} else {
		for _, a := range sr.Actions {
			r.Problem("out of sync: would %s (run `mestiere sync`)", a)
		}
		r.Problems = append(r.Problems, sr.Problems...)
		r.Warnings = append(r.Warnings, sr.Warnings...)
	}

	if inRepo {
		st, err := p.Status()
		if err != nil {
			return r, err
		}
		strict := opts.Strict || p.Manifest.Strict
		for _, l := range st.Local {
			if l.Committed {
				continue
			}
			msg := fmt.Sprintf("local skill %s isn't committed; run `mestiere track %s` to commit it", l.Path, l.Name)
			if strict {
				r.Problem("%s", msg)
			} else {
				r.Warn("%s", msg)
			}
		}
	}
	return r, nil
}

// checkTrackedInstalls finds installed skills that git still tracks even
// though the ignore rule covers them, which happens when they were committed
// before mestiere was set up.
func (p *Project) checkTrackedInstalls(r *Report) {
	desired, err := p.desired()
	if err != nil {
		return
	}
	var dirs []string
	for _, a := range agents.Known() {
		dirs = append(dirs, a.Dir)
	}
	out, err := gitx.RunInput(p.Root, nil, append([]string{"ls-files", "-z", "--"}, dirs...)...)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, file := range strings.Split(string(out), "\x00") {
		for _, a := range agents.Known() {
			rest, ok := strings.CutPrefix(file, a.Dir+"/")
			if !ok {
				continue
			}
			name, _, _ := strings.Cut(rest, "/")
			rel := path.Join(a.Dir, name)
			if _, managed := desired[name]; !managed || seen[rel] {
				continue
			}
			seen[rel] = true
			if ignored, err := gitignore.IsIgnored(p.Root, rel); err == nil && ignored {
				r.Warn("%s is committed but ignored; run `git rm -r --cached %s`, or `mestiere track %s` to keep committing it", rel, rel, name)
			}
		}
	}
}
