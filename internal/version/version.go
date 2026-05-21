package version

import (
	"fmt"
	"runtime"
)

// Set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Full() string {
	return fmt.Sprintf("okctl %s (%s) built %s %s/%s", Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}
