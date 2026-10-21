package deploy

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

// formatLogTimestamp turns a Unix-nanoseconds string into an RFC3339 stamp,
// falling back to the raw value if it can't be parsed.
func formatLogTimestamp(ns string) string {
	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return ns
	}
	return time.Unix(0, n).UTC().Format(time.RFC3339)
}

func logsCmd() *cobra.Command {
	var (
		build, latest, follow    bool
		revisionID, since, until string
	)
	c := &cobra.Command{
		Use:   "logs [app]",
		Short: "Fetch logs for a deployment",
		Long: "Fetch an app's runtime logs, or with --build a revision's build log: " +
			"the live revision's by default, --latest for the newest one (even one " +
			"that failed or hasn't gone live), or --revision.\n\n" +
			"Runtime logs cover the last hour by default. --range looks back a number " +
			"of seconds; --since and --until pick a window, each a duration back from " +
			"now (30m, 2h, 7d), a date or time (2026-10-21, 2026-10-21T09:30, RFC 3339), " +
			"or unix seconds. A window can span at most 90 days.\n\n" +
			"--follow (-f) keeps printing new runtime lines until interrupted, or the " +
			"build log until the revision is live or fails.",
		Example: "  okctl deploy logs api -f\n" +
			"  okctl deploy logs api --since 2h --filter error\n" +
			"  okctl deploy logs api --since 2026-10-20 --until 2026-10-21\n" +
			"  okctl deploy logs api --build --latest -f",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			if latest || revisionID != "" {
				build = true
			}
			if latest && revisionID != "" {
				return fmt.Errorf("pass --latest or --revision, not both")
			}
			if build {
				if since != "" || until != "" {
					return fmt.Errorf("--since and --until apply to runtime logs, not --build")
				}
				return buildLog(cmd, api, args[0], revisionID, latest, follow)
			}

			rangeSec, _ := cmd.Flags().GetInt("range")
			limit, _ := cmd.Flags().GetInt("limit")
			filter, _ := cmd.Flags().GetString("filter")
			opts := client.LogsOptions{RangeSeconds: rangeSec, Limit: limit, Filter: filter}
			if err := logWindow(&opts, since, until, follow, time.Now()); err != nil {
				return err
			}
			if follow {
				return followLogs(cmd, api, args[0], opts)
			}

			entries, err := api.GetDeploymentLogs(cmd.Context(), args[0], opts)
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), entries)
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No log entries.")
				return nil
			}
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", formatLogTimestamp(e.TimestampNs), e.Message)
			}
			return nil
		},
	}
	c.Flags().Int("range", 0, "Look back this many seconds")
	c.Flags().StringVar(&since, "since", "", "Start of the window: a duration back (2h), a date or time, or unix seconds")
	c.Flags().StringVar(&until, "until", "", "End of the window, in the same forms as --since")
	c.Flags().Int("limit", 0, "Maximum number of log entries to return")
	c.Flags().String("filter", "", "Filter log entries by substring")
	c.Flags().BoolVar(&build, "build", false, "Show a revision's build log instead of runtime logs")
	c.Flags().BoolVar(&latest, "latest", false, "Build log of the newest revision, live or not (implies --build)")
	c.Flags().StringVar(&revisionID, "revision", "", "Build log of this revision (implies --build)")
	c.Flags().BoolVarP(&follow, "follow", "f", false, "Keep printing new lines (with --build: until the revision is live or fails)")
	return c
}

// buildLog prints (or follows) one revision's build log.
func buildLog(cmd *cobra.Command, cl *client.Client, depID, revisionID string, latest, follow bool) error {
	ctx, out := cmd.Context(), cmd.OutOrStdout()
	if revisionID == "" {
		revs, err := cl.ListRevisions(ctx, depID)
		if err != nil {
			return err
		}
		rev := liveRevision(revs)
		if latest || rev == nil {
			rev = ux.LatestRevision(revs)
		}
		if rev == nil {
			return fmt.Errorf("this deployment has no revisions yet")
		}
		revisionID = rev.Id
	}

	if follow {
		if !ux.IsStructured() {
			return ux.FollowRevision(ctx, cl, out, depID, revisionID)
		}
		// Structured output gets the finished log once the revision settles.
		if err := ux.FollowRevision(ctx, cl, io.Discard, depID, revisionID); err != nil {
			logs, _ := cl.BuildLogs(ctx, depID, revisionID, nil)
			_ = ux.Print(out, map[string]string{"revisionId": revisionID, "logs": logs})
			return err
		}
	}

	logs, err := cl.BuildLogs(ctx, depID, revisionID, nil)
	if err != nil {
		return err
	}
	if ux.IsStructured() {
		return ux.Print(out, map[string]string{"revisionId": revisionID, "logs": logs})
	}
	if logs == "" {
		fmt.Fprintf(out, "No build log for revision %s yet.\n", revisionID)
		return nil
	}
	fmt.Fprint(out, logs)
	return nil
}

// maxLogWindow is the longest window one logs read can cover.
const maxLogWindow = 90 * 24 * time.Hour

// logWindow turns --since/--until into opts' Start/End.
func logWindow(opts *client.LogsOptions, since, until string, follow bool, now time.Time) error {
	if since == "" && until == "" {
		return nil
	}
	if opts.RangeSeconds > 0 {
		return fmt.Errorf("pass --range or --since/--until, not both")
	}
	if follow && until != "" {
		return fmt.Errorf("--follow keeps reading new lines, so it can't take --until")
	}
	end := now
	if until != "" {
		t, err := parseWhen(until, now)
		if err != nil {
			return fmt.Errorf("--until: %w", err)
		}
		end = t
	}
	start := end.Add(-time.Hour)
	if since != "" {
		t, err := parseWhen(since, now)
		if err != nil {
			return fmt.Errorf("--since: %w", err)
		}
		start = t
	}
	if !end.After(start) {
		return fmt.Errorf("--until must be after --since")
	}
	if end.Sub(start) > maxLogWindow {
		return fmt.Errorf("a logs window can span at most 90 days")
	}
	opts.Start, opts.End = start.Unix(), end.Unix()
	if opts.End == opts.Start {
		opts.End++
	}
	return nil
}

// parseWhen reads a point in time: a duration back from now (90s, 30m, 2h,
// 7d), unix seconds, an RFC 3339 time, or a local date or date and time.
func parseWhen(s string, now time.Time) (time.Time, error) {
	if days, ok := strings.CutSuffix(s, "d"); ok {
		if n, err := strconv.Atoi(days); err == nil && n >= 0 {
			return now.Add(-time.Duration(n) * 24 * time.Hour), nil
		}
	}
	if d, err := time.ParseDuration(s); err == nil {
		if d < 0 {
			d = -d
		}
		return now.Add(-d), nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%q isn't a duration (2h, 7d), a date or time (2026-10-21T09:30), or unix seconds", s)
}

// followPoll is how often --follow asks for new runtime lines.
const followPoll = 2 * time.Second

// followLogs prints the window opts selects, then keeps polling for newer
// lines until interrupted. Structured output is one JSON object per line.
func followLogs(cmd *cobra.Command, cl *client.Client, depID string, opts client.LogsOptions) error {
	if ux.OutputFormat() == "yaml" {
		return fmt.Errorf("--follow prints one JSON object per line; use -o json")
	}
	ctx, out := cmd.Context(), cmd.OutOrStdout()
	if opts.Start == 0 {
		lookBack := time.Hour
		if opts.RangeSeconds > 0 {
			lookBack = time.Duration(opts.RangeSeconds) * time.Second
		}
		now := time.Now()
		opts.Start, opts.End = now.Add(-lookBack).Unix(), now.Unix()+1
	}
	opts.RangeSeconds = 0

	// A line is told apart from one already printed by its timestamp and text;
	// polls overlap by the last second so nothing on its boundary is lost.
	seen := map[string]int64{}
	enc := json.NewEncoder(out)
	for {
		entries, err := cl.GetDeploymentLogs(ctx, depID, opts)
		if err != nil {
			return err
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return logNs(entries[i].TimestampNs) < logNs(entries[j].TimestampNs)
		})
		latest := opts.Start
		for _, e := range entries {
			ns := logNs(e.TimestampNs)
			key := e.TimestampNs + "\x00" + e.Message
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = ns
			if ux.IsStructured() {
				if err := enc.Encode(e); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(out, "%s  %s\n", formatLogTimestamp(e.TimestampNs), e.Message)
			}
			if s := ns / int64(time.Second); s > latest {
				latest = s
			}
		}
		for k, ns := range seen {
			if ns/int64(time.Second) < latest {
				delete(seen, k)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(followPoll):
		}
		opts.Start, opts.End = latest, time.Now().Unix()+1
	}
}

func logNs(ns string) int64 {
	n, _ := strconv.ParseInt(ns, 10, 64)
	return n
}
