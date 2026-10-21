package up

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/gitrepo"
	"github.com/spf13/cobra"
)

// upFromRepo switches an existing app to build from a connected repository
// (the same call as `okctl deploy source`) and follows the build it starts.
func upFromRepo(cmd *cobra.Command, cl *client.Client, dir, depID, repo, branch string, noFollow, finalJSON bool) error {
	ctx, out := cmd.Context(), cmd.OutOrStdout()
	repository, br, err := ux.ResolveRepoBranch(ctx, dir, repo, branch)
	if err != nil {
		return err
	}
	res, err := cl.SetDeploymentSource(ctx, depID, api.SetDeploymentSourceBody{Repository: repository, Branch: br})
	if err != nil {
		return err
	}
	if finalJSON {
		return printOutcome(cmd, cl, depID, res.RevisionId, res, !noFollow)
	}
	if ux.IsStructured() {
		return ux.Print(out, res)
	}
	fmt.Fprintf(out, "Connected %s to %s@%s; pushes to %s now deploy it.\n", depID, res.Repository, res.Branch, res.Branch)
	if res.RevisionId == "" {
		return nil
	}
	fmt.Fprintf(out, "Build queued (revision %s).\n", res.RevisionId)
	if noFollow {
		return nil
	}
	return ux.FollowRevision(ctx, cl, out, depID, res.RevisionId)
}

// printPushDeployTip suggests connecting the repository when dir is a git
// checkout and the app doesn't build from a repository yet. Best-effort: any
// lookup failure just skips the tip.
func printPushDeployTip(cmd *cobra.Command, cl *client.Client, dir, depID string) {
	if ux.IsStructured() {
		return
	}
	info, err := gitrepo.Detect(cmd.Context(), dir)
	if err != nil {
		return
	}
	d, err := cl.GetDeployment(cmd.Context(), depID)
	if err != nil || (d.Source != nil && d.Source.Type == api.DeploymentSourceTypeGit) {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Tip: deploy every push to %s with 'okctl deploy source %s'.\n", info.Repository, depID)
}
