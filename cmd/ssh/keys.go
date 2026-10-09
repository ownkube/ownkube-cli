package ssh

import (
	"fmt"
	"os"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/ownkube/okctl/internal/sshconn"
	"github.com/spf13/cobra"
)

func keysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage the SSH keys on your Ownkube account",
	}
	cmd.AddCommand(keysListCmd(), keysAddCmd(), keysRemoveCmd(), keysGitHubCmd())
	return cmd
}

func keysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your SSH keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			keys, err := api.ListSSHKeys(cmd.Context())
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), keys)
			}
			if len(keys) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No SSH keys yet. 'okctl ssh' creates one the first time you connect, or run 'okctl ssh keys add'.")
				return nil
			}
			rows := [][]string{{"ID", "NAME", "FINGERPRINT", "SOURCE", "ADDED", "LAST USED"}}
			for _, k := range keys {
				lastUsed := ux.Deref(k.LastUsedAt)
				if lastUsed == "" {
					lastUsed = "never"
				}
				rows = append(rows, []string{k.ID, k.Name, k.Fingerprint, k.Source, k.CreatedAt, lastUsed})
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

func keysAddCmd() *cobra.Command {
	var nameFlag string
	cmd := &cobra.Command{
		Use:   "add [key-file]",
		Short: "Add an SSH public key to your account",
		Long: "Add an SSH public key to your Ownkube account. key-file is a private key or its\n" +
			".pub. With no key-file, okctl uses ~/.ssh/ownkube_ed25519, creating it if it\n" +
			"does not exist yet.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var key *sshconn.LocalKey
			if len(args) == 1 {
				key, err = sshconn.LoadKey(args[0])
			} else {
				key, err = defaultKey(cmd)
			}
			if err != nil {
				return err
			}

			if err := sshconn.Register(cmd.Context(), api, key, nameFlag, cmd.ErrOrStderr()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s is on your account (%s)\n", key.Label(), key.Fingerprint)
			return nil
		},
	}
	cmd.Flags().StringVar(&nameFlag, "name", "", "Name for the key (default: okctl@<this machine>)")
	return cmd
}

// defaultKey loads ~/.ssh/ownkube_ed25519, creating it when missing.
func defaultKey(cmd *cobra.Command) (*sshconn.LocalKey, error) {
	path, err := sshconn.DefaultKeyPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		return sshconn.LoadKey(path)
	}
	key, err := sshconn.GenerateKey(path, sshconn.DefaultKeyLabel())
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Created a new SSH key at %s\n", key.Label())
	return key, nil
}

func keysRemoveCmd() *cobra.Command {
	var yesFlag bool
	cmd := &cobra.Command{
		Use:     "remove <key-id>",
		Aliases: []string{"rm"},
		Short:   "Remove an SSH key from your account",
		Long: "Remove an SSH key from your account. Open sessions that used it keep running\n" +
			"until they end; new connections with it are refused.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			if !yesFlag {
				ok, err := prompt.Confirm(fmt.Sprintf("Remove SSH key %s?", args[0]))
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			if err := api.RemoveSSHKey(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed SSH key %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

func keysGitHubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "github [username]",
		Short: "Import your public SSH keys from GitHub",
		Long: "Import the public SSH keys published on a GitHub profile. With no username,\n" +
			"okctl uses the GitHub account connected to your Ownkube account.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}
			username := ""
			if len(args) == 1 {
				username = args[0]
			}
			keys, err := api.ImportGitHubSSHKeys(cmd.Context(), username)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), keys)
			}
			if len(keys) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "That GitHub profile has no public SSH keys to import.")
				return nil
			}
			for _, k := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "Imported %s (%s)\n", k.Name, k.Fingerprint)
			}
			return nil
		},
	}
}
