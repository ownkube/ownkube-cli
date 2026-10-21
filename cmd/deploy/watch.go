package deploy

import (
	"fmt"
	"io"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func watchCmd() *cobra.Command {
	var revisionID string

	cmd := &cobra.Command{
		Use:   "watch [app]",
		Short: "Follow the latest revision until it's live or fails",
		Long: "Follow a deployment's latest revision (or --revision) through queued, " +
			"building and deploying, streaming its build log, until it goes live or " +
			"fails. Exits non-zero when the build fails or is superseded. With -o " +
			"json/yaml the log is not streamed; the final revision is printed instead.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			ctx, id, out := cmd.Context(), args[0], cmd.OutOrStdout()

			revs, err := cl.ListRevisions(ctx, id)
			if err != nil {
				return err
			}
			rev, err := pickRevision(revs, revisionID)
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				followErr := ux.FollowRevision(ctx, cl, io.Discard, id, rev.Id)
				if revs, err := cl.ListRevisions(ctx, id); err == nil {
					if r, err := pickRevision(revs, rev.Id); err == nil {
						rev = r
					}
				}
				if err := ux.Print(out, rev); err != nil {
					return err
				}
				return followErr
			}

			header := "Revision " + rev.Id
			if c := revisionCommit(*rev); c != "" {
				header += ": " + c
			}
			fmt.Fprintln(out, header)
			return ux.FollowRevision(ctx, cl, out, id, rev.Id)
		},
	}
	cmd.Flags().StringVar(&revisionID, "revision", "", "Revision to follow (default: the latest)")
	return cmd
}

// pickRevision returns the revision with the given id, or the latest when id
// is empty.
func pickRevision(revs []api.Revision, id string) (*api.Revision, error) {
	if id == "" {
		if latest := ux.LatestRevision(revs); latest != nil {
			return latest, nil
		}
		return nil, fmt.Errorf("this deployment has no revisions yet")
	}
	for i := range revs {
		if revs[i].Id == id {
			return &revs[i], nil
		}
	}
	return nil, fmt.Errorf("revision %s not found", id)
}
