package functions

import (
	"fmt"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

func sourceCmd() *cobra.Command {
	var codeOnly bool

	cmd := &cobra.Command{
		Use:   "source <deployment-id>",
		Short: "Print a function's inline source",
		Long: "Fetch a Compute function's stored source file. By default it prints a " +
			"metadata header followed by the code; pass --code-only to print just the " +
			"source (handy for piping to a file).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			src, err := api.GetFunctionSource(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), src)
			}
			if codeOnly {
				fmt.Fprintln(cmd.OutOrStdout(), src.Code)
				return nil
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "# %s (%s)", src.Filename, string(src.Language))
			if src.SourceHash != "" {
				fmt.Fprintf(out, " sha:%s", src.SourceHash)
			}
			fmt.Fprintf(out, "\n\n%s\n", src.Code)
			return nil
		},
	}

	cmd.Flags().BoolVar(&codeOnly, "code-only", false, "Print only the source code, without the metadata header")
	return cmd
}
