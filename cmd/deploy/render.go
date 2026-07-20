package deploy

import (
	"encoding/json"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// manifestToJSON normalizes a deployment manifest (JSON or YAML) into JSON
// bytes for the raw-body create/update calls. JSON is valid YAML, so a single
// YAML decode handles both; yaml.v3 decodes mappings into map[string]any, which
// marshals back to clean JSON.
func manifestToJSON(raw []byte) ([]byte, error) {
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}

// renderActionResult prints a DeploymentActionResult as a field table, or the
// raw object when structured output is requested. Chart-backed versions are
// surfaced as "Platform Version" per the product vocabulary.
func renderActionResult(cmd *cobra.Command, r *api.DeploymentActionResult) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), r)
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", r.Id},
		{"Name", r.Name},
		{"Type", string(r.ResourceType)},
		{"Status", r.Status},
		{"Status Message", ux.Deref(r.StatusMessage)},
		{"Cluster", ux.Deref(r.ClusterId)},
		{"Environment", ux.Deref(r.EnvironmentId)},
		{"Platform Version", ux.Deref(r.ChartVersion)},
	})
}

// renderCreated prints a freshly created deployment.
func renderCreated(cmd *cobra.Command, r *api.CreateDeploymentResponse) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), r)
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", r.Id},
		{"Name", r.Name},
		{"Type", string(r.ResourceType)},
		{"Status", r.Status},
		{"Status Message", ux.Deref(r.StatusMessage)},
		{"Cluster", ux.Deref(r.ClusterId)},
		{"Environment", ux.Deref(r.EnvironmentId)},
		{"Public Hostname", ux.Deref(r.PublicHostname)},
		{"Platform Version", ux.Deref(r.ChartVersion)},
	})
}

// renderRevision prints a single revision (the new one produced by rollback or
// promote).
func renderRevision(cmd *cobra.Command, r *api.Revision) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), r)
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", r.Id},
		{"Status", r.Status},
		{"Image", r.Image},
		{"Tag", r.Tag},
		{"Source", ux.Deref(r.Source)},
		{"Note", ux.Deref(r.Note)},
	})
}
