package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	var (
		file     string
		function bool
	)

	cmd := &cobra.Command{
		Use:   "update <deployment-id>",
		Short: "Update a deployment from a manifest",
		Long: "Update a deployment's configuration from a manifest file (-f), or '-' " +
			"for stdin. The body carries the resource's config plus an optional note. " +
			"Pass --function for a function deployment (its update body is shaped " +
			"differently from cluster-hosted resources).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
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

			var result = api.UpdateDeployment
			if function {
				result = api.UpdateFunction
			}
			d, err := result(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderActionResult(cmd, d)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to a manifest file (JSON or YAML); '-' for stdin")
	cmd.Flags().BoolVar(&function, "function", false, "Treat the target as a function deployment")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}
