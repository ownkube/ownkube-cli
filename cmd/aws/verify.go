package aws

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func verifyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "verify <account-id>",
		Short: "Verify a connected AWS account's access",
		Long: `Verify that Ownkube can assume the cross-account role.

Normally the CloudFormation stack phones home and verification happens
automatically. Pass --aws-account-id to verify immediately (autonomous mode)
after you have deployed the stack yourself.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			awsAccountID, _ := cmd.Flags().GetString("aws-account-id")

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			res, err := api.VerifyAwsAccount(cmd.Context(), args[0], awsAccountID)
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			if res.Success {
				fmt.Fprintln(cmd.OutOrStdout(), "Verified.")
				return nil
			}
			return fmt.Errorf("verification failed: %s", ux.Deref(res.Error))
		},
	}
	c.Flags().String("aws-account-id", "", "12-digit AWS account ID (verify immediately instead of waiting for phone-home)")
	return c
}
