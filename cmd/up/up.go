// Package up implements `okctl up`: build and deploy the current directory's
// local working-tree source. It archives the working tree (honouring
// .gitignore), uploads the tarball to a short-lived pre-signed slot, then
// triggers an in-cluster build+deploy of that source and follows the build.
//
// Unlike `okctl deploy` (which builds a committed git revision), `up` ships
// whatever is on disk right now — uncommitted edits included — for fast
// inner-loop iteration.
package up

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	linkstore "github.com/ownkube/okctl/internal/link"
	"github.com/spf13/cobra"
)

// pollInterval is how often the follow loop re-reads build logs and revision
// status. The build itself takes minutes; a 2s cadence keeps output live
// without hammering the API.
const pollInterval = 2 * time.Second

// terminalStatuses are the revision states at which the follow loop stops.
// image_pushed/deploying are intermediate — the loop keeps going until the
// rollout settles at live (or the revision fails / is superseded).
var terminalStatuses = map[string]bool{
	"live":       true,
	"failed":     true,
	"superseded": true,
}

// New builds the `okctl up` command.
func New() *cobra.Command {
	var (
		serviceFlag string
		noteFlag    string
		pathFlag    string
		noFollow    bool
	)

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Build and deploy the current directory",
		Long: "Archive the current directory's working tree, upload it, and build + " +
			"deploy it — uncommitted changes included. The target deployment is taken " +
			"from --service, or inferred from this directory's link (see 'okctl link').\n\n" +
			"Files ignored by .gitignore are never uploaded. The command follows the " +
			"build until it goes live; pass --no-follow to return as soon as the build " +
			"is queued.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

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

			depID, err := resolveDeployment(serviceFlag, dir)
			if err != nil {
				return err
			}

			cl, err := client.New(ux.APIURL(), creds.APIKey, ux.Organization())
			if err != nil {
				return err
			}

			structured := ux.IsStructured()
			out := cmd.OutOrStdout()

			// 1. Mint an upload slot.
			slot, err := cl.PresignSourceUpload(ctx)
			if err != nil {
				return err
			}

			// 2. Archive the working tree to a temp tarball.
			if !structured {
				fmt.Fprintln(out, "Packaging source...")
			}
			archivePath, size, err := archiveWorkingTree(ctx, dir)
			if err != nil {
				return err
			}
			defer os.Remove(archivePath)

			// 3. Upload it straight to object storage via the pre-signed URL.
			if !structured {
				fmt.Fprintf(out, "Uploading source (%s)...\n", humanBytes(size))
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
			result, err := cl.DeployFromUpload(ctx, depID, slot.UploadId, note)
			if err != nil {
				return err
			}

			if structured {
				return ux.Print(out, result)
			}
			fmt.Fprintf(out, "Build queued (revision %s).\n", result.RevisionId)

			if noFollow {
				return nil
			}
			return follow(ctx, cl, out, depID, result.RevisionId)
		},
	}

	cmd.Flags().StringVar(&serviceFlag, "service", "", "Deployment ID to deploy to (defaults to this directory's link)")
	cmd.Flags().StringVar(&noteFlag, "note", "", "Note to record on the revision")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Directory to deploy (defaults to the current directory)")
	cmd.Flags().BoolVar(&noFollow, "no-follow", false, "Return once the build is queued instead of following it")
	return cmd
}

// resolveDeployment returns the target deployment id: the --service flag when
// set, otherwise the deployment bound to dir via `okctl link`.
func resolveDeployment(flag, dir string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	key, err := linkstore.ResolveKey(dir)
	if err != nil {
		return "", err
	}
	mgr := linkstore.NewManager(ux.Config().Dir())
	binding, ok, err := mgr.Get(key)
	if err != nil {
		return "", err
	}
	if !ok || binding.DeploymentID == "" {
		return "", fmt.Errorf(
			"no deployment for this directory — pass --service or run 'okctl link' first")
	}
	return binding.DeploymentID, nil
}

// follow streams build logs and reports status transitions until the revision
// reaches a terminal state. It returns an error when the build fails or is
// superseded so the exit code reflects the outcome.
func follow(ctx context.Context, cl *client.Client, out io.Writer, depID, revisionID string) error {
	printed := 0     // bytes of build log already emitted
	lastStatus := "" // last status we announced

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		// Stream any new build-log output. Errors are transient early on (the
		// build job may not exist yet) — status polling still drives the loop.
		if logs, err := cl.BuildLogs(ctx, depID, revisionID, nil); err == nil && len(logs) > printed {
			fmt.Fprint(out, logs[printed:])
			printed = len(logs)
		}

		status, err := revisionStatus(ctx, cl, depID, revisionID)
		if err != nil {
			return err
		}
		if status != "" && status != lastStatus {
			fmt.Fprintf(out, "\n[%s]\n", statusLabel(status))
			lastStatus = status
		}

		if terminalStatuses[status] {
			switch status {
			case "live":
				fmt.Fprintln(out, "Deployment is live.")
				return nil
			case "failed":
				return fmt.Errorf("build failed for revision %s", revisionID)
			default: // superseded
				return fmt.Errorf("revision %s was superseded before going live", revisionID)
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// revisionStatus finds the given revision among the deployment's revisions and
// returns its status. A missing revision yields "" (keep polling — it may not
// be listed yet).
func revisionStatus(ctx context.Context, cl *client.Client, depID, revisionID string) (string, error) {
	revs, err := cl.ListRevisions(ctx, depID)
	if err != nil {
		return "", err
	}
	for _, r := range revs {
		if r.Id == revisionID {
			return r.Status, nil
		}
	}
	return "", nil
}

// statusLabel renders a revision status for humans (underscores → spaces).
func statusLabel(status string) string {
	switch status {
	case "image_pushed":
		return "image pushed"
	default:
		return status
	}
}

// humanBytes formats a byte count for the upload progress line.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
