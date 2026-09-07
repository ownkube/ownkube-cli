// Package manifest reads and writes ownkube.yaml — the committed, hand-editable
// deploy spec for a project directory.
//
// ownkube.yaml is config-as-code: build/run settings live here and are meant to
// be checked in. It holds no ids and no secret values (secret env vars are
// referenced by name only); the target a directory deploys to lives in the
// separate, never-committed link binding (see internal/link) — the committed
// spec and the machine-local target binding stay deliberately apart.
//
// The server is the authoritative validator of a deploy spec. This package
// keeps a small typed view for CLI ergonomics (name, resource type, the web
// common case) plus a JSON projection, mirroring how `deploy create -f` posts
// the manifest to the API.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the fixed name of the deploy spec in a project directory.
const FileName = "ownkube.yaml"

// EnvVar is one environment entry. A secret entry omits Value in the file and
// carries only the reference; the value is supplied out of band (never
// committed).
type EnvVar struct {
	Name   string `yaml:"name" json:"name"`
	Value  string `yaml:"value,omitempty" json:"value,omitempty"`
	Secret bool   `yaml:"secret,omitempty" json:"secret,omitempty"`
}

// Manifest is the typed view of ownkube.yaml. Fields cover the common web case;
// the server validates the full spec on deploy.
type Manifest struct {
	Name         string   `yaml:"name" json:"name"`
	Type         string   `yaml:"type,omitempty" json:"type,omitempty"` // resourceType: web|worker|job|database|function
	Image        string   `yaml:"image,omitempty" json:"image,omitempty"`
	Tag          string   `yaml:"tag,omitempty" json:"tag,omitempty"`
	Port         int      `yaml:"port,omitempty" json:"port,omitempty"`
	Replicas     int      `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Public       bool     `yaml:"public,omitempty" json:"public,omitempty"`
	StartCommand string   `yaml:"startCommand,omitempty" json:"startCommand,omitempty"`
	Env          []EnvVar `yaml:"env,omitempty" json:"env,omitempty"`
}

// Path returns the ownkube.yaml path for dir.
func Path(dir string) string { return filepath.Join(dir, FileName) }

// Exists reports whether dir contains an ownkube.yaml.
func Exists(dir string) bool {
	_, err := os.Stat(Path(dir))
	return err == nil
}

// Load reads and parses ownkube.yaml from dir.
func Load(dir string) (*Manifest, error) {
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		return nil, err
	}
	m := &Manifest{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("%s: 'name' is required", FileName)
	}
	return m, nil
}

// Write serializes m to ownkube.yaml in dir. It refuses to overwrite an
// existing file unless force is set, so scaffolding never clobbers a spec a
// developer has edited.
func Write(dir string, m *Manifest, force bool) error {
	p := Path(dir)
	if !force {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("%s already exists", FileName)
		}
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", FileName, err)
	}
	return os.WriteFile(p, data, 0644)
}

// JSON returns the manifest as JSON — the shape the deploy API accepts as a
// create/update body.
func (m *Manifest) JSON() ([]byte, error) {
	return json.Marshal(m)
}
