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
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
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
				if depID, err = ux.ResolveDeployment(serviceFlag, dir); err != nil {
					return err
				}
			}

			cl, err := client.New(ux.APIURL(), creds.APIKey, ux.Organization())
			if err != nil {
				return err
			}

			structured := ux.IsStructured()
			out := cmd.OutOrStdout()

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
			archivePath, size, err := archiveWorkingTree(ctx, dir)
			if err != nil {
				return err
			}
			defer os.Remove(archivePath)
			if limit := int64(slot.MaxBytes); limit > 0 && size > limit {
				return fmt.Errorf("source is %s; the limit is %s. Add build output, dependencies, and other large files to .gitignore, then retry",
					humanBytes(size), humanBytes(limit))
			}

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
			if creating {
				created, err := createApp(ctx, cmd, cl, &create, dir, slot.UploadId, note)
				if err != nil {
					return err
				}
				if structured {
					return ux.Print(out, created)
				}
				fmt.Fprintf(out, "Build queued (revision %s).\n", created.RevisionId)
				if noFollow {
					return nil
				}
				return follow(ctx, cl, out, created.DeploymentId, created.RevisionId)
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
	create.register(cmd)
	return cmd
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
