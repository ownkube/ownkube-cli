package environments

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// validColors mirrors the server's accepted accent colors. An empty string
// leaves the color unset (the server defaults it).
var validColors = map[string]struct{}{
	"amber":  {},
	"blue":   {},
	"green":  {},
	"purple": {},
	"red":    {},
}

// validateColor returns a normalized color pointer, or nil when unset. It
// rejects anything outside the accepted set so the user gets a clear message
// instead of a server-side 400.
func validateColor(color string) (*string, error) {
	if color == "" {
		return nil, nil
	}
	if _, ok := validColors[color]; !ok {
		return nil, fmt.Errorf("invalid --color %q: expected amber, blue, green, purple, or red", color)
	}
	return &color, nil
}

// renderEnvironment prints a single environment as JSON (structured mode) or a
// field/value table, matching `environments get`.
func renderEnvironment(cmd *cobra.Command, e *api.Environment) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), e)
	}
	count := ""
	if e.DeploymentCount != nil {
		count = fmt.Sprintf("%g", *e.DeploymentCount)
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", e.Id},
		{"Name", e.Name},
		{"Slug", e.Slug},
		{"Description", ux.Deref(e.Description)},
		{"Color", ux.Deref(e.Color)},
		{"Deployments", count},
	})
}
