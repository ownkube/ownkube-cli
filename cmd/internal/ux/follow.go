package ux

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
)

// followInterval is how often FollowRevision re-reads build logs and revision
// status. The build itself takes minutes; a 2s cadence keeps output live
// without hammering the API.
const followInterval = 2 * time.Second

// terminalStatuses are the revision states at which the follow loop stops.
// image_pushed/deploying are intermediate — the loop keeps going until the
// rollout settles at live (or the revision fails / is superseded).
var terminalStatuses = map[string]bool{
	"live":       true,
	"failed":     true,
	"superseded": true,
}

// IsTerminalRevisionStatus reports whether a revision in this status is done.
func IsTerminalRevisionStatus(status string) bool { return terminalStatuses[status] }

// FollowRevision streams build logs and reports status transitions until the
// revision reaches a terminal state. It returns an error when the build fails
// or is superseded so the exit code reflects the outcome.
func FollowRevision(ctx context.Context, cl *client.Client, out io.Writer, depID, revisionID string) error {
	printed := 0     // bytes of build log already emitted
	lastStatus := "" // last status we announced

	ticker := time.NewTicker(followInterval)
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
			fmt.Fprintf(out, "\n[%s]\n", RevisionStatusLabel(status))
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

// LatestRevision returns the deployment's most recently created revision, or
// nil when it has none. It picks by createdAt rather than trusting list order.
func LatestRevision(revs []api.Revision) *api.Revision {
	var latest *api.Revision
	latestAt := ""
	for i := range revs {
		at := ""
		if revs[i].CreatedAt != nil {
			at = UnionString(revs[i].CreatedAt)
		}
		// ISO-8601 timestamps order lexically; without any, keep the first.
		if latest == nil || at > latestAt {
			latest, latestAt = &revs[i], at
		}
	}
	return latest
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

// RevisionStatusLabel renders a revision status for humans.
func RevisionStatusLabel(status string) string {
	switch status {
	case "image_pushed":
		return "image pushed"
	default:
		return status
	}
}
