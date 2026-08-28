package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// subdomainCmd groups the address-label commands under the platform domain.
func subdomainCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "subdomain",
		Short: "Manage a deployment's address label under the platform domain",
	}
	root.AddCommand(subdomainSetCmd(), subdomainCheckCmd(), subdomainSuggestCmd())
	return root
}

func subdomainSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <deployment-id> <subdomain>",
		Short: "Move the deployment to a new address label",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.ChangeSubdomain(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if !res.Changed {
				fmt.Fprintf(cmd.OutOrStdout(), "No change: already at %s.\n", res.Hostname)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Address is now %s.\n", res.Hostname)
			return nil
		},
	}
}

func subdomainCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <deployment-id> <subdomain>",
		Short: "Check whether an address label is available",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.CheckSubdomain(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Subdomain", res.Subdomain},
				{"Hostname", res.Hostname},
				{"Available", fmt.Sprintf("%t", res.Available)},
				{"Current", fmt.Sprintf("%t", res.Current)},
				{"Message", res.Message},
			})
		},
	}
}

func subdomainSuggestCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "suggest <deployment-id>",
		Short: "Suggest available address labels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			var desired *string
			if cmd.Flags().Changed("desired") {
				v, _ := cmd.Flags().GetString("desired")
				desired = &v
			}
			var limit *int
			if cmd.Flags().Changed("limit") {
				v, _ := cmd.Flags().GetInt("limit")
				limit = &v
			}
			res, err := cl.SuggestSubdomains(cmd.Context(), args[0], desired, limit)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if len(res.Suggestions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No suggestions.")
				return nil
			}
			rows := [][]string{{"SUBDOMAIN", "HOSTNAME", "KIND"}}
			for _, s := range res.Suggestions {
				rows = append(rows, []string{s.Subdomain, s.Hostname, s.Kind})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
	c.Flags().String("desired", "", "Seed suggestions from a partial label")
	c.Flags().Int("limit", 0, "Maximum number of suggestions")
	return c
}
