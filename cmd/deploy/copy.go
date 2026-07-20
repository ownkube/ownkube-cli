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
		note          string
	)

	cmd := &cobra.Command{
		Use:   "copy <deployment-id>",
		Short: "Copy a deployment into another environment/cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			body := apiCopyBody(toEnvironment, toCluster, note)
			d, err := api.CopyDeployment(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderActionResult(cmd, d)
		},
	}

	cmd.Flags().StringVar(&toEnvironment, "to-environment", "", "Target environment ID")
	cmd.Flags().StringVar(&toCluster, "to-cluster", "", "Target cluster ID")
	cmd.Flags().StringVar(&note, "note", "", "Optional note for the revision")
	_ = cmd.MarkFlagRequired("to-environment")
	_ = cmd.MarkFlagRequired("to-cluster")
	return cmd
}

func apiCopyBody(env, cluster, note string) api.CopyDeploymentBody {
	body := api.CopyDeploymentBody{
		TargetEnvironmentId: env,
		TargetClusterId:     cluster,
	}
	if note != "" {
		body.Note = &note
	}
	return body
}
