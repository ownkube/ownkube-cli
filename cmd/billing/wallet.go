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
		Short: "Show Ownkube Compute credit status",
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
				{"Wallet Remaining", usd(res.Balance.RemainingUsd)},
				{"Top-up Minimum", usd(res.TopUpMinUsd)},
			})
		},
	}
	c.AddCommand(creditRedeemCmd())
	return c
}

func creditRedeemCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "redeem <code>",
		Short: "Redeem a promo code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.RedeemPromo(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if !res.Ok {
				message := "That promo code could not be redeemed."
				if res.Message != nil {
					message = *res.Message
				}
				fmt.Fprintln(cmd.ErrOrStderr(), message)
				return fmt.Errorf("promo code not redeemed")
			}
			out := cmd.OutOrStdout()
			var amount float32
			if res.AmountUsd != nil {
				amount = *res.AmountUsd
			}
			var remaining float32
			if res.Balance != nil {
				remaining = res.Balance.RemainingUsd
			}
			fmt.Fprintf(out, "Added %s in credit. Balance: %s.\n", usd(amount), usd(remaining))
			return nil
		},
	}
}
