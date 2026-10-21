// Package gitrepo reads the repository and branch a local checkout tracks, so
// commands that connect an app to a repository can default to "this one".
package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// Info is what a checkout says about itself: the remote host, the repository
// path on it (owner/repo, or group/subgroup/project on GitLab), and the
// checked-out branch ("" on a detached HEAD).
type Info struct {
	Host       string
	Repository string
	Branch     string
}

// ErrNoRemote means dir is a git checkout without an origin remote.
var ErrNoRemote = errors.New("this checkout has no 'origin' remote")

// Detect reads origin's URL and the current branch of the checkout at dir.
// It errors when git is missing, dir isn't a work tree, or origin isn't set.
func Detect(ctx context.Context, dir string) (Info, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return Info{}, fmt.Errorf("git is not installed")
	}
	if _, err := run(ctx, git, dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return Info{}, fmt.Errorf("%s is not a git checkout", dir)
	}
	remote, err := run(ctx, git, dir, "remote", "get-url", "origin")
	if err != nil || remote == "" {
		return Info{}, ErrNoRemote
	}
	host, repo, ok := ParseRemoteURL(remote)
	if !ok {
		return Info{}, fmt.Errorf("can't read a repository from origin %q", remote)
	}
	// symbolic-ref fails on a detached HEAD, which leaves Branch empty.
	branch, _ := run(ctx, git, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	return Info{Host: host, Repository: repo, Branch: branch}, nil
}

// ParseRemoteURL extracts the host and repository path from a git remote URL.
// It accepts the scp-like SSH form (git@host:owner/repo.git) plus ssh://,
// git:// and http(s):// URLs (with or without a port or userinfo), and strips a
// trailing ".git" and slashes. The path keeps every segment, so a GitLab
// group/subgroup/project comes back whole.
func ParseRemoteURL(raw string) (host, repository string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}

	var path string
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return "", "", false
		}
		host, path = u.Hostname(), u.Path
	} else {
		// scp-like: [user@]host:path. A colon must come before any slash,
		// otherwise it's a local path, which has no repository to connect.
		colon := strings.Index(raw, ":")
		if colon <= 0 || strings.Contains(raw[:colon], "/") {
			return "", "", false
		}
		host, path = raw[:colon], raw[colon+1:]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
	}

	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")
	if host == "" || !strings.Contains(path, "/") {
		return "", "", false
	}
	return strings.ToLower(host), path, true
}

// SplitRepository splits owner/repo at the last slash, so a GitLab
// group/subgroup/project yields owner "group/subgroup" and repo "project".
func SplitRepository(repository string) (owner, repo string, ok bool) {
	repository = strings.TrimSuffix(strings.Trim(strings.TrimSpace(repository), "/"), ".git")
	i := strings.LastIndex(repository, "/")
	if i <= 0 || i == len(repository)-1 {
		return "", "", false
	}
	return repository[:i], repository[i+1:], true
}

func run(ctx context.Context, git, dir string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, git, append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}
