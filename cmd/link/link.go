// Package link implements `okctl link` / `okctl unlink`: binding the current
// directory to specific Ownkube resources so later commands (up, logs, connect)
// can infer their target instead of taking ids on the flag line. The binding
// lives in global config (internal/link), never in the repo; the committed
// deploy spec is ownkube.yaml (internal/manifest).
package link

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	linkstore "github.com/ownkube/okctl/internal/link"
	"github.com/ownkube/okctl/internal/manifest"
	"github.com/ownkube/okctl/internal/prompt"
	"github.com/spf13/cobra"
)

// Link builds the `okctl link` command.
func Link() *cobra.Command {
	var (
		orgFlag     string
		clusterFlag string
		envFlag     string
		serviceFlag string
		initSpec    bool
	)

	cmd := &cobra.Command{
		Use:   "link",
		Short: "Link this directory to an Ownkube deployment",
		Long: "Bind the current directory to an organization, cluster, environment, " +
			"and deployment so later commands can infer their target. The binding is " +
			"stored in your global config, never in the repository.\n\n" +
			"Run without flags for an interactive picker, or pass --organization / " +
			"--cluster / --environment / --service to link non-interactively (e.g. in CI).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			creds, err := ux.RequireAuth()
			if err != nil {
				return err
			}

			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			key, err := linkstore.ResolveKey(dir)
			if err != nil {
				return err
			}

			structured := ux.IsStructured()

			// Organization first — it scopes every other list. Resolve from the
			// flag, the global default, or an interactive pick.
			orgID, err := resolveOrg(ctx, orgFlag, structured)
			if err != nil {
				return err
			}

			// A client scoped to the chosen org for the remaining lists.
			c, err := client.New(ux.APIURL(), creds.APIKey, orgID)
			if err != nil {
				return err
			}

			clusterID, err := resolveCluster(ctx, c, clusterFlag, structured)
			if err != nil {
				return err
			}
			envID, err := resolveEnvironment(ctx, c, envFlag, structured)
			if err != nil {
				return err
			}
			depID, err := resolveDeployment(ctx, c, serviceFlag, clusterID, envID, structured)
			if err != nil {
				return err
			}

			binding := linkstore.Binding{
				OrganizationID: orgID,
				ClusterID:      clusterID,
				EnvironmentID:  envID,
				DeploymentID:   depID,
			}
			mgr := linkstore.NewManager(ux.Config().Dir())
			if err := mgr.Set(key, binding); err != nil {
				return err
			}

			if initSpec {
				if err := scaffoldManifest(ctx, c, dir, depID); err != nil {
					// Scaffolding is best-effort — the link still succeeded.
					fmt.Fprintf(os.Stderr, "Note: could not write %s: %v\n", manifest.FileName, err)
				}
			}

			if structured {
				return ux.Print(cmd.OutOrStdout(), binding)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Linked %s\n", key)
			printBinding(cmd, binding)
			if !manifest.Exists(dir) {
				fmt.Fprintf(cmd.OutOrStdout(),
					"\nNo %s found. Run 'okctl link --init' to scaffold a deploy spec.\n",
					manifest.FileName)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&orgFlag, "organization", "", "Organization ID (skips the org picker)")
	cmd.Flags().StringVar(&clusterFlag, "cluster", "", "Cluster ID (skips the cluster picker)")
	cmd.Flags().StringVar(&envFlag, "environment", "", "Environment ID (skips the environment picker)")
	cmd.Flags().StringVar(&serviceFlag, "service", "", "Deployment ID to target (skips the deployment picker)")
	cmd.Flags().BoolVar(&initSpec, "init", false, fmt.Sprintf("Scaffold an %s deploy spec if one does not exist", manifest.FileName))
	return cmd
}

// Unlink builds the `okctl unlink` command.
func Unlink() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove this directory's link",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			key, err := linkstore.ResolveKey(dir)
			if err != nil {
				return err
			}
			mgr := linkstore.NewManager(ux.Config().Dir())
			_, ok, err := mgr.Get(key)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "This directory is not linked.")
				return nil
			}
			if !yes && !ux.IsStructured() {
				confirmed, err := prompt.Confirm(fmt.Sprintf("Remove the link for %s?", key))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}
			if _, err := mgr.Remove(key); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Unlinked.")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

// resolveOrg returns the org id from the flag, the global default, or a picker.
func resolveOrg(ctx context.Context, flag string, structured bool) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if o := ux.Organization(); o != "" {
		return o, nil
	}
	c, err := ux.RequireClient()
	if err != nil {
		return "", err
	}
	orgs, err := c.ListOrganizations(ctx)
	if err != nil {
		return "", err
	}
	if len(orgs) == 0 {
		return "", fmt.Errorf("no organizations found for this account")
	}
	labels := make([]string, len(orgs))
	ids := make([]string, len(orgs))
	for i, o := range orgs {
		labels[i] = fmt.Sprintf("%s (%s)", o.Name, o.Slug)
		ids[i] = o.Id
	}
	return choose("organization", ids, labels, "", true, structured)
}

func resolveCluster(ctx context.Context, c *client.Client, flag string, structured bool) (string, error) {
	if flag != "" {
		return flag, nil
	}
	clusters, err := c.ListClusters(ctx)
	if err != nil {
		return "", err
	}
	labels := make([]string, len(clusters))
	ids := make([]string, len(clusters))
	for i, cl := range clusters {
		region := ux.Deref(cl.Region)
		if region != "" {
			labels[i] = fmt.Sprintf("%s (%s, %s)", cl.Name, cl.ClusterType, region)
		} else {
			labels[i] = fmt.Sprintf("%s (%s)", cl.Name, cl.ClusterType)
		}
		ids[i] = cl.Id
	}
	// A cluster is optional — Ownkube Compute orgs have none until first deploy.
	return choose("cluster", ids, labels, "", false, structured)
}

func resolveEnvironment(ctx context.Context, c *client.Client, flag string, structured bool) (string, error) {
	if flag != "" {
		return flag, nil
	}
	envs, err := c.ListEnvironments(ctx)
	if err != nil {
		return "", err
	}
	labels := make([]string, len(envs))
	ids := make([]string, len(envs))
	for i, e := range envs {
		labels[i] = fmt.Sprintf("%s (%s)", e.Name, e.Slug)
		ids[i] = e.Id
	}
	return choose("environment", ids, labels, "", false, structured)
}

func resolveDeployment(ctx context.Context, c *client.Client, flag, clusterID, envID string, structured bool) (string, error) {
	if flag != "" {
		return flag, nil
	}
	deps, err := c.ListDeployments(ctx, client.ListDeploymentsFilter{ClusterID: clusterID, EnvironmentID: envID})
	if err != nil {
		return "", err
	}
	labels := make([]string, len(deps))
	ids := make([]string, len(deps))
	for i, d := range deps {
		labels[i] = fmt.Sprintf("%s (%s)", d.Name, d.ResourceType)
		ids[i] = d.Id
	}
	return choose("deployment", ids, labels, "", false, structured)
}

// choose returns flagVal when set; otherwise picks from ids/labels. An empty
// list yields "" (or an error when required). A single option is chosen
// automatically. In structured/non-interactive mode a needed choice errors,
// pointing the caller at the flag.
func choose(kind string, ids, labels []string, flagVal string, required, structured bool) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	switch len(ids) {
	case 0:
		if required {
			return "", fmt.Errorf("no %s available for this account", kind)
		}
		return "", nil
	case 1:
		return ids[0], nil
	}
	if structured {
		return "", fmt.Errorf("multiple %ss found — pass --%s to select one in non-interactive mode", kind, flagName(kind))
	}
	idx, err := prompt.Select(fmt.Sprintf("Select a %s:", kind), labels)
	if err != nil {
		return "", err
	}
	return ids[idx], nil
}

// flagName maps a resource kind to its selecting flag (deployment is --service).
func flagName(kind string) string {
	if kind == "deployment" {
		return "service"
	}
	return kind
}

// scaffoldManifest writes a starter ownkube.yaml. When a deployment is linked it
// seeds name + type from that deployment; otherwise it uses the directory name.
func scaffoldManifest(ctx context.Context, c *client.Client, dir, depID string) error {
	if manifest.Exists(dir) {
		return fmt.Errorf("%s already exists", manifest.FileName)
	}
	m := &manifest.Manifest{
		Name:     filepath.Base(dir),
		Type:     "web",
		Port:     8080,
		Replicas: 1,
	}
	if depID != "" {
		if d, err := c.GetDeployment(ctx, depID); err == nil {
			m.Name = d.Name
			m.Type = string(d.ResourceType)
		}
	}
	return manifest.Write(dir, m, false)
}

func printBinding(cmd *cobra.Command, b linkstore.Binding) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "  organization: %s\n", b.OrganizationID)
	if b.ClusterID != "" {
		fmt.Fprintf(w, "  cluster:      %s\n", b.ClusterID)
	}
	if b.EnvironmentID != "" {
		fmt.Fprintf(w, "  environment:  %s\n", b.EnvironmentID)
	}
	if b.DeploymentID != "" {
		fmt.Fprintf(w, "  deployment:   %s\n", b.DeploymentID)
	}
}
