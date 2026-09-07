// Package link manages the per-directory binding between a working directory
// and specific Ownkube resources (org / cluster / environment / deployment).
//
// Bindings live in a single global links.yaml alongside config.yaml and
// credentials.yaml, keyed by the absolute path of the directory's git root (or
// the directory itself when it is not inside a repo). They hold ids only —
// never credentials — and are never written into the project, so cloning a
// teammate's repo never inherits their binding. This mirrors how the config
// package keeps global preferences: the deploy spec (ownkube.yaml, see
// internal/manifest) is committed; the binding is not.
package link

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Binding records which Ownkube resources a directory targets. Every field is
// an id; there are no credentials here. Commands resolve missing flags from the
// binding for the current directory, with explicit flags always overriding.
type Binding struct {
	OrganizationID string `yaml:"organization_id,omitempty"`
	ClusterID      string `yaml:"cluster_id,omitempty"`
	EnvironmentID  string `yaml:"environment_id,omitempty"`
	DeploymentID   string `yaml:"deployment_id,omitempty"`
}

// store is the on-disk shape of links.yaml: a map of directory path -> binding.
type store struct {
	Links map[string]Binding `yaml:"links"`
}

// Manager reads and writes the global links.yaml.
type Manager struct {
	path string
}

// NewManager returns a Manager for links.yaml inside dir, which should be the
// shared config directory (config.Manager.Dir()).
func NewManager(dir string) *Manager {
	return &Manager{path: filepath.Join(dir, "links.yaml")}
}

func (m *Manager) load() (*store, error) {
	s := &store{Links: map[string]Binding{}}
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading links: %w", err)
	}
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing links: %w", err)
	}
	if s.Links == nil {
		s.Links = map[string]Binding{}
	}
	return s, nil
}

func (m *Manager) save(s *store) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshalling links: %w", err)
	}
	// 0600: the binding is not secret, but it sits in the same dir as
	// credentials.yaml and there is no reason to make it group/world readable.
	return os.WriteFile(m.path, data, 0600)
}

// Get returns the binding for key and whether one exists.
func (m *Manager) Get(key string) (Binding, bool, error) {
	s, err := m.load()
	if err != nil {
		return Binding{}, false, err
	}
	b, ok := s.Links[key]
	return b, ok, nil
}

// Set stores the binding for key, replacing any existing one.
func (m *Manager) Set(key string, b Binding) error {
	s, err := m.load()
	if err != nil {
		return err
	}
	s.Links[key] = b
	return m.save(s)
}

// Remove deletes the binding for key. Returns whether one was present.
func (m *Manager) Remove(key string) (bool, error) {
	s, err := m.load()
	if err != nil {
		return false, err
	}
	if _, ok := s.Links[key]; !ok {
		return false, nil
	}
	delete(s.Links, key)
	return true, m.save(s)
}

// ResolveKey returns the binding key for dir: the absolute path of the nearest
// enclosing git repository root, or the absolute directory itself when it is
// not inside a repo. Anchoring on the repo root means every subdirectory of a
// project shares one binding.
func ResolveKey(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if root := findRepoRoot(abs); root != "" {
		return root, nil
	}
	return abs, nil
}

// findRepoRoot walks up from start looking for a .git entry (a directory for a
// normal clone, a file for worktrees/submodules). Returns "" when none is
// found before the filesystem root.
func findRepoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
