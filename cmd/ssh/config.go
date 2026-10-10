package ssh

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/sshconn"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	var (
		aliasFlag    string
		identityFlag string
		dryRun       bool
		yesFlag      bool
	)
	cmd := &cobra.Command{
		Use:   "config [deployment-id...]",
		Short: "Add your apps to ~/.ssh/config for ssh, scp and sftp",
		Long: "Write a Host entry per app to ~/.ssh/config so standard tools work without\n" +
			"okctl: 'ssh ownkube-api', 'scp ownkube-api:/app/log.txt .', 'sftp ownkube-api'.\n" +
			"Also records the Ownkube host keys in ~/.ssh/known_hosts. Re-running updates\n" +
			"the entries okctl wrote before.\n\n" +
			"With no deployment IDs, uses this directory's link.",
		Example: "  okctl ssh config\n" +
			"  okctl ssh config dep_123 dep_456\n" +
			"  okctl ssh config dep_123 --alias api --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if aliasFlag != "" && len(args) > 1 {
				return fmt.Errorf("--alias works with a single deployment")
			}
			ids := args
			if len(ids) == 0 {
				id, err := ux.ResolveDeployment("", ".")
				if err != nil {
					return err
				}
				ids = []string{id}
			}

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			// Resolve the key okctl will name in IdentityFile. A dry run never
			// creates or uploads anything.
			identity := identityFlag
			if !dryRun {
				key, err := sshconn.EnsureKey(ctx, api, sshconn.EnsureOptions{
					IdentityFile: identityFlag, Yes: yesFlag, Log: cmd.ErrOrStderr(),
				})
				if err != nil {
					return err
				}
				identity = key.Path
			} else if identity == "" {
				if p, err := sshconn.DefaultKeyPath(); err == nil {
					identity = p
				}
			}

			var entries []sshconn.HostEntry
			for _, id := range ids {
				target, err := sshconn.GetTarget(ctx, api, id)
				if err != nil {
					return fmt.Errorf("%s: %w", id, err)
				}
				if target.Kind != "app" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %s: a %s has no shell. Use 'okctl connect %s' instead.\n",
						target.Name, target.Kind, id)
					continue
				}
				alias := aliasFlag
				if alias == "" {
					alias = "ownkube-" + slug(target.Name)
				}
				entries = append(entries, sshconn.HostEntry{
					Alias:        alias,
					HostName:     target.SSHHost,
					Port:         target.SSHPort,
					User:         sshconn.LoginUser(target),
					IdentityFile: identity,
				})
				if !dryRun {
					if err := sshconn.RememberHostKeys(target.SSHHost, target.SSHPort, target.HostKeys); err != nil {
						return err
					}
				}
			}
			if len(entries) == 0 {
				return nil
			}

			if dryRun {
				fmt.Fprint(cmd.OutOrStdout(), sshconn.RenderHostEntries(entries))
				return nil
			}
			path, err := sshconn.WriteHostEntries(entries)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s:\n", path)
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "  ssh %s\n", e.Alias)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&aliasFlag, "alias", "", "Host alias to write (default: ownkube-<app name>)")
	cmd.Flags().StringVarP(&identityFlag, "identity-file", "i", "", "SSH private key to reference (default: the key okctl uses)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the entries instead of writing them")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Create and add an SSH key without asking when none is set up")
	return cmd
}

// slug lowercases name and replaces anything outside [a-z0-9-] with '-'.
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
