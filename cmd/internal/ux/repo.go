package ux

import (
	"context"
	"fmt"
	"strings"

	"github.com/ownkube/okctl/internal/gitrepo"
)

// ResolveRepoBranch fills the repository and branch to connect from flags,
// falling back to the origin remote and current branch of the checkout at dir.
func ResolveRepoBranch(ctx context.Context, dir, repo, branch string) (string, string, error) {
	if repo != "" && branch != "" {
		return repo, branch, nil
	}
	info, detectErr := gitrepo.Detect(ctx, dir)

	if repo == "" {
		if detectErr != nil {
			return "", "", fmt.Errorf("%v; pass --repo owner/repo", detectErr)
		}
		repo = info.Repository
	} else if detectErr != nil || !strings.EqualFold(info.Repository, repo) {
		// The current branch only means something for this checkout's repo.
		return "", "", fmt.Errorf("pass --branch for %s", repo)
	}
	if branch == "" {
		if info.Branch == "" {
			return "", "", fmt.Errorf("HEAD is detached; pass --branch")
		}
		branch = info.Branch
	}
	return repo, branch, nil
}
