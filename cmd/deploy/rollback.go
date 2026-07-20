package deploy

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func rollbackCmd() *cobra.Command {
	var note string

	cmd := &cobra.Command{
		Use:   "rollback <deployment-id> <revision-id>",
		Short: "Roll a deployment back to an earlier revision",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var notePtr *string
			if note != "" {
				notePtr = &note
			}
			rev, err := api.Rollback(cmd.Context(), args[0], args[1], notePtr)
			if err != nil {
				return err
			}
			return renderRevision(cmd, rev)
		},
	}

	cmd.Flags().StringVar(&note, "note", "", "Optional note for the revision")
	return cmd
}
