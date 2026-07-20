package registries

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <registry-id>",
		Short: "Get registry details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			r, err := api.GetRegistry(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), r)
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"ID", r.Id},
				{"Provider", r.Provider},
				{"Account", r.Account},
				{"Account Type", ux.Deref(r.AccountType)},
				{"Status", ux.Deref(r.Status)},
			})
		},
	}
}
