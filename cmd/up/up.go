// Package up implements `okctl up`: build and deploy the current directory's
// local working-tree source. It archives the working tree (honouring
// .gitignore), uploads the tarball to a short-lived pre-signed slot, then
// triggers an in-cluster build+deploy of that source and follows the build.
// In an unlinked directory it creates the app first (create.go), with defaults
// resolved server-side, and links the directory to it.
//
// Unlike `okctl deploy` (which builds a committed git revision), `up` ships
// whatever is on disk right now — uncommitted edits included — for fast
// inner-loop iteration.
package up

import (
	"fmt"
	"os"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/sourcearchive"
	"github.com/spf13/cobra"
)

// New builds the `okctl up` command.
func New() *cobra.Command {
	var (
		serviceFlag string
		noteFlag    string
		pathFlag    string
		noFollow    bool
		repoFlag    string
		branchFlag  string
		jsonFlag    bool
		ciFlag      bool
		create      createFlags
	)

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Build and deploy the current directory",
		Long: "Archive the current directory's working tree, upload it, and build + " +
			"deploy it — uncommitted changes included. The target deployment is taken " +
			"from --service, or inferred from this directory's link (see 'okctl link').\n\n" +
			"In a directory that isn't linked yet, 'up' creates a new web app on Ownkube " +
			"Compute with sensible defaults (named after the directory, in the region you " +
			"already use, port from the Dockerfile or framework, public URL), prints what " +
			"it chose, and links the directory so the next 'up' redeploys it. --name, " +
			"--region, --port, --type, --public, --project and --environment override " +
			"those defaults.\n\n" +
			"Files ignored by .gitignore are never uploaded, nor are .git, node_modules " +
			"and .env* files. The command follows the build until it goes live; pass " +
			"--no-follow to return as soon as the build is queued.\n\n" +
			"--repo/--branch switch an existing app to build from a connected " +
			"repository instead (nothing is uploaded; later pushes deploy it), the " +
			"same as 'okctl deploy source'. Either one alone is filled in from this " +
			"checkout's origin remote and current branch.\n\n" +
			"--json (or --ci) is for scripts and CI: nothing prompts, the build log goes " +
			"to stderr, stdout gets one JSON result once the revision is live or fails, " +
			"and the exit code is non-zero when it fails. (-o json alone prints the " +
			"queued build and returns without following.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			finalJSON := jsonFlag || ciFlag
			if finalJSON {
				ux.SetOutputFormat("json")
			}

			creds, err := ux.RequireAuth()
			if err != nil {
				return err
			}

			dir := pathFlag
			if dir == "" {
				dir, err = os.Getwd()
				if err != nil {
					return err
				}
			}

			// No --service and no link: create the app from this directory.
			creating := false
			if serviceFlag == "" {
				linked, err := linkedDeploymentID(dir)
				if err != nil {
					return err
				}
				creating = linked == ""
			}
			var depID string
			if !creating {
				if err := rejectCreateFlags(cmd); err != nil {
					return err
				}
			}

			cl, err := client.New(ux.APIURL(), creds.APIKey, ux.Organization())
			if err != nil {
				return err
			}
			if !creating {
				if depID, err = ux.ResolveApp(cmd.Context(), cl, serviceFlag, ux.Scope{}, dir); err != nil {
					return err
				}
			}

			structured := ux.IsStructured()
			out := cmd.OutOrStdout()

			if repoFlag != "" || branchFlag != "" {
				if creating {
					return fmt.Errorf("--repo connects an existing app; create one from the repository with 'okctl deploy create --repo', or run 'okctl up' without it first")
				}
				if noteFlag != "" {
					return fmt.Errorf("--note applies to uploaded source, not --repo")
				}
				return upFromRepo(cmd, cl, dir, depID, repoFlag, branchFlag, noFollow, finalJSON)
			}

			// 1. Mint an upload slot. The server refuses here, before any
			// upload, when the wallet is empty (with a link to add credit).
			slot, err := cl.PresignSourceUpload(ctx, depID)
			if err != nil {
				return err
			}

			// 2. Archive the working tree to a temp tarball.
			if !structured {
				fmt.Fprintln(out, "Packaging source...")
			}
			archivePath, size, err := sourcearchive.Create(ctx, dir)
			if err != nil {
				return err
			}
			defer os.Remove(archivePath)
			if err := sourcearchive.CheckSize(size, int64(slot.MaxBytes)); err != nil {
				return err
			}

			// 3. Upload it straight to object storage via the pre-signed URL.
			if !structured {
				fmt.Fprintf(out, "Uploading source (%s)...\n", sourcearchive.HumanBytes(size))
			}
			f, err := os.Open(archivePath)
			if err != nil {
				return fmt.Errorf("opening archive: %w", err)
			}
			// Close explicitly after upload rather than defer — the follow loop
			// below can run for minutes and there's no reason to hold the fd.
			uploadErr := cl.UploadSource(ctx, slot.UploadUrl, slot.ContentType, f, size)
			f.Close()
			if uploadErr != nil {
				return uploadErr
			}

			// 4. Trigger the in-cluster build+deploy of the uploaded source.
			var note *string
			if noteFlag != "" {
				note = &noteFlag
			}
			if creating {
				created, err := createApp(ctx, cmd, cl, &create, dir, slot.UploadId, note)
				if err != nil {
					return err
				}
				if finalJSON {
					return printOutcome(cmd, cl, created.DeploymentId, created.RevisionId, created, !noFollow)
				}
				if structured {
					return ux.Print(out, created)
				}
				fmt.Fprintf(out, "Build queued (revision %s).\n", created.RevisionId)
				if !noFollow {
					if err := ux.FollowRevision(ctx, cl, out, created.DeploymentId, created.RevisionId); err != nil {
						return err
					}
				}
				printPushDeployTip(cmd, cl, dir, created.DeploymentId)
				return nil
			}
			result, err := cl.DeployFromUpload(ctx, depID, slot.UploadId, note)
			if err != nil {
				return err
			}

			if finalJSON {
				return printOutcome(cmd, cl, depID, result.RevisionId, result, !noFollow)
			}
			if structured {
				return ux.Print(out, result)
			}
			fmt.Fprintf(out, "Build queued (revision %s).\n", result.RevisionId)

			if !noFollow {
				if err := ux.FollowRevision(ctx, cl, out, depID, result.RevisionId); err != nil {
					return err
				}
			}
			printPushDeployTip(cmd, cl, dir, depID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&serviceFlag, "service", "s", "", "App name or id to deploy to (defaults to this directory's link)")
	cmd.Flags().StringVar(&noteFlag, "note", "", "Note to record on the revision")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Directory to deploy (defaults to the current directory)")
	cmd.Flags().BoolVar(&noFollow, "no-follow", false, "Return once the build is queued instead of following it")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Build an existing app from this connected repository (owner/repo) instead of uploading; default with --branch: this checkout's origin")
	cmd.Flags().StringVar(&branchFlag, "branch", "", "Branch to build and deploy on push (with --repo); default: the current branch")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Follow the build (log on stderr) and print one JSON result; non-zero exit on failure")
	cmd.Flags().BoolVar(&ciFlag, "ci", false, "Same as --json, for CI pipelines")
	create.register(cmd)
	return cmd
}
