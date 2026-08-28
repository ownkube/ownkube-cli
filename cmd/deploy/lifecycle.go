package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// renderLifecycle prints the shared deployment shape returned by the maintenance,
// rename and auto-deploy toggles.
func renderLifecycle(cmd *cobra.Command, d *api.LifecycleDeploymentResult) error {
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"ID", d.Id},
		{"Name", d.Name},
		{"Status", d.Status},
		{"Message", d.StatusMessage},
		{"Cluster", d.ClusterId},
		{"Environment", d.EnvironmentId},
	})
}

func restartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <deployment-id>",
		Short: "Roll the workload, restarting every pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.Restart(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restarted %g workload(s) at %s.\n", res.Restarted, res.RestartedAt)
			return nil
		},
	}
}

func rebuildCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "rebuild <deployment-id>",
		Short: "Queue a fresh build of the current source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			var note *string
			if cmd.Flags().Changed("note") {
				v, _ := cmd.Flags().GetString("note")
				note = &v
			}
			res, err := cl.Rebuild(cmd.Context(), args[0], note)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Queued rebuild: revision %s (%s @ %s).\n", res.RevisionId, res.Branch, res.GitSha)
			return nil
		},
	}
	c.Flags().String("note", "", "Optional note recorded on the queued revision")
	return c
}

func restoreCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "restore <deployment-id>",
		Short: "Restore a managed database to a point in time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			var targetTime *string
			if cmd.Flags().Changed("target-time") {
				v, _ := cmd.Flags().GetString("target-time")
				targetTime = &v
			}
			res, err := cl.RestoreDatabase(cmd.Context(), args[0], targetTime)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restore started: %t.\n", res.Started)
			return nil
		},
	}
	c.Flags().String("target-time", "", "RFC3339 recovery target (default: latest available point)")
	return c
}

func maintenanceCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "maintenance <deployment-id>",
		Short: "Toggle the maintenance page for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			enabled, _ := cmd.Flags().GetBool("enabled")
			body := api.SetMaintenanceBody{Enabled: enabled}
			if cmd.Flags().Changed("message") {
				v, _ := cmd.Flags().GetString("message")
				body.Message = &v
			}
			res, err := cl.SetMaintenance(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return renderLifecycle(cmd, res)
		},
	}
	c.Flags().Bool("enabled", false, "Whether the maintenance page is shown")
	c.Flags().String("message", "", "Optional line shown on the maintenance page")
	return c
}

func renameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <deployment-id> <display-name>",
		Short: "Set the cosmetic display label (empty clears it)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			displayName := ""
			if len(args) == 2 {
				displayName = args[1]
			}
			res, err := cl.RenameDeployment(cmd.Context(), args[0], displayName)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return renderLifecycle(cmd, res)
		},
	}
}

func autoDeployCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auto-deploy <deployment-id>",
		Short: "Toggle automatic deploys on new commits",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			enabled, _ := cmd.Flags().GetBool("enabled")
			res, err := cl.SetAutoDeploy(cmd.Context(), args[0], enabled)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return renderLifecycle(cmd, res)
		},
	}
	c.Flags().Bool("enabled", true, "Whether auto-deploy is active")
	return c
}
