package clusters

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	apiPkg "github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	var (
		name              string
		account           string
		provider          string
		clusterType       string
		region            string
		kubernetesVersion string
		nodeInstanceType  string
		podsCIDR          string
		servicesCIDR      string
		desiredVCPU       int
		desiredMemoryGiB  int
		autoscaling       bool
		telemetryDays     int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a cluster",
		Long: "Provision a cluster into a verified cloud account.\n\n" +
			"--type production runs a managed cluster; --type starter runs a lightweight " +
			"self-hosted cluster on a single instance. The command returns as soon as " +
			"provisioning is queued; watch progress with 'okctl clusters status <id>'.\n\n" +
			"Tuning flags (--node-type, --vcpu, --memory, --pods-cidr, --services-cidr, " +
			"--autoscaling, --telemetry-retention) fall back to sensible defaults when omitted.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			body := apiPkg.PostV1ClustersJSONRequestBody{
				ClusterName:       name,
				CloudAccountId:    account,
				Provider:          apiPkg.PostV1ClustersJSONBodyProvider(provider),
				ClusterType:       apiPkg.PostV1ClustersJSONBodyClusterType(resolveClusterType(clusterType)),
				Region:            region,
				KubernetesVersion: kubernetesVersion,
			}
			// Only send tuning fields the caller actually set; otherwise the
			// server applies its own defaults.
			if cmd.Flags().Changed("node-type") {
				body.NodeInstanceType = &nodeInstanceType
			}
			if cmd.Flags().Changed("pods-cidr") {
				body.PodsCidr = &podsCIDR
			}
			if cmd.Flags().Changed("services-cidr") {
				body.ServicesCidr = &servicesCIDR
			}
			if cmd.Flags().Changed("vcpu") {
				body.DesiredVcpu = &desiredVCPU
			}
			if cmd.Flags().Changed("memory") {
				body.DesiredMemoryGiB = &desiredMemoryGiB
			}
			if cmd.Flags().Changed("autoscaling") {
				body.EnableAutoscaling = &autoscaling
			}
			if cmd.Flags().Changed("telemetry-retention") {
				body.TelemetryRetentionDays = &telemetryDays
			}

			created, err := api.CreateCluster(cmd.Context(), body)
			if err != nil {
				return err
			}
			return renderCreated(cmd, created)
		},
	}

	f := cmd.Flags()
	f.StringVar(&name, "name", "", "Cluster name (lowercase alphanumeric + hyphens, required)")
	f.StringVar(&account, "account", "", "Cloud account ID to provision into (required)")
	f.StringVar(&provider, "provider", "aws", "Cloud provider")
	f.StringVar(&clusterType, "type", "production", "Cluster shape: production or starter")
	f.StringVar(&region, "region", "", "Cloud region (required)")
	f.StringVar(&kubernetesVersion, "kubernetes-version", "", "Kubernetes version (required)")
	f.StringVar(&nodeInstanceType, "node-type", "", "Node instance type")
	f.StringVar(&podsCIDR, "pods-cidr", "", "Pods CIDR block")
	f.StringVar(&servicesCIDR, "services-cidr", "", "Services CIDR block")
	f.IntVar(&desiredVCPU, "vcpu", 0, "Desired total vCPU")
	f.IntVar(&desiredMemoryGiB, "memory", 0, "Desired total memory (GiB)")
	f.BoolVar(&autoscaling, "autoscaling", false, "Enable node autoscaling")
	f.IntVar(&telemetryDays, "telemetry-retention", 0, "Telemetry retention (days)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("account")
	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("kubernetes-version")
	return cmd
}

// resolveClusterType maps the user-facing cluster-shape names to the API's
// clusterType values. The raw API values pass through unchanged so power users
// can still supply them directly.
func resolveClusterType(v string) string {
	switch v {
	case "production", "prod":
		return "eks"
	case "starter":
		return "k3s"
	default:
		return v
	}
}

func renderCreated(cmd *cobra.Command, r *apiPkg.CreateClusterResponse) error {
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), r)
	}
	return ux.Print(cmd.OutOrStdout(), [][]string{
		{"FIELD", "VALUE"},
		{"Cluster ID", r.ClusterId},
		{"Status", r.Status},
		{"Status Message", ux.Deref(r.StatusMessage)},
	})
}
