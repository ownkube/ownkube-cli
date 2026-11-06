package deploy

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func copyCmd() *cobra.Command {
	var (
		toEnvironment string
		toCluster     string
		toRegion      string
		note          string
	)

	cmd := &cobra.Command{
		Use:   "copy [app]",
		Short: "Copy a deployment into another environment",
		Long: "Copy a deployment into another environment. Pass --to-cluster to land it on " +
			"one of your clusters, or --to-region (from 'okctl regions list') to land it " +
			"on Ownkube Compute.",
		Example: "  okctl deploy copy my-app --to-environment <env-id> --to-region europe\n" +
			"  okctl deploy copy my-app --to-environment <env-id> --to-cluster <cluster-id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			body := apiCopyBody(toEnvironment, toCluster, toRegion, note)
			d, err := api.CopyDeployment(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderActionResult(cmd, d)
		},
	}

	cmd.Flags().StringVar(&toEnvironment, "to-environment", "", "Target environment ID")
	cmd.Flags().StringVar(&toCluster, "to-cluster", "", "Target cluster ID")
	cmd.Flags().StringVar(&toRegion, "to-region", "", "Target Ownkube Compute region (from 'okctl regions list')")
	cmd.Flags().StringVar(&note, "note", "", "Optional note for the revision")
	_ = cmd.MarkFlagRequired("to-environment")
	cmd.MarkFlagsOneRequired("to-cluster", "to-region")
	cmd.MarkFlagsMutuallyExclusive("to-cluster", "to-region")
	return cmd
}

func apiCopyBody(env, cluster, region, note string) api.CopyDeploymentBody {
	body := api.CopyDeploymentBody{TargetEnvironmentId: env}
	if cluster != "" {
		body.TargetClusterId = &cluster
	}
	if region != "" {
		body.TargetRegion = &region
	}
	if note != "" {
		body.Note = &note
	}
	return body
}
