package marketplace

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

func checkCmd() *cobra.Command {
	var (
		inputs []string
		name   string
	)
	cmd := &cobra.Command{
		Use:   "check <slug>",
		Short: "Check inputs and name before deploying (creates nothing)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			values, err := parseInputs(inputs)
			if err != nil {
				return err
			}
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			body := api.MarketplaceCheckBody{BaseName: optional(name)}
			if len(values) > 0 {
				m := make(map[string]api.MarketplaceCheckBody_Inputs_AdditionalProperties, len(values))
				for k, v := range values {
					var p api.MarketplaceCheckBody_Inputs_AdditionalProperties
					if err := p.FromMarketplaceCheckBodyInputs0(v); err != nil {
						return err
					}
					m[k] = p
				}
				body.Inputs = &m
			}
			res, err := cl.CheckMarketplaceApp(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if res.Ok {
				fmt.Fprintln(cmd.OutOrStdout(), "Ready to deploy.")
				return nil
			}
			for _, e := range res.Errors {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", e)
			}
			return fmt.Errorf("%s isn't ready to deploy", args[0])
		},
	}
	cmd.Flags().StringArrayVar(&inputs, "input", nil, "Input KEY=VALUE (repeatable; see 'okctl marketplace get <slug>')")
	cmd.Flags().StringVar(&name, "name", "", "Name prefix for every piece (default: the app's slug)")
	return cmd
}

func deployCmd() *cobra.Command {
	var (
		inputs                                []string
		name, region, cluster                 string
		project, environment, dbBox, cacheBox string
	)
	cmd := &cobra.Command{
		Use:   "deploy <slug>",
		Short: "Deploy an app with its database and cache",
		Long: "Deploy a marketplace app. On Ownkube Compute pass --region (from " +
			"'okctl regions list'), plus --db-box / --cache-box (from 'okctl boxes " +
			"list') when the app includes a database or cache. To use your own " +
			"cluster pass --cluster instead.",
		Example: "  okctl marketplace deploy n8n --region us-east-1 --db-box db-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if region != "" && cluster != "" {
				return fmt.Errorf("pass --region or --cluster, not both")
			}
			values, err := parseInputs(inputs)
			if err != nil {
				return err
			}
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			body := api.MarketplaceDeployBody{
				BaseName:      optional(name),
				Region:        optional(region),
				ClusterId:     optional(cluster),
				ProjectId:     optional(project),
				EnvironmentId: optional(environment),
				DbSkuId:       optional(dbBox),
				CacheSkuId:    optional(cacheBox),
			}
			if len(values) > 0 {
				m := make(map[string]api.MarketplaceDeployBody_Inputs_AdditionalProperties, len(values))
				for k, v := range values {
					var p api.MarketplaceDeployBody_Inputs_AdditionalProperties
					if err := p.FromMarketplaceDeployBodyInputs0(v); err != nil {
						return err
					}
					m[k] = p
				}
				body.Inputs = &m
			}
			res, err := cl.DeployMarketplaceApp(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Deploying %s (install %s).\n", res.Slug, res.TemplateInstanceId)
			rows := [][]string{{"ID", "NAME", "REF", "TYPE", "URL"}}
			for _, d := range res.Deployments {
				url := ""
				if d.PublicHostname != "" {
					url = "https://" + d.PublicHostname
				}
				rows = append(rows, []string{d.DeploymentId, d.Name, d.Ref, string(d.ResourceType), url})
			}
			if err := ux.Print(w, rows); err != nil {
				return err
			}
			fmt.Fprintln(w, "Follow progress with 'okctl deploy status <id>'.")
			return nil
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&inputs, "input", nil, "Input KEY=VALUE (repeatable; see 'okctl marketplace get <slug>')")
	f.StringVar(&name, "name", "", "Name prefix for every piece (default: the app's slug)")
	f.StringVar(&region, "region", "", "Ownkube Compute region id from 'okctl regions list'")
	f.StringVar(&cluster, "cluster", "", "Your own cluster ID (instead of --region)")
	f.StringVar(&project, "project", "", "Project ID (default: your default project)")
	f.StringVar(&environment, "environment", "", "Environment ID (default: the project's default environment)")
	f.StringVar(&dbBox, "db-box", "", "Database box id from 'okctl boxes list --kind database'")
	f.StringVar(&cacheBox, "cache-box", "", "Cache box id from 'okctl boxes list --kind cache'")
	return cmd
}

func deleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <install-id>",
		Short: "Delete a deployed marketplace app and everything it created",
		Long: "Delete every piece one marketplace deploy created, including its " +
			"database and its data. The install id is printed by 'okctl " +
			"marketplace deploy' and shown as templateInstanceId by " +
			"'okctl deploy get <id> -o json'.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			if !yes {
				ok, err := prompt.Confirm(fmt.Sprintf("Delete marketplace app %s and everything it created, including its data?", args[0]))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}
			res, err := cl.DeleteMarketplaceApp(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d deployments.\n", int(res.Count))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

// parseInputs turns repeated KEY=VALUE flags into a map. Values go up as
// strings; the server coerces them to the input's declared type.
func parseInputs(pairs []string) (map[string]string, error) {
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --input %q: expected KEY=VALUE", p)
		}
		out[k] = v
	}
	return out, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
