// Package domains exposes the `okctl domains` command tree — linking custom
// hostnames to a deployment, verifying their certificates, and unlinking them.
package domains

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// New returns the `okctl domains` command with every subcommand attached.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:     "domains",
		Aliases: []string{"custom-domains"},
		Short:   "Manage custom domains linked to a deployment",
	}
	root.AddCommand(listCmd(), addCmd(), verifyCmd(), deleteCmd())
	return root
}

// printDNS appends the DNS records the user must create for a linked host.
func printDNS(rows [][]string, d api.CustomDomainHostDns) [][]string {
	return append(rows,
		[]string{"Routing Record", fmt.Sprintf("%s %s -> %s", d.RoutingRecordName, string(d.RoutingRecordType), d.RoutingTarget)},
		[]string{"ACME Challenge", fmt.Sprintf("%s -> %s", d.AcmeChallengeName, d.AcmeDelegationTarget)},
	)
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <deployment-id>",
		Short: "List linked custom domains and available parent domains",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := cl.ListCustomDomains(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			if len(res.Linked) == 0 {
				fmt.Fprintln(out, "No linked domains.")
			} else {
				rows := [][]string{{"ID", "HOSTNAME", "STATUS", "CERT"}}
				for i := range res.Linked {
					d := &res.Linked[i]
					rows = append(rows, []string{d.Id, d.Hostname, d.Status, d.CertStatus})
				}
				if err := ux.Print(out, rows); err != nil {
					return err
				}
			}
			if len(res.AvailableParents) > 0 {
				rows := [][]string{{"AVAILABLE PARENT", "CERT"}}
				for _, p := range res.AvailableParents {
					rows = append(rows, []string{p.WildcardDomain, p.CertStatus})
				}
				fmt.Fprintln(out)
				return ux.Print(out, rows)
			}
			return nil
		},
	}
}

func addCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <deployment-id> <hostname>",
		Short: "Link a custom hostname to a deployment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			d, err := cl.LinkCustomDomain(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), d)
			}
			rows := [][]string{
				{"FIELD", "VALUE"},
				{"ID", d.Id},
				{"Hostname", d.Hostname},
				{"Status", d.Status},
				{"Cert", d.CertStatus},
			}
			rows = printDNS(rows, d.Dns)
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

func verifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify <domain-id>",
		Short: "Re-check a linked domain's DNS and certificate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			certStatus, err := cl.VerifyCustomDomain(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), map[string]string{"certStatus": certStatus})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Certificate status: %s\n", certStatus)
			return nil
		},
	}
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <domain-id>",
		Aliases: []string{"unlink"},
		Short:   "Unlink a custom domain from its deployment",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			if err := cl.UnlinkCustomDomain(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unlinked custom domain %s.\n", args[0])
			return nil
		},
	}
}
