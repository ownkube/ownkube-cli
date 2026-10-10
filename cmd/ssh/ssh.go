// Package ssh implements `okctl ssh`: an interactive shell or one-shot command
// on a running Ownkube Compute app over SSH, plus SSH key management and
// ~/.ssh/config generation so stock ssh/scp/sftp work too.
package ssh

import (
	"fmt"
	"os"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/sshconn"
	"github.com/spf13/cobra"
)

// New returns the `okctl ssh` command.
func New() *cobra.Command {
	var (
		serviceFlag  string
		instanceFlag int
		identityFlag string
		yesFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "ssh [deployment-id] [-- command...]",
		Short: "Open a shell on a running app, or run a command in it",
		Long: "Open an interactive shell on a running Ownkube Compute app, or run a single\n" +
			"command after -- and exit with its status.\n\n" +
			"The target is the deployment ID argument, --service, or this directory's link\n" +
			"(see 'okctl link'). The first time you connect, okctl creates an SSH key at\n" +
			"~/.ssh/ownkube_ed25519 (or offers a key you already have) and adds it to your\n" +
			"Ownkube account. Pass --yes to create it without asking.\n\n" +
			"An app with several instances: pick one with --instance N (1-based).\n" +
			"Databases and caches have no shell: use 'okctl connect' for those.",
		Example: "  okctl ssh\n" +
			"  okctl ssh dep_123 --instance 2\n" +
			"  okctl ssh -- ls -la /app\n" +
			"  okctl ssh -- 'cd /app && node scripts/migrate.js'\n" +
			"  okctl ssh config   # then: ssh ownkube-api, scp ownkube-api:/app/x .",
		Args: func(cmd *cobra.Command, args []string) error {
			if n := positional(cmd, args); len(n) > 1 {
				return fmt.Errorf("expected at most one deployment ID before --, got %d", len(n))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pos := positional(cmd, args)
			command := args[len(pos):]

			flag := serviceFlag
			if len(pos) == 1 {
				flag = pos[0]
			}
			depID, err := ux.ResolveDeployment(flag, ".")
			if err != nil {
				return err
			}

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			target, err := sshconn.GetTarget(ctx, api, depID)
			if err != nil {
				return err
			}
			if target.Kind != "app" {
				return fmt.Errorf("%s is a %s, which has no shell. Use 'okctl connect %s' to open a client or a local tunnel",
					target.Name, target.Kind, depID)
			}

			c, err := sshconn.Open(ctx, api, target, sshconn.OpenOptions{
				EnsureOptions: sshconn.EnsureOptions{IdentityFile: identityFlag, Yes: yesFlag, Log: cmd.ErrOrStderr()},
				Instance:      instanceFlag,
			})
			if err != nil {
				return err
			}

			var code int
			if len(command) == 0 {
				code, err = sshconn.Shell(c)
			} else {
				code, err = sshconn.Run(c, sshconn.JoinArgs(command))
			}
			c.Close()
			if err != nil {
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&serviceFlag, "service", "", "Deployment ID (defaults to this directory's link)")
	cmd.Flags().IntVar(&instanceFlag, "instance", 0, "Instance number to connect to, starting at 1 (default: any running instance)")
	cmd.Flags().StringVarP(&identityFlag, "identity-file", "i", "", "SSH private key to use (added to your account if needed)")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Create and add an SSH key without asking when none is set up")

	cmd.AddCommand(keysCmd(), configCmd())
	return cmd
}

// positional returns the args before "--" (all args when there is none).
func positional(cmd *cobra.Command, args []string) []string {
	if at := cmd.ArgsLenAtDash(); at >= 0 {
		return args[:at]
	}
	return args
}
