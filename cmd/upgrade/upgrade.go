// Package upgrade implements `okctl upgrade`: move to the latest release.
package upgrade

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/update"
	"github.com/ownkube/okctl/internal/version"
	"github.com/spf13/cobra"
)

type status struct {
	Current         string `json:"current" yaml:"current"`
	Latest          string `json:"latest" yaml:"latest"`
	UpdateAvailable bool   `json:"updateAvailable" yaml:"updateAvailable"`
	ReleaseURL      string `json:"releaseUrl" yaml:"releaseUrl"`
	UpgradeCommand  string `json:"upgradeCommand,omitempty" yaml:"upgradeCommand,omitempty"`
}

// New returns the `okctl upgrade` command.
func New() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade okctl to the latest release",
		Long: "Upgrade okctl to the latest GitHub release. A Homebrew install upgrades " +
			"with brew and a go install build prints the go install command; a " +
			"release binary is downloaded, checked against the release checksums, and " +
			"replaced in place. --check only reports whether a newer release exists.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, out := cmd.Context(), cmd.OutOrStdout()
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("can't find the okctl binary: %w", err)
			}
			rel, err := update.Latest(ctx)
			if err != nil {
				return err
			}
			current := version.Version
			st := status{
				Current:         current,
				Latest:          rel.Version(),
				UpdateAvailable: update.Newer(rel.Version(), current),
				ReleaseURL:      rel.URL,
			}
			if st.UpdateAvailable {
				st.UpgradeCommand = update.UpgradeHint(exe)
			}

			if check {
				if ux.IsStructured() {
					return ux.Print(out, st)
				}
				if !update.IsRelease(current) {
					fmt.Fprintf(out, "This okctl (%s) is a development build; the latest release is %s: %s\n",
						current, st.Latest, rel.URL)
					return nil
				}
				if !st.UpdateAvailable {
					fmt.Fprintf(out, "okctl %s is the latest release.\n", current)
					return nil
				}
				fmt.Fprintf(out, "okctl %s is available (you have %s): %s\nUpgrade with: %s\n",
					st.Latest, current, rel.URL, st.UpgradeCommand)
				return nil
			}

			if !update.IsRelease(current) {
				return fmt.Errorf("this okctl (%s) is a development build; install a release from %s", current, rel.URL)
			}
			if !st.UpdateAvailable {
				fmt.Fprintf(out, "okctl %s is already the latest release.\n", current)
				return nil
			}

			switch update.InstallMethod(exe) {
			case update.Homebrew:
				brew, err := exec.LookPath("brew")
				if err != nil {
					return fmt.Errorf("okctl was installed with Homebrew; run: brew upgrade ownkube/tap/okctl")
				}
				fmt.Fprintln(out, "Upgrading with Homebrew...")
				c := exec.CommandContext(ctx, brew, "upgrade", "ownkube/tap/okctl")
				c.Stdout, c.Stderr = out, cmd.ErrOrStderr()
				return c.Run()
			case update.GoInstall:
				fmt.Fprintf(out, "okctl was installed with go install; run:\n  go install github.com/ownkube/okctl@latest\n")
				return nil
			}

			fmt.Fprintf(out, "Downloading okctl %s...\n", st.Latest)
			if err := update.Replace(ctx, rel, exe); err != nil {
				return err
			}
			fmt.Fprintf(out, "Upgraded okctl %s -> %s.\n", current, st.Latest)
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Only report whether a newer release exists")
	return cmd
}
