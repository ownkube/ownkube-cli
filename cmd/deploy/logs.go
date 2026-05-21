package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

func logsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "logs <deployment-id>",
		Short: "Fetch logs for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			rangeSec, _ := cmd.Flags().GetInt("range")
			limit, _ := cmd.Flags().GetInt("limit")
			filter, _ := cmd.Flags().GetString("filter")

			entries, err := api.GetDeploymentLogs(cmd.Context(), args[0], client.LogsOptions{
				RangeSeconds: rangeSec,
				Limit:        limit,
				Filter:       filter,
			})
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), entries)
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No log entries.")
				return nil
			}
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", ux.Deref(e.Timestamp), ux.Deref(e.Message))
			}
			return nil
		},
	}
	c.Flags().Int("range", 0, "Look back this many seconds")
	c.Flags().Int("limit", 0, "Maximum number of log entries to return")
	c.Flags().String("filter", "", "Filter log entries by substring")
	return c
}
