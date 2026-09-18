package projects

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// validColors mirrors the server's accepted accent colors. An empty string
// leaves the color unset (the server defaults it to blue).
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

// renderProject prints a single project as JSON (structured mode) or a
// field/value table, matching `projects get`.
func renderProject(cmd *cobra.Command, p *api.Project) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), p)
	}
	isDefault := ""
	if p.IsDefault != nil {
		isDefault = fmt.Sprintf("%t", *p.IsDefault)
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", p.Id},
		{"Name", p.Name},
		{"Slug", p.Slug},
		{"Description", ux.Deref(p.Description)},
		{"Color", ux.Deref(p.Color)},
		{"Default", isDefault},
		{"Deployments", floatCount(p.DeploymentCount)},
		{"Environments", floatCount(p.EnvironmentCount)},
	})
}

// floatCount renders an optional *float32 count as a plain integer-ish string,
// or empty when the server omitted it.
func floatCount(v *float32) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%g", *v)
}
