package projects

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	var (
		name        string
		slug        string
		description string
		color       string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project",
		Long: "Create a project. --name and --slug are required. The slug is " +
			"lowercase, starts with a letter, and contains only letters, numbers, and hyphens.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if name == "" || slug == "" {
				return fmt.Errorf("--name and --slug are required")
			}
			c, err := validateColor(color)
			if err != nil {
				return err
			}

			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			p, err := api.CreateProject(cmd.Context(), buildCreateBody(name, slug, description, c))
			if err != nil {
				return err
			}
			return renderProject(cmd, p)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe slug (required)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	cmd.Flags().StringVar(&color, "color", "", "Accent color: amber, blue, green, purple, or red")
	return cmd
}

func buildCreateBody(name, slug, description string, color *string) api.CreateProjectBody {
	body := api.CreateProjectBody{Name: name, Slug: slug}
	if description != "" {
		body.Description = &description
	}
	if color != nil {
		v := api.CreateProjectBodyColor(*color)
		body.Color = &v
	}
	return body
}
