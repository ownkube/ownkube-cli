package ux

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

// appArg is the placeholder a command's Use puts first to take an app.
const appArg = "[app]"

const appHelp = "The app is a name or id, given as the argument, with --service, or taken " +
	"from this directory's link ('okctl link'). --environment and --project narrow a " +
	"name that more than one app shares."

// WireAppTargets walks root and wires every command whose first positional in
// Use is "[app]": the app may be a name or an id, given as that argument, with
// --service/-s, or taken from this directory's link, and --environment/-e and
// --project/-p narrow a name lookup. The command's RunE then sees the resolved
// deployment id as args[0], so it is written as if it always took an id.
// A command that defines its own --service (ssh, connect) resolves its own
// target with ResolveApp and is left alone.
func WireAppTargets(root *cobra.Command) {
	for _, c := range root.Commands() {
		WireAppTargets(c)
	}
	f := strings.Fields(root.Use)
	if len(f) > 1 && f[1] == appArg && root.RunE != nil && root.Flags().Lookup("service") == nil {
		wireApp(root)
	}
}

func wireApp(c *cobra.Command) {
	orig := c.Args
	if orig == nil {
		orig = cobra.ArbitraryArgs
	}
	addRefFlag(c, "service", "s", "App name or id (default: this directory's link)")
	hasEnv := addRefFlag(c, "environment", "e", "Environment name or id, to narrow an app name")
	hasProject := addRefFlag(c, "project", "p", "Project name or id, to narrow an app or environment name")

	if c.Long == "" {
		c.Long = c.Short + "."
	}
	c.Long += "\n\n" + appHelp

	// The app argument may be left out; the command's own validator decides
	// whether args already include it.
	c.Args = func(cmd *cobra.Command, args []string) error {
		if orig(cmd, args) == nil || orig(cmd, append([]string{""}, args...)) == nil {
			return nil
		}
		return orig(cmd, args)
	}

	run := c.RunE
	c.RunE = func(cmd *cobra.Command, args []string) error {
		service, _ := cmd.Flags().GetString("service")
		ref, rest := "", args
		switch {
		case service != "":
			// --service names the app, so every argument is the command's own.
			if orig(cmd, append([]string{service}, args...)) != nil {
				return fmt.Errorf("pass the app as an argument or with --service, not both")
			}
			ref = service
		case orig(cmd, args) == nil && len(args) > 0:
			ref, rest = args[0], args[1:]
		}

		cl, err := RequireClient()
		if err != nil {
			return err
		}
		var envRef, projectRef string
		if hasEnv {
			envRef, _ = cmd.Flags().GetString("environment")
		}
		if hasProject {
			projectRef, _ = cmd.Flags().GetString("project")
		}
		scope, err := ResolveScope(cmd.Context(), cl, projectRef, envRef)
		if err != nil {
			return err
		}
		id, err := ResolveApp(cmd.Context(), cl, ref, scope, cwd())
		if err != nil {
			return err
		}
		return run(cmd, append([]string{id}, rest...))
	}
}

// addRefFlag adds a string flag unless the command already defines one by that
// name, dropping the shorthand when it's taken. It reports whether it added it.
func addRefFlag(c *cobra.Command, name, short, usage string) bool {
	if c.Flags().Lookup(name) != nil {
		return false
	}
	if c.Flags().ShorthandLookup(short) != nil {
		short = ""
	}
	c.Flags().StringP(name, short, "", usage)
	return true
}

// ResolveScope resolves --project and --environment refs (names or ids) into a
// Scope. An environment name given without --project is looked up in the
// linked environment's project first, since every project has its own
// "Production".
func ResolveScope(ctx context.Context, cl *client.Client, projectRef, envRef string) (Scope, error) {
	projectID, err := ResolveProject(ctx, cl, projectRef)
	if err != nil {
		return Scope{}, err
	}
	if envRef == "" {
		return Scope{ProjectID: projectID}, nil
	}
	if projectID == "" && !LooksLikeID(envRef) {
		if linked := linkedProject(ctx, cl); linked != "" {
			if id, err := ResolveEnvironment(ctx, cl, envRef, linked); err == nil {
				return Scope{ProjectID: linked, EnvironmentID: id}, nil
			}
		}
	}
	envID, err := ResolveEnvironment(ctx, cl, envRef, projectID)
	if err != nil {
		return Scope{}, err
	}
	return Scope{ProjectID: projectID, EnvironmentID: envID}, nil
}

// linkedProject is the project of this directory's linked environment, or "".
func linkedProject(ctx context.Context, cl *client.Client) string {
	b, ok, err := Linked(cwd())
	if err != nil || !ok || b.EnvironmentID == "" {
		return ""
	}
	env, err := cl.GetEnvironment(ctx, b.EnvironmentID)
	if err != nil {
		return ""
	}
	return Deref(env.ProjectId)
}

// cwd returns the working directory, or "." when it can't be read.
func cwd() string {
	if d, err := os.Getwd(); err == nil {
		return d
	}
	return "."
}
