package vars

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func setCmd() *cobra.Command {
	var secret, stdin bool
	cmd := &cobra.Command{
		Use:   "set [app] NAME=VALUE...",
		Short: "Set one or more variables on an app",
		Long: "Set variables on the app in one change. A NAME given without =VALUE is " +
			"prompted for with hidden input, so the value stays out of your shell " +
			"history; --stdin reads it from stdin instead (one NAME only). --secret " +
			"stores every value as a secret; without it, an existing variable keeps " +
			"its secret or plain flag and a new one is plain.\n\n" +
			"With more than one bare NAME, give the app too (or --service): the first " +
			"argument is otherwise taken as the app.",
		Example: "  okctl vars set LOG_LEVEL=debug\n" +
			"  okctl vars set api LOG_LEVEL=debug FEATURE_X=on\n" +
			"  okctl vars set STRIPE_KEY --secret\n" +
			"  printf %s \"$TOKEN\" | okctl vars set API_TOKEN --secret --stdin",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 || strings.Contains(args[0], "=") {
				return fmt.Errorf("expected NAME=VALUE (or a bare NAME to be prompted for)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			depID, entries := args[0], args[1:]
			if stdin && (len(entries) != 1 || strings.Contains(entries[0], "=")) {
				return fmt.Errorf("--stdin reads the value of one NAME: okctl vars set NAME --stdin")
			}

			var set []api.AppVariableInput
			seen := map[string]bool{}
			for _, e := range entries {
				name, value, hasValue := strings.Cut(e, "=")
				if err := checkName(name); err != nil {
					return err
				}
				if seen[name] {
					return fmt.Errorf("%s is given twice", name)
				}
				seen[name] = true
				if !hasValue {
					v, err := readValue(name, stdin)
					if err != nil {
						return err
					}
					value = v
				}
				in := api.AppVariableInput{Name: name, Value: value}
				if secret {
					if value == "" {
						return fmt.Errorf("%s is empty; a secret needs a value", name)
					}
					t := true
					in.Secret = &t
				}
				set = append(set, in)
			}

			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			return update(cmd, cl, depID, api.UpdateAppVariablesBody{Set: &set})
		},
	}
	cmd.Flags().BoolVar(&secret, "secret", false, "Store the values as secrets")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read the value of the one NAME from stdin")
	return cmd
}

// readValue reads name's value from stdin, or prompts for it with hidden input.
func readValue(name string, fromStdin bool) (string, error) {
	if fromStdin {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		v := strings.TrimSuffix(string(raw), "\n")
		return strings.TrimSuffix(v, "\r"), nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("%s has no value: pass %s=VALUE, or --stdin to read it from stdin", name, name)
	}
	return prompt.ReadSecret(fmt.Sprintf("Value for %s: ", name))
}
