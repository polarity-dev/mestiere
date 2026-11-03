// Package manifest reads and writes skills.yaml.
//
// Saving edits the parsed YAML node tree in place rather than re-encoding a
// struct, so comments and key order written by users survive every rewrite.
package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	FileName       = "skills.yaml"
	CurrentVersion = 1
)

var ErrNotFound = errors.New("no " + FileName + " found (run `mestiere init`)")

type LockEntry struct {
	Path   string `yaml:"path"`
	Commit string `yaml:"commit"`
	Tree   string `yaml:"tree"`
}

type Registry struct {
	Name    string
	URL     string
	Ref     string
	Install []string
	Exclude []string
	Lock    map[string]LockEntry
}

type Manifest struct {
	Path       string
	Version    int
	Agents     []string
	Link       string
	Strict     bool
	Registries []*Registry
	doc        *yaml.Node
}

type rawRegistry struct {
	URL     string               `yaml:"url"`
	Ref     string               `yaml:"ref"`
	Install []string             `yaml:"install"`
	Exclude []string             `yaml:"exclude"`
	Lock    map[string]LockEntry `yaml:"lock"`
}

func New(path string) *Manifest {
	return &Manifest{Path: path, Version: CurrentVersion}
}

// Find walks up from start to the nearest directory containing skills.yaml.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	m := &Manifest{Path: path, doc: &doc}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: file is empty", path)
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: expected a mapping at the top level", path)
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i].Value, root.Content[i+1]
		var err error
		switch key {
		case "version":
			err = value.Decode(&m.Version)
		case "agents":
			err = value.Decode(&m.Agents)
		case "link":
			err = value.Decode(&m.Link)
		case "strict":
			err = value.Decode(&m.Strict)
		case "registries":
			err = m.decodeRegistries(value)
		default:
			err = fmt.Errorf("unknown key %q", key)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, key, err)
		}
	}
	if m.Version != CurrentVersion {
		return nil, fmt.Errorf("%s: unsupported version %d (expected %d)", path, m.Version, CurrentVersion)
	}
	return m, nil
}

func (m *Manifest) decodeRegistries(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("expected a mapping of registry names")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name := node.Content[i].Value
		var raw rawRegistry
		if err := node.Content[i+1].Decode(&raw); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if raw.Lock == nil {
			raw.Lock = map[string]LockEntry{}
		}
		m.Registries = append(m.Registries, &Registry{
			Name: name, URL: raw.URL, Ref: raw.Ref,
			Install: raw.Install, Exclude: raw.Exclude, Lock: raw.Lock,
		})
	}
	return nil
}

func (m *Manifest) Registry(name string) *Registry {
	for _, r := range m.Registries {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func (m *Manifest) RemoveRegistry(name string) bool {
	for i, r := range m.Registries {
		if r.Name == name {
			m.Registries = append(m.Registries[:i], m.Registries[i+1:]...)
			return true
		}
	}
	return false
}

// Clone returns a deep copy, used to compute changes without touching the original.
func (r *Registry) Clone() *Registry {
	c := *r
	c.Install = append([]string(nil), r.Install...)
	c.Exclude = append([]string(nil), r.Exclude...)
	c.Lock = make(map[string]LockEntry, len(r.Lock))
	for k, v := range r.Lock {
		c.Lock[k] = v
	}
	return &c
}

func (m *Manifest) Save() error {
	if m.doc == nil || len(m.doc.Content) == 0 {
		m.doc = &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	root := m.doc.Content[0]
	setKey(root, "version", encode(m.Version))
	setKey(root, "agents", encode(nonNil(m.Agents)))
	setOptional(root, "link", m.Link != "", m.Link)
	setOptional(root, "strict", m.Strict, m.Strict)
	setKey(root, "registries", m.registriesNode(lookup(root, "registries")))

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(m.doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.Path), 0o755); err != nil {
		return err
	}
	return writeAtomic(m.Path, buf.Bytes())
}

func (m *Manifest) registriesNode(old *yaml.Node) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	if old != nil && old.Kind == yaml.MappingNode {
		copyComments(node, old)
	}
	for _, reg := range m.Registries {
		var keyNode, valueNode *yaml.Node
		if old != nil {
			keyNode, valueNode = lookupPair(old, reg.Name)
		}
		if keyNode == nil {
			keyNode = &yaml.Node{Kind: yaml.ScalarNode, Value: reg.Name}
		}
		if valueNode == nil || valueNode.Kind != yaml.MappingNode {
			valueNode = &yaml.Node{Kind: yaml.MappingNode}
		}
		valueNode.Style = 0
		setKey(valueNode, "url", encode(reg.URL))
		setOptional(valueNode, "ref", reg.Ref != "", reg.Ref)
		setKey(valueNode, "install", encode(nonNil(reg.Install)))
		setOptional(valueNode, "exclude", len(reg.Exclude) > 0, reg.Exclude)
		if len(reg.Lock) > 0 {
			if lookup(valueNode, "lock") == nil {
				setKey(valueNode, "lock", lockNode(reg.Lock))
				k, _ := lookupPair(valueNode, "lock")
				k.LineComment = "# written by mestiere"
			} else {
				setKey(valueNode, "lock", lockNode(reg.Lock))
			}
		} else {
			deleteKey(valueNode, "lock")
		}
		node.Content = append(node.Content, keyNode, valueNode)
	}
	return node
}

func lockNode(lock map[string]LockEntry) *yaml.Node {
	names := make([]string, 0, len(lock))
	for name := range lock {
		names = append(names, name)
	}
	sort.Strings(names)
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, name := range names {
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: name}, encode(lock[name]))
	}
	return node
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func encode(v any) *yaml.Node {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		panic(err)
	}
	return &n
}

func lookupPair(mapping *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i], mapping.Content[i+1]
		}
	}
	return nil, nil
}

func lookup(mapping *yaml.Node, key string) *yaml.Node {
	_, v := lookupPair(mapping, key)
	return v
}

func setOptional(mapping *yaml.Node, key string, present bool, value any) {
	if present {
		setKey(mapping, key, encode(value))
	} else {
		deleteKey(mapping, key)
	}
}

func setKey(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = merge(mapping.Content[i+1], value)
			return
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
}

func deleteKey(mapping *yaml.Node, key string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

// merge returns next, reusing nodes from old wherever the value is unchanged so
// that comments attached to them are kept.
func merge(old, next *yaml.Node) *yaml.Node {
	if old == nil || old.Kind != next.Kind {
		return next
	}
	switch next.Kind {
	case yaml.ScalarNode:
		if old.Value == next.Value {
			return old
		}
	case yaml.SequenceNode:
		used := map[*yaml.Node]bool{}
		for i, item := range next.Content {
			for _, candidate := range old.Content {
				if !used[candidate] && candidate.Kind == yaml.ScalarNode && item.Kind == yaml.ScalarNode && candidate.Value == item.Value {
					used[candidate] = true
					next.Content[i] = candidate
					break
				}
			}
		}
		// Empty sequences are always emitted in flow style, so only a
		// non-empty one reflects a style the user chose.
		if len(old.Content) > 0 {
			next.Style = old.Style
		}
	case yaml.MappingNode:
		// Mappings stay in block style: comments mestiere attaches to keys
		// can't be emitted inside a flow mapping.
		for i := 0; i+1 < len(next.Content); i += 2 {
			if k, v := lookupPair(old, next.Content[i].Value); k != nil {
				next.Content[i] = k
				next.Content[i+1] = merge(v, next.Content[i+1])
			}
		}
	}
	copyComments(next, old)
	return next
}

func copyComments(dst, src *yaml.Node) {
	dst.HeadComment, dst.LineComment, dst.FootComment = src.HeadComment, src.LineComment, src.FootComment
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".skills-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
