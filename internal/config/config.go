package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultAPIURL       = "https://app.ownkube.io"
	DefaultOutputFormat = "table"
)

// Config holds CLI preferences stored in config.yaml.
type Config struct {
	APIURL       string `yaml:"api_url,omitempty"`
	OutputFormat string `yaml:"output_format,omitempty"`
	// Organization is the default org ID sent as the x-ownkube-organization
	// header. Required for org-scoped commands when the account belongs to
	// more than one organization.
	Organization string `yaml:"organization,omitempty"`
}

// Credentials holds auth info stored in credentials.yaml (0600).
type Credentials struct {
	APIKey        string `yaml:"api_key"`
	UserID        string `yaml:"user_id"`
	UserName      string `yaml:"user_name"`
	UserEmail     string `yaml:"user_email"`
	EmailVerified bool   `yaml:"email_verified"`
}

// Manager handles reading and writing config/credentials files.
type Manager struct {
	dir string
}

// NewManager creates a Manager using the given config directory.
// If dir is empty, it defaults to ~/.config/ownkube.
func NewManager(dir string) (*Manager, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".config", "ownkube")
	}
	return &Manager{dir: dir}, nil
}

// Dir returns the config directory path.
func (m *Manager) Dir() string {
	return m.dir
}

func (m *Manager) configPath() string {
	return filepath.Join(m.dir, "config.yaml")
}

func (m *Manager) credentialsPath() string {
	return filepath.Join(m.dir, "credentials.yaml")
}

func (m *Manager) ensureDir() error {
	return os.MkdirAll(m.dir, 0700)
}

// LoadConfig reads config.yaml, returning defaults if it doesn't exist.
func (m *Manager) LoadConfig() (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(m.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes config.yaml.
func (m *Manager) SaveConfig(cfg *Config) error {
	if err := m.ensureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	return os.WriteFile(m.configPath(), data, 0644)
}

// LoadCredentials reads credentials.yaml, returning nil if it doesn't exist.
func (m *Manager) LoadCredentials() (*Credentials, error) {
	data, err := os.ReadFile(m.credentialsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}
	creds := &Credentials{}
	if err := yaml.Unmarshal(data, creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	return creds, nil
}

// SaveCredentials writes credentials.yaml with 0600 permissions.
func (m *Manager) SaveCredentials(creds *Credentials) error {
	if err := m.ensureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshalling credentials: %w", err)
	}
	return os.WriteFile(m.credentialsPath(), data, 0600)
}

// DeleteCredentials removes credentials.yaml.
func (m *Manager) DeleteCredentials() error {
	err := os.Remove(m.credentialsPath())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing credentials: %w", err)
	}
	return nil
}

// Get returns the value of a config key.
func (cfg *Config) Get(key string) (string, error) {
	switch key {
	case "api_url":
		return cfg.APIURL, nil
	case "output_format":
		return cfg.OutputFormat, nil
	case "organization":
		return cfg.Organization, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Set sets the value of a config key.
func (cfg *Config) Set(key, value string) error {
	switch key {
	case "api_url":
		cfg.APIURL = value
	case "output_format":
		cfg.OutputFormat = value
	case "organization":
		cfg.Organization = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: api_url, output_format, organization)", key)
	}
	return nil
}
