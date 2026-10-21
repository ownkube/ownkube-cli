package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func revisionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revisions [app]",
		Short: "List revisions for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}

			revs, err := cl.ListRevisions(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), revs)
			}
			if len(revs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No revisions found.")
				return nil
			}

			rows := [][]string{{"ID", "STATUS", "COMMIT", "MESSAGE", "AUTHOR", "IMAGE", "SOURCE", "NOTE"}}
			for _, r := range revs {
				rows = append(rows, []string{
					r.Id, r.Status, revisionSha(r),
					truncateText(ux.Deref(r.CommitMessage), 50), ux.Deref(r.CommitAuthor),
					revisionImage(r), ux.Deref(r.Source), ux.Deref(r.Note),
				})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

// revisionImage renders the image a revision runs as repository:tag. An
// Ownkube-built image has no reference to show.
func revisionImage(r api.Revision) string {
	if r.Image == "" {
		if r.ImageSource != nil {
			return "(built by Ownkube)"
		}
		return ""
	}
	if r.Tag == "" {
		return r.Image
	}
	return r.Image + ":" + r.Tag
}
