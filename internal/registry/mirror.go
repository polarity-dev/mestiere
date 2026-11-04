// Package registry reads skill registries through a local bare mirror.
package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/polarity-dev/mestiere/internal/fsx"
	"github.com/polarity-dev/mestiere/internal/gitx"
	"github.com/polarity-dev/mestiere/internal/naming"
)

var ErrNotFound = errors.New("not found")

// CacheDir returns mestiere's cache directory, resolved in order from
// MESTIERE_CACHE_DIR, $XDG_CACHE_HOME/mestiere, and the platform default
// (~/Library/Caches/mestiere on macOS, ~/.cache/mestiere on Linux).
//
// XDG_CACHE_HOME is honored on every platform, unlike os.UserCacheDir, which
// ignores it on macOS even though developers commonly set it there.
func CacheDir() (string, error) {
	if dir := os.Getenv("MESTIERE_CACHE_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "mestiere"), nil
	}
	return filepath.Join(platformCacheDir(), "mestiere"), nil
}

// platformCacheDir returns the OS cache directory without consulting
// XDG_CACHE_HOME, which CacheDir has already handled. os.UserCacheDir can't be
// used on Linux because it returns an error when XDG_CACHE_HOME is relative,
// where the XDG spec says to ignore the value.
func platformCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches")
	}
	return filepath.Join(home, ".cache")
}

// CacheRoot returns where registry mirrors live.
func CacheRoot() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "registries"), nil
}

var mirrorDirRe = regexp.MustCompile(`^[0-9a-f]{16}$`)

// CleanCache deletes every registry mirror and leftover staging directory.
// Only entries mestiere creates are removed, so a cache directory pointed at
// a shared location keeps everything else.
func CleanCache() (removed []string, err error) {
	root, err := CacheRoot()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || (!mirrorDirRe.MatchString(e.Name()) && !strings.HasPrefix(e.Name(), fsx.TempPrefix)) {
			continue
		}
		p := filepath.Join(root, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return removed, err
		}
		removed = append(removed, p)
	}
	return removed, nil
}

// CacheSize returns the total size in bytes of the registry mirrors.
func CacheSize() (int64, error) {
	root, err := CacheRoot()
	if err != nil {
		return 0, err
	}
	var total int64
	err = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// Mirror is a bare clone of a registry, shared by every project on the machine.
type Mirror struct {
	URL     string
	Dir     string
	fetched bool
	// history caches HistoricalTrees per skill; contents caches Materialize
	// per (skill tree, shared tree) pair.
	history  map[naming.Spec]map[string]bool
	contents map[[2]string]*Content
}

func Open(url string) (*Mirror, error) {
	root, err := CacheRoot()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(url))
	return &Mirror{URL: url, Dir: filepath.Join(root, hex.EncodeToString(sum[:])[:16])}, nil
}

func (m *Mirror) Exists() bool {
	_, err := os.Stat(filepath.Join(m.Dir, "HEAD"))
	return err == nil
}

// Fetch clones the mirror if missing and updates all branches and tags.
// It runs at most once per Mirror value.
func (m *Mirror) Fetch() error {
	if m.fetched {
		return nil
	}
	if !m.Exists() {
		if err := m.clone(); err != nil {
			return err
		}
	} else if _, err := gitx.Run(m.Dir, "fetch", "--quiet", "--prune", "--force", "origin",
		"+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"); err != nil {
		return err
	}
	m.fetched = true
	m.history = nil
	return nil
}

func (m *Mirror) clone() error {
	if err := os.MkdirAll(filepath.Dir(m.Dir), 0o755); err != nil {
		return err
	}
	staged, err := fsx.TempSibling(m.Dir)
	if err != nil {
		return err
	}
	defer os.RemoveAll(staged)
	if _, err := gitx.Run(filepath.Dir(m.Dir), "clone", "--bare", "--quiet", m.URL, staged); err != nil {
		return err
	}
	return os.Rename(staged, m.Dir)
}

// Resolve turns a branch, tag or commit into a commit ID. An empty ref means
// the registry's default branch.
func (m *Mirror) Resolve(ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	commit, err := gitx.Run(m.Dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil || commit == "" {
		return "", fmt.Errorf("ref %q: %w", ref, ErrNotFound)
	}
	return commit, nil
}

func (m *Mirror) ObjectFormat() fsx.ObjectFormat {
	if out, err := gitx.Run(m.Dir, "rev-parse", "--show-object-format"); err == nil && out == string(fsx.SHA256) {
		return fsx.SHA256
	}
	return fsx.SHA1
}

func (m *Mirror) HasObject(id string) bool {
	_, err := gitx.Run(m.Dir, "cat-file", "-e", id)
	return err == nil
}

// Catalog lists valid namespaces and their skills at commit. A skill is a
// directory at skills/<ns>/<skill> containing SKILL.md.
func (m *Mirror) Catalog(commit string) (map[string][]string, error) {
	out, err := gitx.RunInput(m.Dir, nil, "ls-tree", "-r", "-z", "--name-only", commit, "--", "skills/")
	if err != nil {
		return nil, err
	}
	catalog := map[string][]string{}
	for _, p := range strings.Split(string(out), "\x00") {
		if path.Base(p) != "SKILL.md" {
			continue
		}
		spec, ok := naming.ParsePath(path.Dir(p), false)
		if !ok {
			continue
		}
		catalog[spec.Namespace] = append(catalog[spec.Namespace], spec.Skill)
	}
	for ns := range catalog {
		sort.Strings(catalog[ns])
	}
	return catalog, nil
}

// TreeAt returns the tree ID of path at commit.
func (m *Mirror) TreeAt(commit, p string) (string, error) {
	out, err := gitx.RunInput(m.Dir, strings.NewReader(commit+":"+p+"\n"), "cat-file", "--batch-check")
	fields := strings.Fields(string(out))
	if err != nil || len(fields) != 3 || fields[1] != "tree" {
		return "", fmt.Errorf("%s at %s: %w", p, Short(commit), ErrNotFound)
	}
	return fields[0], nil
}

// Reachable checks that the remote answers for ref without touching the cache.
func Reachable(dir, url, ref string) error {
	if ref == "" {
		ref = "HEAD"
	}
	_, err := gitx.Run(dir, "ls-remote", "--exit-code", url, ref)
	return err
}

func Short(id string) string {
	if len(id) > 7 {
		return id[:7]
	}
	return id
}
