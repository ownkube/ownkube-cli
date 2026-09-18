package projects

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	var (
		name        string
		description string
		color       string
	)

	cmd := &cobra.Command{
		Use:   "update <project-id>",
		Short: "Update a project's name, description, or color",
		Long:  "Update a project. Only the flags you pass are changed. The slug is immutable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := validateColor(color)
			if err != nil {
				return err
			}

			body := api.UpdateProjectBody{}
			if cmd.Flags().Changed("name") {
				body.Name = &name
			}
			if cmd.Flags().Changed("description") {
				body.Description = &description
			}
			if c != nil {
				v := api.UpdateProjectBodyColor(*c)
				body.Color = &v
			}
			if body.Name == nil && body.Description == nil && body.Color == nil {
				return fmt.Errorf("nothing to update: pass at least one of --name, --description, or --color")
			}

			client, err := ux.RequireClient()
			if err != nil {
				return err
			}
			p, err := client.UpdateProject(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderProject(cmd, p)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New project name")
	cmd.Flags().StringVar(&description, "description", "", "New description (pass empty to clear)")
	cmd.Flags().StringVar(&color, "color", "", "New accent color: amber, blue, green, purple, or red")
	return cmd
}
