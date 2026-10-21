// Package status implements `okctl status` (alias `overview`): who you're
// logged in as and, for a linked directory or --project, that project's
// environments and apps with their URLs and health.
package status

import (
	"context"
	"fmt"
	"sort"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	linkstore "github.com/ownkube/okctl/internal/link"
	"github.com/spf13/cobra"
)

type user struct {
	UserID string `json:"user_id" yaml:"user_id"`
	Name   string `json:"name" yaml:"name"`
	Email  string `json:"email" yaml:"email"`
	APIURL string `json:"api_url" yaml:"api_url"`
}

type app struct {
	ID            string `json:"id" yaml:"id"`
	Name          string `json:"name" yaml:"name"`
	Type          string `json:"type" yaml:"type"`
	Status        string `json:"status" yaml:"status"`
	StatusMessage string `json:"statusMessage,omitempty" yaml:"statusMessage,omitempty"`
	EnvironmentID string `json:"environmentId" yaml:"environmentId"`
	Environment   string `json:"environment" yaml:"environment"`
	URL           string `json:"url,omitempty" yaml:"url,omitempty"`
	Linked        bool   `json:"linked,omitempty" yaml:"linked,omitempty"`
}

type overview struct {
	User         user              `json:"user" yaml:"user"`
	Project      *api.Project      `json:"project" yaml:"project"`
	Environments []api.Environment `json:"environments" yaml:"environments"`
	Apps         []app             `json:"apps" yaml:"apps"`
}

// New returns the `okctl status` command.
func New() *cobra.Command {
	var projectRef string
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"overview"},
		Short:   "Show the linked project's apps, environments, URLs and health",
		Long: "Show who you're logged in as and, for this directory's linked project " +
			"(or --project), every environment and app with its URL and status. " +
			"'okctl whoami' shows just the login.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			ctx, out := cmd.Context(), cmd.OutOrStdout()

			info, err := cl.GetUserInfo(ctx)
			if err != nil {
				return err
			}
			ov := overview{
				User:         user{UserID: info.ID, Name: info.Name, Email: info.Email, APIURL: ux.APIURL()},
				Environments: []api.Environment{},
				Apps:         []app{},
			}

			binding, _, _ := ux.Linked(".")
			projectID, err := ux.ResolveProject(ctx, cl, projectRef)
			if err != nil {
				return err
			}
			if projectID == "" {
				projectID = linkedProject(ctx, cl, binding)
			}
			if projectID != "" {
				if err := fill(ctx, cl, &ov, projectID, binding); err != nil {
					return err
				}
			}

			if ux.IsStructured() {
				return ux.Print(out, ov)
			}
			fmt.Fprintf(out, "Logged in as %s <%s>\n", info.Name, info.Email)
			if ov.Project == nil {
				fmt.Fprintln(out, "This directory isn't linked. Run 'okctl link', or pass --project.")
				return nil
			}
			fmt.Fprintf(out, "Project:     %s (%s)\n", ov.Project.Name, ov.Project.Id)
			for _, a := range ov.Apps {
				if a.Linked {
					fmt.Fprintf(out, "Linked app:  %s (%s, %s)\n", a.Name, a.Environment, a.ID)
				}
			}
			fmt.Fprintln(out)
			if len(ov.Apps) == 0 {
				fmt.Fprintf(out, "No apps yet in %d environment(s). Deploy this directory with 'okctl up'.\n", len(ov.Environments))
				return nil
			}
			rows := [][]string{{"ENVIRONMENT", "APP", "TYPE", "STATUS", "URL", "ID"}}
			for _, a := range ov.Apps {
				name := a.Name
				if a.Linked {
					name += " *"
				}
				rows = append(rows, []string{a.Environment, name, a.Type, a.Status, a.URL, a.ID})
			}
			return ux.Print(out, rows)
		},
	}
	cmd.Flags().StringVarP(&projectRef, "project", "p", "", "Project name or id (default: this directory's linked project)")
	return cmd
}

// fill loads the project, its environments and its apps into ov.
func fill(ctx context.Context, cl *client.Client, ov *overview, projectID string, b linkstore.Binding) error {
	project, err := cl.GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	ov.Project = project

	envs, err := cl.ListEnvironments(ctx)
	if err != nil {
		return err
	}
	envNames := map[string]string{}
	for _, e := range envs {
		if ux.Deref(e.ProjectId) == projectID {
			ov.Environments = append(ov.Environments, e)
			envNames[e.Id] = e.Name
		}
	}

	deps, err := cl.ListDeployments(ctx, client.ListDeploymentsFilter{ProjectID: projectID})
	if err != nil {
		return err
	}
	for _, d := range deps {
		a := app{
			ID:            d.Id,
			Name:          d.Name,
			Type:          string(d.ResourceType),
			Status:        d.Status,
			StatusMessage: d.StatusMessage,
			EnvironmentID: d.EnvironmentId,
			Environment:   envNames[d.EnvironmentId],
			Linked:        d.Id == b.DeploymentID,
		}
		if a.Environment == "" {
			a.Environment = d.EnvironmentId
		}
		if d.PublicHostname != "" {
			a.URL = "https://" + d.PublicHostname
		}
		ov.Apps = append(ov.Apps, a)
	}
	sort.SliceStable(ov.Apps, func(i, j int) bool {
		if ov.Apps[i].Environment != ov.Apps[j].Environment {
			return ov.Apps[i].Environment < ov.Apps[j].Environment
		}
		return ov.Apps[i].Name < ov.Apps[j].Name
	})
	return nil
}

// linkedProject finds the project of the linked environment (or, failing that,
// the linked app's environment). "" when the directory isn't linked.
func linkedProject(ctx context.Context, cl *client.Client, b linkstore.Binding) string {
	envID := b.EnvironmentID
	if envID == "" && b.DeploymentID != "" {
		if d, err := cl.GetDeployment(ctx, b.DeploymentID); err == nil {
			envID = d.EnvironmentId
		}
	}
	if envID == "" {
		return ""
	}
	env, err := cl.GetEnvironment(ctx, envID)
	if err != nil {
		return ""
	}
	return ux.Deref(env.ProjectId)
}
