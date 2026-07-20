package deploy

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func promoteCmd() *cobra.Command {
	var (
		to   string
		note string
	)

	cmd := &cobra.Command{
		Use:   "promote <deployment-id>",
		Short: "Promote a deployment's live revision to a target deployment",
		Long: "Ship the source deployment's currently live revision to a target " +
			"deployment (for example, staging → production).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var notePtr *string
			if note != "" {
				notePtr = &note
			}
			rev, err := api.Promote(cmd.Context(), args[0], to, notePtr)
			if err != nil {
				return err
			}
			return renderRevision(cmd, rev)
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Target deployment ID")
	cmd.Flags().StringVar(&note, "note", "", "Optional note for the revision")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
