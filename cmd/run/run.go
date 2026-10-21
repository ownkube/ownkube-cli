// Package run implements `okctl run`: a local command, or a local shell, with
// an app's variables in its environment.
package run

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

// New returns the `okctl run` command.
func New() *cobra.Command {
	var serviceFlag, envFlag, projectFlag string

	cmd := &cobra.Command{
		Use:   "run [app] [-- command...]",
		Short: "Run a local command, or a local shell, with an app's variables",
		Long: "Run a command on this machine with the app's variables, secrets included, " +
			"in its environment, and exit with its status. With no command, open your " +
			"$SHELL with them instead; leave it with exit.\n\n" +
			"The values only go into that process's environment: okctl never prints them " +
			"or writes them to disk. They take precedence over variables already set in " +
			"your shell. Each run is recorded in the organization's audit log.\n\n" +
			"The app is a name or id, given before --, with --service, or taken from this " +
			"directory's link ('okctl link'). To get a shell on the running app itself, " +
			"use 'okctl ssh' (or 'okctl shell').",
		Example: "  okctl run -- npm run dev\n" +
			"  okctl run api -- ./scripts/migrate.sh\n" +
			"  okctl run -s worker -e staging -- python manage.py shell\n" +
			"  okctl run",
		Args: func(cmd *cobra.Command, args []string) error {
			pos := positional(cmd, args)
			if cmd.ArgsLenAtDash() < 0 && len(pos) > 1 {
				return fmt.Errorf("put the command after --: okctl run [app] -- %s", strings.Join(args, " "))
			}
			if len(pos) > 1 {
				return fmt.Errorf("expected at most one app before --, got %d", len(pos))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pos := positional(cmd, args)
			command := args[len(pos):]
			ref := serviceFlag
			if len(pos) == 1 {
				if ref != "" {
					return fmt.Errorf("pass the app as an argument or with --service, not both")
				}
				ref = pos[0]
			}

			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			scope, err := ux.ResolveScope(ctx, cl, projectFlag, envFlag)
			if err != nil {
				return err
			}
			depID, err := ux.ResolveApp(ctx, cl, ref, scope, ".")
			if err != nil {
				return err
			}

			purpose := api.Run
			if len(command) == 0 {
				purpose = api.Shell
			}
			res, err := cl.ResolveVariables(ctx, depID, purpose)
			if err != nil {
				return err
			}
			env := environ(res)

			var c *exec.Cmd
			if len(command) == 0 {
				sh := shell()
				fmt.Fprintf(cmd.ErrOrStderr(), "Starting %s with %d variable(s) from %s. Type exit to leave.\n", sh, len(res.Variables), depID)
				c = exec.Command(sh)
			} else {
				path, err := exec.LookPath(command[0])
				if err != nil {
					return err
				}
				c = exec.Command(path, command[1:]...)
			}
			c.Env = env
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr

			// Ctrl-C reaches the child through the terminal; okctl stays up to
			// report the child's exit status.
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt)
			defer signal.Stop(sig)

			if err := c.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					code := exitErr.ExitCode()
					if code < 0 {
						code = 1
					}
					return &ux.ExitError{Code: code}
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&serviceFlag, "service", "s", "", "App name or id (default: this directory's link)")
	cmd.Flags().StringVarP(&envFlag, "environment", "e", "", "Environment name or id, to narrow an app name")
	cmd.Flags().StringVarP(&projectFlag, "project", "p", "", "Project name or id, to narrow an app or environment name")
	return cmd
}

func positional(cmd *cobra.Command, args []string) []string {
	if at := cmd.ArgsLenAtDash(); at >= 0 {
		return args[:at]
	}
	return args
}

// environ is this process's environment with the app's variables laid over it.
func environ(res *api.ResolvedAppVariables) []string {
	vars := map[string]string{}
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			vars[k] = v
		}
	}
	for _, v := range res.Variables {
		vars[v.Name] = v.Value
	}
	names := make([]string, 0, len(vars))
	for k := range vars {
		names = append(names, k)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, k := range names {
		out = append(out, k+"="+vars[k])
	}
	return out
}

// shell is the user's login shell.
func shell() string {
	if runtime.GOOS == "windows" {
		if s := os.Getenv("COMSPEC"); s != "" {
			return s
		}
		return "cmd.exe"
	}
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}
