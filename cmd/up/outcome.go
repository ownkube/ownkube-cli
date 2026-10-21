package up

import (
	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

// outcome is the single JSON result `up --json` / `up --ci` prints.
type outcome struct {
	DeploymentID string `json:"deploymentId" yaml:"deploymentId"`
	RevisionID   string `json:"revisionId,omitempty" yaml:"revisionId,omitempty"`
	Status       string `json:"status,omitempty" yaml:"status,omitempty"`
	URL          string `json:"url,omitempty" yaml:"url,omitempty"`
	Error        string `json:"error,omitempty" yaml:"error,omitempty"`
	Result       any    `json:"result" yaml:"result"`
}

// printOutcome follows the revision (build log on stderr) unless follow is
// false, then prints one JSON result to stdout. It returns the follow error so
// a failed build exits non-zero.
func printOutcome(cmd *cobra.Command, cl *client.Client, depID, revisionID string, result any, follow bool) error {
	ctx := cmd.Context()
	o := outcome{DeploymentID: depID, RevisionID: revisionID, Result: result}

	var followErr error
	if follow && revisionID != "" {
		followErr = ux.FollowRevision(ctx, cl, cmd.ErrOrStderr(), depID, revisionID)
		if followErr != nil {
			o.Error = followErr.Error()
		}
		if revs, err := cl.ListRevisions(ctx, depID); err == nil {
			for _, r := range revs {
				if r.Id == revisionID {
					o.Status = r.Status
				}
			}
		}
	}
	if d, err := cl.GetDeployment(ctx, depID); err == nil && d.PublicHostname != "" {
		o.URL = "https://" + d.PublicHostname
	}
	if err := ux.Print(cmd.OutOrStdout(), o); err != nil {
		return err
	}
	return followErr
}
