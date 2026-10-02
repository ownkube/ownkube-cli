package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func linkCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "link <app-id> <datastore-id>",
		Short: "Give an app a database's or cache's connection string as a secret variable",
		Long: "Write a database's or cache's connection string into an app's " +
			"environment as a secret variable, then redeploy the app. The " +
			"credential is never printed. The app and the datastore must be in the " +
			"same environment. The variable defaults to DATABASE_URL for a database " +
			"and REDIS_URL for a cache; an existing variable with the same name is " +
			"replaced.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			var envVar *string
			if cmd.Flags().Changed("env-var") {
				v, _ := cmd.Flags().GetString("env-var")
				envVar = &v
			}
			res, err := cl.LinkDatastore(cmd.Context(), args[0], args[1], envVar)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Linked %s into %s as %s. The app is redeploying.\n", args[1], args[0], res.EnvVar)
			return nil
		},
	}
	c.Flags().String("env-var", "", "Environment variable to set (default DATABASE_URL or REDIS_URL)")
	return c
}
