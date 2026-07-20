package environments

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
		Use:   "update <environment-id>",
		Short: "Update an environment's name, description, or color",
		Long: "Update an environment. Only the flags you pass are changed. To edit " +
			"shared environment variables, use 'okctl environments set-env'.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := validateColor(color)
			if err != nil {
				return err
			}

			body := api.UpdateEnvironmentBody{}
			if cmd.Flags().Changed("name") {
				body.Name = &name
			}
			if cmd.Flags().Changed("description") {
				body.Description = &description
			}
			if c != nil {
				v := api.UpdateEnvironmentBodyColor(*c)
				body.Color = &v
			}
			if body.Name == nil && body.Description == nil && body.Color == nil {
				return fmt.Errorf("nothing to update: pass at least one of --name, --description, or --color")
			}

			client, err := ux.RequireClient()
			if err != nil {
				return err
			}
			e, err := client.UpdateEnvironment(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderEnvironment(cmd, e)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New environment name")
	cmd.Flags().StringVar(&description, "description", "", "New description (pass empty to clear)")
	cmd.Flags().StringVar(&color, "color", "", "New accent color: amber, blue, green, purple, or red")
	return cmd
}
