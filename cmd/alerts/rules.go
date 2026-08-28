package alerts

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create <deployment-id>",
		Short: "Create an alert rule for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			flags := cmd.Flags()

			metric := api.PostV1DeploymentsDeploymentIdAlertsJSONBodyMetric(mustString(flags, "metric"))
			if !metric.Valid() {
				return fmt.Errorf("invalid --metric: use cpu, memory, or restarts")
			}
			name := mustString(flags, "name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			threshold, _ := flags.GetFloat32("threshold")

			body := api.PostV1DeploymentsDeploymentIdAlertsJSONRequestBody{
				Metric:    metric,
				Name:      name,
				Threshold: threshold,
			}
			if flags.Changed("comparator") {
				comp := api.PostV1DeploymentsDeploymentIdAlertsJSONBodyComparator(mustString(flags, "comparator"))
				if !comp.Valid() {
					return fmt.Errorf("invalid --comparator: use gt or lt")
				}
				body.Comparator = &comp
			}
			if flags.Changed("severity") {
				sev := api.PostV1DeploymentsDeploymentIdAlertsJSONBodySeverity(mustString(flags, "severity"))
				if !sev.Valid() {
					return fmt.Errorf("invalid --severity: use warning or critical")
				}
				body.Severity = &sev
			}
			if flags.Changed("window") {
				v, _ := flags.GetInt("window")
				body.WindowSeconds = &v
			}
			if flags.Changed("for") {
				v, _ := flags.GetInt("for")
				body.ForDurationSeconds = &v
			}

			rule, err := cl.CreateAlertRule(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), rule)
			}
			return renderRule(cmd, rule)
		},
	}
	c.Flags().String("metric", "", "Metric to watch: cpu, memory, or restarts (required)")
	c.Flags().String("name", "", "Human-readable rule name (required)")
	c.Flags().Float32("threshold", 0, "Threshold: percent (cpu/memory) or count (restarts) (required)")
	c.Flags().String("comparator", "", "Comparison: gt or lt")
	c.Flags().String("severity", "", "Severity: warning or critical")
	c.Flags().Int("window", 0, "Evaluation window in seconds")
	c.Flags().Int("for", 0, "Sustained duration before firing, in seconds")
	return c
}

func updateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update <rule-id>",
		Short: "Update an alert rule (only the flags you pass change)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			flags := cmd.Flags()

			var body api.PatchV1AlertRulesRuleIdJSONRequestBody
			if flags.Changed("name") {
				v := mustString(flags, "name")
				body.Name = &v
			}
			if flags.Changed("metric") {
				m := api.PatchV1AlertRulesRuleIdJSONBodyMetric(mustString(flags, "metric"))
				if !m.Valid() {
					return fmt.Errorf("invalid --metric: use cpu, memory, or restarts")
				}
				body.Metric = &m
			}
			if flags.Changed("threshold") {
				v, _ := flags.GetFloat32("threshold")
				body.Threshold = &v
			}
			if flags.Changed("comparator") {
				comp := api.PatchV1AlertRulesRuleIdJSONBodyComparator(mustString(flags, "comparator"))
				if !comp.Valid() {
					return fmt.Errorf("invalid --comparator: use gt or lt")
				}
				body.Comparator = &comp
			}
			if flags.Changed("severity") {
				sev := api.PatchV1AlertRulesRuleIdJSONBodySeverity(mustString(flags, "severity"))
				if !sev.Valid() {
					return fmt.Errorf("invalid --severity: use warning or critical")
				}
				body.Severity = &sev
			}
			if flags.Changed("window") {
				v, _ := flags.GetInt("window")
				body.WindowSeconds = &v
			}
			if flags.Changed("for") {
				v, _ := flags.GetInt("for")
				body.ForDurationSeconds = &v
			}
			if flags.Changed("enabled") {
				v, _ := flags.GetBool("enabled")
				body.Enabled = &v
			}

			rule, err := cl.UpdateAlertRule(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), rule)
			}
			return renderRule(cmd, rule)
		},
	}
	c.Flags().String("name", "", "Human-readable rule name")
	c.Flags().String("metric", "", "Metric to watch: cpu, memory, or restarts")
	c.Flags().Float32("threshold", 0, "Threshold: percent (cpu/memory) or count (restarts)")
	c.Flags().String("comparator", "", "Comparison: gt or lt")
	c.Flags().String("severity", "", "Severity: warning or critical")
	c.Flags().Int("window", 0, "Evaluation window in seconds")
	c.Flags().Int("for", 0, "Sustained duration before firing, in seconds")
	c.Flags().Bool("enabled", true, "Whether the rule is active")
	return c
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete an alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			if err := cl.DeleteAlertRule(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted alert rule %s.\n", args[0])
			return nil
		},
	}
}

func mustString(flags interface{ GetString(string) (string, error) }, name string) string {
	v, _ := flags.GetString(name)
	return v
}
