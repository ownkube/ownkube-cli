package billing

import (
	"fmt"
	"strconv"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func subscribeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "subscribe <personal|team>",
		Short: "Start a subscription checkout (loads wallet credit each cycle)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tier := api.SubscriptionCheckoutBodyTier(args[0])
			if !tier.Valid() {
				return fmt.Errorf("invalid tier %q: use personal or team", args[0])
			}
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.CheckoutSubscription(cmd.Context(), tier)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return openCheckoutURL(cmd, res.Url)
		},
	}
	c.Flags().Bool("no-browser", false, "Print the checkout URL but do not open a browser")
	return c
}

func topUpCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "top-up <amount-usd>",
		Short: "Start a wallet top-up checkout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := strconv.ParseFloat(args[0], 32)
			if err != nil {
				return fmt.Errorf("invalid amount %q: %w", args[0], err)
			}
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.CheckoutTopUp(cmd.Context(), float32(amount))
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return openCheckoutURL(cmd, res.Url)
		},
	}
	c.Flags().Bool("no-browser", false, "Print the checkout URL but do not open a browser")
	return c
}

func portalCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "portal",
		Short: "Open the billing portal to manage payment methods and invoices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			res, err := api.Portal(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			return openCheckoutURL(cmd, res.Url)
		},
	}
	c.Flags().Bool("no-browser", false, "Print the portal URL but do not open a browser")
	return c
}
