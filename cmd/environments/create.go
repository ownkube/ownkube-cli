package environments

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
		Short: "Create an environment",
		Long: "Create an environment. --name and --slug are required. The slug is " +
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

			body := api2CreateBody(name, slug, description, c)
			e, err := api.CreateEnvironment(cmd.Context(), body)
			if err != nil {
				return err
			}
			return renderEnvironment(cmd, e)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Environment name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe slug (required)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	cmd.Flags().StringVar(&color, "color", "", "Accent color: amber, blue, green, purple, or red")
	return cmd
}

func api2CreateBody(name, slug, description string, color *string) api.CreateEnvironmentBody {
	body := api.CreateEnvironmentBody{Name: name, Slug: slug}
	if description != "" {
		body.Description = &description
	}
	if color != nil {
		v := api.CreateEnvironmentBodyColor(*color)
		body.Color = &v
	}
	return body
}
