package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// jobRunsCmd groups the job-run lifecycle commands for job deployments.
func jobRunsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "job-runs",
		Short: "Manage runs of a job deployment",
	}
	root.AddCommand(jobRunsListCmd(), jobRunsTriggerCmd(), jobRunsCancelCmd())
	return root
}

func jobRunsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <deployment-id>",
		Short: "List a job's run history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			history, err := api.GetJobRuns(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), history)
			}
			rows := [][]string{
				{"FIELD", "VALUE"},
				{"Exists", fmt.Sprintf("%t", history.Exists)},
				{"Schedule", ux.Deref(history.Schedule)},
				{"Time Zone", ux.Deref(history.TimeZone)},
				{"Last Scheduled", ux.Deref(history.LastScheduleTime)},
				{"Last Successful", ux.Deref(history.LastSuccessfulTime)},
				{"Runs", fmt.Sprintf("%d", len(history.Runs))},
			}
			if history.Suspended != nil {
				rows = append(rows, []string{"Suspended", fmt.Sprintf("%t", *history.Suspended)})
			}
			if len(history.Runs) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Use --output json to see individual run details.")
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

func jobRunsTriggerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trigger <deployment-id>",
		Short: "Trigger a one-off run of a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			result, err := api.TriggerJobRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Triggered job run %s.\n", result.JobName)
			return nil
		},
	}
}

func jobRunsCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <deployment-id> <job-name>",
		Short: "Cancel an in-flight job run",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			result, err := api.CancelJobRun(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cancel requested for %s (canceled=%t).\n", args[1], result.Canceled)
			return nil
		},
	}
}
