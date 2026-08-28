package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/output"
	"github.com/spf13/cobra"
)

// rangeStepFlags adds the shared time-window flags used by the analytics reads.
func rangeStepFlags(c *cobra.Command) {
	c.Flags().Int("range-seconds", 0, "Look-back window in seconds")
	c.Flags().Int("step", 0, "Sample step in seconds")
}

func rangeStepValues(cmd *cobra.Command) (rangeSeconds, step *int) {
	if cmd.Flags().Changed("range-seconds") {
		v, _ := cmd.Flags().GetInt("range-seconds")
		rangeSeconds = &v
	}
	if cmd.Flags().Changed("step") {
		v, _ := cmd.Flags().GetInt("step")
		step = &v
	}
	return rangeSeconds, step
}

// printBlob renders a nested analytics payload. These are metric time-series
// with no meaningful flat-table shape, so table mode falls back to JSON.
func printBlob(cmd *cobra.Command, data any) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), data)
	}
	return output.New(cmd.OutOrStdout(), "json").Print(data)
}

func observabilityCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "observability <deployment-id>",
		Short: "Show request/latency/error observability metrics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			rangeSeconds, step := rangeStepValues(cmd)
			res, err := cl.Observability(cmd.Context(), args[0], rangeSeconds, step)
			if err != nil {
				return err
			}
			return printBlob(cmd, res)
		},
	}
	rangeStepFlags(c)
	return c
}

func telemetryCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "telemetry <deployment-id>",
		Short: "Show CPU/memory/replica telemetry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			rangeSeconds, step := rangeStepValues(cmd)
			res, err := cl.Telemetry(cmd.Context(), args[0], rangeSeconds, step)
			if err != nil {
				return err
			}
			return printBlob(cmd, res)
		},
	}
	rangeStepFlags(c)
	return c
}

func cacheConnectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cache-connection <deployment-id>",
		Short: "Show how to connect to a managed cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.CacheConnection(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			rows := [][]string{
				{"FIELD", "VALUE"},
				{"Namespace", res.Namespace},
				{"Service", res.ServiceName},
				{"Secret", res.SecretName},
			}
			if err := ux.Print(cmd.OutOrStdout(), rows); err != nil {
				return err
			}
			if len(res.Details) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Use --output json to see connection details.")
			}
			return nil
		},
	}
}

func buildLogsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "build-logs <deployment-id> <revision-id>",
		Short: "Print the build logs for a revision",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			var tailLines *int
			if cmd.Flags().Changed("tail") {
				v, _ := cmd.Flags().GetInt("tail")
				tailLines = &v
			}
			logs, err := cl.BuildLogs(cmd.Context(), args[0], args[1], tailLines)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), map[string]string{"logs": logs})
			}
			fmt.Fprintln(cmd.OutOrStdout(), logs)
			return nil
		},
	}
	c.Flags().Int("tail", 0, "Show only the last N lines")
	return c
}
