package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func upgradeCmd() *cobra.Command {
	var function bool

	cmd := &cobra.Command{
		Use:   "upgrade <deployment-id> <platform-version>",
		Short: "Upgrade a deployment's platform version",
		Long: "Move a deployment to a newer platform version. Pass --function for a " +
			"function deployment. List available versions with 'okctl deploy platform-versions'.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var upgrade = api.UpgradePlatformVersion
			if function {
				upgrade = api.UpgradeFunctionPlatformVersion
			}
			result, err := upgrade(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), result)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Platform Version", result.ChartVersion},
				{"Success", fmt.Sprintf("%t", result.Success)},
			})
		},
	}

	cmd.Flags().BoolVar(&function, "function", false, "Treat the target as a function deployment")
	return cmd
}

func platformVersionsCmd() *cobra.Command {
	var (
		clusterType  string
		resourceType string
		function     bool
	)

	cmd := &cobra.Command{
		Use:   "platform-versions",
		Short: "List available platform versions",
		Long: "List the platform versions available for a resource type on a cluster " +
			"type, or with --function for functions.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var versions []api.PlatformVersionEntry
			if function {
				versions, err = c.ListFunctionPlatformVersions(cmd.Context())
			} else {
				if clusterType == "" || resourceType == "" {
					return fmt.Errorf("--cluster-type and --resource-type are required (or pass --function)")
				}
				versions, err = c.ListPlatformVersions(cmd.Context(), clusterType, resourceType)
			}
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), versions)
			}
			if len(versions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No platform versions found.")
				return nil
			}
			rows := [][]string{{"VERSION", "APP VERSION"}}
			for _, v := range versions {
				rows = append(rows, []string{v.Version, ux.Deref(v.AppVersion)})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}

	cmd.Flags().StringVar(&clusterType, "cluster-type", "", "Cluster type: eks, gke, aks, or k3s")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "Resource type: web, worker, job, or database")
	cmd.Flags().BoolVar(&function, "function", false, "List function platform versions instead")
	return cmd
}
