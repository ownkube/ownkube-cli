package functions

import (
	"encoding/json"
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func deployCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "deploy <deployment-id>",
		Short: "Deploy code and settings to a Compute function",
		Long: "Deploy a function's combined source and configuration from a manifest " +
			"file (-f), or '-' for stdin, in one revision. The body carries the inline " +
			"`code`, the entrypoint `filename`, and the function `config` (handler, " +
			"runtime, triggers, and resource settings).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ux.RequireClient()
			if err != nil {
				return err
			}

			raw, err := ux.ReadFileOrStdin(file)
			if err != nil {
				return fmt.Errorf("reading manifest: %w", err)
			}
			body, err := manifestToJSON(raw)
			if err != nil {
				return fmt.Errorf("parsing manifest: %w", err)
			}

			d, err := client.DeployFunction(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderActionResult(cmd, d)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to a manifest file (JSON or YAML); '-' for stdin")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

// manifestToJSON normalizes a function manifest (JSON or YAML) into JSON bytes
// for the raw-body deploy call. JSON is valid YAML, so a single YAML decode
// handles both.
func manifestToJSON(raw []byte) ([]byte, error) {
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}

// renderActionResult prints the DeploymentActionResult from a function deploy.
// Chart-backed versions are surfaced as "Platform Version" per the product
// vocabulary.
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
		{"Status Message", r.StatusMessage},
		{"Environment", r.EnvironmentId},
		{"Platform Version", r.ChartVersion},
	})
}
