package deploy

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func connectionInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connection-info <deployment-id>",
		Short: "Show a database's or cache's full connection info, including credentials",
		Long: "Print everything needed to connect to a database or cache: host, port, " +
			"database name, username, password, and the connection string for apps " +
			"in the same environment. When public access is on, also prints the " +
			"public connection string for connecting from outside. The output " +
			"contains live credentials.\n\n" +
			"To give an app the connection string without printing it, use " +
			"`okctl deploy link`.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			info, err := cl.ConnectionInfo(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), info)
			}

			rows := [][]string{
				{"FIELD", "VALUE"},
				{"Type", string(info.ResourceType)},
				{"Host", info.Host},
				{"Port", fmt.Sprintf("%g", info.Port)},
			}
			if info.Database != "" {
				rows = append(rows, []string{"Database", info.Database})
			}
			if info.Username != "" {
				rows = append(rows, []string{"Username", info.Username})
			}
			rows = append(rows,
				[]string{"Password", info.Password},
				[]string{"URI", info.Uri},
			)
			if info.PublicHostname != "" {
				rows = append(rows,
					[]string{"Public host", info.PublicHostname},
					[]string{"Public port", fmt.Sprintf("%g", info.PublicPort)},
					[]string{"Public URI", info.PublicUri},
					[]string{"Public ready", fmt.Sprintf("%t", info.PublicReady)},
				)
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}
