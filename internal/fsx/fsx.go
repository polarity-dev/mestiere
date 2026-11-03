// Package fsx hashes and copies skill directories.
package fsx

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ObjectFormat is the git object hash algorithm of a registry.
type ObjectFormat string

const (
	SHA1   ObjectFormat = "sha1"
	SHA256 ObjectFormat = "sha256"
)

// FormatOf infers the object format from a hex object ID.
func FormatOf(id string) ObjectFormat {
	if len(id) == 64 {
		return SHA256
	}
	return SHA1
}

func (f ObjectFormat) newHash() hash.Hash {
	if f == SHA256 {
		return sha256.New()
	}
	return sha1.New()
}

// IsHidden reports whether a file or directory name is skipped when hashing
// and copying skills. Editors and file managers drop hidden files (.DS_Store,
// swap files, .vscode/) into folders they open, which would otherwise make an
// untouched installed skill look locally modified. Registries can't publish
// hidden files for the same reason, so installed content never contains any.
func IsHidden(name string) bool { return strings.HasPrefix(name, ".") }

// TreeHash computes the git tree object ID that dir would have if committed.
//
// It follows git's tree encoding: entries sorted with directories compared as
// "name/", empty directories omitted, and only the executable bit kept from file
// modes. Filesystems without an executable bit (Windows) cannot reproduce trees
// that contain executable files.
func TreeHash(dir string, format ObjectFormat) (string, error) {
	raw, _, err := hashTree(dir, format)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

type treeEntry struct {
	mode string
	name string
	id   []byte
}

// Entry is a file in a tree described by git object IDs instead of disk content.
type Entry struct {
	// Path is slash-separated and relative to the tree root.
	Path string
	// Mode is a git file mode: 100644, 100755 or 120000.
	Mode string
	OID  string
}

type entryDir struct {
	files []treeEntry
	dirs  map[string]*entryDir
}

// HashEntries computes the git tree ID of a tree holding exactly entries.
// For the same files it returns the same ID as TreeHash on disk.
func HashEntries(format ObjectFormat, entries []Entry) (string, error) {
	root := &entryDir{dirs: map[string]*entryDir{}}
	for _, e := range entries {
		parts := strings.Split(e.Path, "/")
		d := root
		for _, part := range parts[:len(parts)-1] {
			sub, ok := d.dirs[part]
			if !ok {
				sub = &entryDir{dirs: map[string]*entryDir{}}
				d.dirs[part] = sub
			}
			d = sub
		}
		id, err := hex.DecodeString(e.OID)
		if err != nil {
			return "", fmt.Errorf("%s: invalid object ID %q", e.Path, e.OID)
		}
		d.files = append(d.files, treeEntry{e.Mode, parts[len(parts)-1], id})
	}
	id, err := root.hash(format, "")
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(id), nil
}

func (d *entryDir) hash(format ObjectFormat, at string) ([]byte, error) {
	entries := append([]treeEntry(nil), d.files...)
	for name, sub := range d.dirs {
		for _, f := range d.files {
			if f.name == name {
				return nil, fmt.Errorf("%s is both a file and a directory", path.Join(at, name))
			}
		}
		id, err := sub.hash(format, path.Join(at, name))
		if err != nil {
			return nil, err
		}
		entries = append(entries, treeEntry{"40000", name, id})
	}
	return encodeTree(format, entries), nil
}

func encodeTree(format ObjectFormat, entries []treeEntry) []byte {
	sort.Slice(entries, func(i, j int) bool { return entries[i].sortKey() < entries[j].sortKey() })
	var body bytes.Buffer
	for _, e := range entries {
		body.WriteString(e.mode)
		body.WriteByte(' ')
		body.WriteString(e.name)
		body.WriteByte(0)
		body.Write(e.id)
	}
	return hashObject(format, "tree", body.Bytes())
}

func (e treeEntry) sortKey() string {
	if e.mode == "40000" {
		return e.name + "/"
	}
	return e.name
}

func hashTree(dir string, format ObjectFormat) (id []byte, empty bool, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false, err
	}
	var entries []treeEntry
	for _, de := range dirEntries {
		name := de.Name()
		if IsHidden(name) {
			continue
		}
		p := filepath.Join(dir, name)
		info, err := os.Lstat(p)
		if err != nil {
			return nil, false, err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return nil, false, err
			}
			entries = append(entries, treeEntry{"120000", name, hashObject(format, "blob", []byte(filepath.ToSlash(target)))})
		case info.IsDir():
			sub, subEmpty, err := hashTree(p, format)
			if err != nil {
				return nil, false, err
			}
			if !subEmpty {
				entries = append(entries, treeEntry{"40000", name, sub})
			}
		case info.Mode().IsRegular():
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, false, err
			}
			mode := "100644"
			if info.Mode().Perm()&0o111 != 0 {
				mode = "100755"
			}
			entries = append(entries, treeEntry{mode, name, hashObject(format, "blob", data)})
		default:
			return nil, false, fmt.Errorf("%s: unsupported file type", p)
		}
	}
	return encodeTree(format, entries), len(entries) == 0, nil
}

func hashObject(format ObjectFormat, kind string, data []byte) []byte {
	h := format.newHash()
	fmt.Fprintf(h, "%s %d\x00", kind, len(data))
	h.Write(data)
	return h.Sum(nil)
}

// CopyDir copies src to dst, preserving symlinks and the executable bit.
func CopyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != src && IsHidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.IsDir():
			return os.MkdirAll(target, 0o755)
		default:
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode().Perm())
		}
	})
}

// TempSibling returns an unused path next to dest. The "." prefix keeps it out
// of skill discovery and inside the default .gitignore rule.
func TempSibling(dest string) (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dest), TempPrefix+filepath.Base(dest)+"-"+hex.EncodeToString(b[:])), nil
}

// TempPrefix marks staging directories that a crashed run may leave behind.
const TempPrefix = ".mestiere-tmp-"

// ReplaceDir moves staged into dest, removing whatever dest held before.
func ReplaceDir(staged, dest string) error {
	if _, err := os.Lstat(dest); err == nil {
		old, err := TempSibling(dest)
		if err != nil {
			return err
		}
		if err := os.Rename(dest, old); err != nil {
			return err
		}
		if err := os.Rename(staged, dest); err != nil {
			_ = os.Rename(old, dest)
			return err
		}
		return os.RemoveAll(old)
	}
	return os.Rename(staged, dest)
}
