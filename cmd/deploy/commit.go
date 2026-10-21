package deploy

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// commitPattern matches an abbreviated or full commit SHA, as the API accepts.
var commitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// commitFlag returns --commit when set, rejecting anything that isn't a 7-40
// character hex SHA before it reaches the API.
func commitFlag(cmd *cobra.Command) (*string, error) {
	if !cmd.Flags().Changed("commit") {
		return nil, nil
	}
	v, _ := cmd.Flags().GetString("commit")
	v = strings.TrimSpace(v)
	if !commitPattern.MatchString(v) {
		return nil, fmt.Errorf("--commit must be a commit SHA of 7 to 40 hex characters, got %q", v)
	}
	return &v, nil
}

// shortSha trims a commit SHA to the usual 7 characters for display.
func shortSha(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// printWatchHint tells the user how to follow a queued build.
func printWatchHint(w io.Writer, deploymentID string) {
	fmt.Fprintf(w, "Follow it with: okctl deploy watch %s\n", deploymentID)
}

// revisionSha is the short commit a revision was built from. A repository
// build's tag is its commit; any other tag (an uploaded-source build, an image
// tag) isn't a commit and yields "".
func revisionSha(r api.Revision) string {
	if !commitPattern.MatchString(r.Tag) {
		return ""
	}
	return shortSha(r.Tag)
}

// revisionCommit renders a revision's commit for one line of output, e.g.
// "1a2b3c4 Fix login redirect (Jane Doe)", or "" when it has no commit.
func revisionCommit(r api.Revision) string {
	parts := make([]string, 0, 3)
	if sha := revisionSha(r); sha != "" {
		parts = append(parts, sha)
	}
	if msg := ux.Deref(r.CommitMessage); msg != "" {
		parts = append(parts, msg)
	}
	if author := ux.Deref(r.CommitAuthor); author != "" {
		parts = append(parts, "("+author+")")
	}
	return strings.Join(parts, " ")
}

// truncateText shortens s to at most n runes for a table cell.
func truncateText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
