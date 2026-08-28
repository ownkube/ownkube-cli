package deploy

import (
	"fmt"
	"sort"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func connectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connection <deployment-id>",
		Short: "Show in-cluster connection details (namespace, service, secret) for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			conn, err := api.GetDeploymentConnection(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), conn)
			}

			rows := [][]string{
				{"FIELD", "VALUE"},
				{"Namespace", conn.Namespace},
				{"Service", conn.ServiceName},
				{"Secret", conn.SecretName},
			}
			if conn.Details != nil {
				keys := make([]string, 0, len(conn.Details))
				for k := range conn.Details {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					rows = append(rows, []string{"details." + k, fmt.Sprintf("%v", conn.Details[k])})
				}
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
