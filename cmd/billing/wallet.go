package billing

import (
	"fmt"
	"sort"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func walletCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wallet",
		Short: "Show the prepaid wallet balance",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetWallet(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if !res.Enabled {
				fmt.Fprintln(cmd.OutOrStdout(), "Ownkube Compute is not enabled for this organization.")
				return nil
			}
			rows := [][]string{
				{"FIELD", "VALUE"},
				{"Remaining", usd(res.Balance.RemainingUsd)},
				{"Granted", usd(res.Balance.GrantedUsd)},
				{"Consumed", usd(res.Balance.ConsumedUsd)},
				{"Exhausted", fmt.Sprintf("%t", res.Balance.Exhausted)},
			}
			reasons := make([]string, 0, len(res.LoadedInByReason))
			for k := range res.LoadedInByReason {
				reasons = append(reasons, k)
			}
			sort.Strings(reasons)
			for _, k := range reasons {
				rows = append(rows, []string{"Loaded (" + k + ")", usd(res.LoadedInByReason[k])})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

func creditCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "credit",
		Short: "Show signup-credit status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.GetCredit(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if !res.Enabled {
				fmt.Fprintln(cmd.OutOrStdout(), "Ownkube Compute is not enabled for this organization.")
				return nil
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Signup Credit", usd(res.AmountUsd)},
				{"Claimed", fmt.Sprintf("%t", res.Claimed)},
				{"Wallet Remaining", usd(res.Balance.RemainingUsd)},
				{"Top-up Minimum", usd(res.TopUpMinUsd)},
			})
		},
	}
	c.AddCommand(creditClaimCmd())
	return c
}

func creditClaimCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claim",
		Short: "Claim the one-time signup credit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.ClaimCredit(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			if res.AlreadyClaimed {
				fmt.Fprintln(out, "Signup credit was already claimed — nothing new granted.")
			} else {
				fmt.Fprintf(out, "Claimed %s in signup credit.\n", usd(res.AmountUsd))
			}
			fmt.Fprintf(out, "Wallet remaining: %s\n", usd(res.Balance.RemainingUsd))
			return nil
		},
	}
}
