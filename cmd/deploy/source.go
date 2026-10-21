package deploy

import (
	"fmt"
	"os"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

func sourceCmd() *cobra.Command {
	var (
		repo, branch, connection string
		watchPaths               []string
		noDeploy, clearFlag, yes bool
	)

	cmd := &cobra.Command{
		Use:   "source [app]",
		Short: "Connect an app to a repository so pushes deploy it",
		Long: "Connect a web app, worker or job to a GitHub or GitLab repository and " +
			"branch, or switch the one it builds from, then build and deploy it. " +
			"Settings, secrets, linked databases, address and domains are kept, and " +
			"the running version keeps serving until the new build is live. Later " +
			"pushes to the branch deploy automatically.\n\n" +
			"With no --repo, the repository and branch are read from this checkout's " +
			"'origin' remote and current branch. The repository must be visible to an " +
			"account connected in the dashboard; see 'okctl vcs connections'.\n\n" +
			"--watch-paths takes gitignore-style globs (src/**, !docs/**; the last " +
			"match wins): a push that touches none of them is skipped. Pass " +
			"--watch-paths '' to build on every push again. --wait-for-ci holds push " +
			"deploys until the commit's checks pass.\n\n" +
			"--clear disconnects the repository: pushes stop deploying and the app " +
			"keeps running its current version.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if clearFlag {
				return clearSource(cmd, id, yes)
			}

			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}

			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			repository, br, err := ux.ResolveRepoBranch(cmd.Context(), dir, repo, branch)
			if err != nil {
				return err
			}

			body := api.SetDeploymentSourceBody{Repository: repository, Branch: br}
			flags := cmd.Flags()
			if connection != "" {
				body.ConnectionId = &connection
			}
			if flags.Changed("context") {
				v, _ := flags.GetString("context")
				body.ContextPath = &v
			}
			if flags.Changed("dockerfile") {
				v, _ := flags.GetString("dockerfile")
				body.DockerfilePath = &v
			}
			if flags.Changed("watch-paths") {
				paths := nonEmpty(watchPaths)
				body.WatchPaths = &paths
			}
			if flags.Changed("wait-for-ci") {
				v, _ := flags.GetBool("wait-for-ci")
				body.WaitForChecks = &v
			}
			if noDeploy {
				f := false
				body.Deploy = &f
			}
			if body.Commit, err = commitFlag(cmd); err != nil {
				return err
			}
			if noDeploy && body.Commit != nil {
				return fmt.Errorf("--commit builds now, so it can't be combined with --no-deploy")
			}

			if !yes {
				ok, err := prompt.Confirm(fmt.Sprintf("Connect %s to %s@%s?", id, repository, br))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			res, err := cl.SetDeploymentSource(cmd.Context(), id, body)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Connected %s to %s@%s (%s).\n", id, res.Repository, res.Branch, res.Provider)
			if res.RevisionId == "" {
				fmt.Fprintf(out, "Nothing is building yet; the next push to %s deploys it.\n", res.Branch)
				return nil
			}
			fmt.Fprintf(out, "Building revision %s from %s.\n", res.RevisionId, shortSha(res.GitSha))
			printWatchHint(out, id)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&repo, "repo", "", "Repository as owner/repo (GitLab: group/subgroup/project); default: this checkout's origin")
	f.StringVar(&branch, "branch", "", "Branch to deploy and track; default: the current branch")
	f.StringVar(&connection, "connection", "", "Connected account to use (see 'okctl vcs connections'); default: any that can see the repository")
	f.String("context", "", "Root directory to build from, relative to the repository root ('' for the root)")
	f.String("dockerfile", "", "Dockerfile path relative to the root directory ('' for the default)")
	f.StringSliceVar(&watchPaths, "watch-paths", nil, "Comma-separated globs a push must touch to deploy ('' to build on every push)")
	f.Bool("wait-for-ci", false, "Hold push deploys until the commit's CI checks pass")
	f.String("commit", "", "Deploy this commit SHA instead of the branch tip (the branch is still tracked)")
	f.BoolVar(&noDeploy, "no-deploy", false, "Save the source without building; the next push or rebuild deploys it")
	f.BoolVar(&clearFlag, "clear", false, "Disconnect the repository; the app keeps running its current version")
	f.BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

func clearSource(cmd *cobra.Command, id string, yes bool) error {
	for _, name := range []string{"repo", "branch", "connection", "context", "dockerfile", "watch-paths", "wait-for-ci", "commit", "no-deploy"} {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("--clear can't be combined with --%s", name)
		}
	}
	cl, err := ux.RequireClient()
	if err != nil {
		return err
	}
	if !yes {
		ok, err := prompt.Confirm(fmt.Sprintf("Disconnect %s from its repository? Pushes will stop deploying it.", id))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}
	res, err := cl.ClearDeploymentSource(cmd.Context(), id)
	if err != nil {
		return err
	}
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), res)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Disconnected %s from its repository; it keeps running its current version.\n", id)
	return nil
}

// nonEmpty drops blank entries, so --watch-paths "" sends an empty list.
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
