// Package ux holds shared helpers for cobra subcommand packages: the resolved
// global flag state, an authenticated client factory, output rendering, and a
// few stdin/string utilities. Subpackages depend on ux rather than reaching
// back into the parent cmd package, which avoids import cycles.
package ux

import (
	"fmt"
	"io"
	"os"

	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/config"
	"github.com/ownkube/okctl/internal/output"
)

// Globals captures the runtime state cmd/root resolves once per invocation.
type Globals struct {
	APIURL       string
	OutputFormat string
	Config       *config.Manager
}

var g Globals

// Set installs the resolved global state. Called from cmd/root's
// PersistentPreRunE before any subcommand runs.
func Set(globals Globals) { g = globals }

// APIURL returns the resolved API base URL.
func APIURL() string { return g.APIURL }

// OutputFormat returns the resolved output format ("table", "json", "yaml").
func OutputFormat() string { return g.OutputFormat }

// IsStructured reports whether the output format expects a structured
// (machine-readable) encoder rather than a human-readable table.
func IsStructured() bool {
	return g.OutputFormat == "json" || g.OutputFormat == "yaml"
}

// Config returns the active config manager.
func Config() *config.Manager { return g.Config }

// RequireAuth loads credentials and returns an error if not logged in.
func RequireAuth() (*config.Credentials, error) {
	creds, err := g.Config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds == nil {
		return nil, fmt.Errorf("not logged in — run 'okctl login' first")
	}
	return creds, nil
}

// RequireClient returns an authenticated API client or an error if the user
// is not logged in.
func RequireClient() (*client.Client, error) {
	creds, err := RequireAuth()
	if err != nil {
		return nil, err
	}
	return client.New(g.APIURL, creds.APIKey)
}

// Print writes data using the configured output format.
func Print(w io.Writer, data any) error {
	return output.New(w, g.OutputFormat).Print(data)
}

// Deref safely dereferences a string pointer, returning "" if nil.
func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ReadFileOrStdin reads bytes from path, or from stdin when path is "-".
func ReadFileOrStdin(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}
