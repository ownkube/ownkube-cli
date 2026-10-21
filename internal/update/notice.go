package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/term"
)

// checkEvery is how often the passive notice asks GitHub for a new release.
const checkEvery = 24 * time.Hour

// cacheFile holds the last check, next to config.yaml.
const cacheFile = "update-check.json"

type cache struct {
	CheckedAt time.Time `json:"checkedAt"`
	Latest    string    `json:"latest"`
}

// ciEnv are variables CI systems set; the notice stays quiet under any of them.
var ciEnv = []string{"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "RUN_ID", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE", "CIRCLECI", "TF_BUILD"}

// NoticeEnabled reports whether the passive update notice may run: a release
// build, stderr on a terminal, not in CI, and not turned off with
// OKCTL_NO_UPDATE_NOTIFIER.
func NoticeEnabled(current string) bool {
	if !IsRelease(current) {
		return false
	}
	if os.Getenv("OKCTL_NO_UPDATE_NOTIFIER") != "" {
		return false
	}
	for _, k := range ciEnv {
		if os.Getenv(k) != "" {
			return false
		}
	}
	return term.IsTerminal(int(os.Stderr.Fd()))
}

// Check returns a channel that yields the latest version when it is newer than
// current, or "" otherwise. It answers from the cache in dir when the last
// check is under a day old, and otherwise asks GitHub in the background and
// refreshes the cache. The channel always receives exactly one value.
func Check(dir, current string) <-chan string {
	ch := make(chan string, 1)
	path := filepath.Join(dir, cacheFile)
	var c cache
	if raw, err := os.ReadFile(path); err == nil && json.Unmarshal(raw, &c) == nil && time.Since(c.CheckedAt) < checkEvery {
		ch <- newerOrEmpty(c.Latest, current)
		return ch
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		r, err := Latest(ctx)
		if err != nil {
			ch <- ""
			return
		}
		if raw, err := json.Marshal(cache{CheckedAt: time.Now(), Latest: r.Version()}); err == nil {
			_ = os.WriteFile(path, raw, 0o600)
		}
		ch <- newerOrEmpty(r.Version(), current)
	}()
	return ch
}

func newerOrEmpty(latest, current string) string {
	if Newer(latest, current) {
		return latest
	}
	return ""
}

// UpgradeHint is the command that upgrades the binary at exe.
func UpgradeHint(exe string) string {
	switch InstallMethod(exe) {
	case Homebrew:
		return "brew upgrade ownkube/tap/okctl"
	case GoInstall:
		return "go install github.com/ownkube/okctl@latest"
	default:
		return "okctl upgrade"
	}
}
