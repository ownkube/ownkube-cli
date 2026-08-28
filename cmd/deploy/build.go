package deploy

import (
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func buildArgsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "build-args <deployment-id>",
		Short: "Replace the build argument map (KEY=VALUE pairs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			pairs, _ := cmd.Flags().GetStringArray("arg")
			buildArgs := make(map[string]string, len(pairs))
			for _, p := range pairs {
				k, v, ok := strings.Cut(p, "=")
				if !ok {
					return fmt.Errorf("invalid --arg %q: expected KEY=VALUE", p)
				}
				buildArgs[k] = v
			}
			res, err := cl.SetBuildArgs(cmd.Context(), args[0], buildArgs)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			rows := [][]string{{"KEY", "VALUE"}}
			for k, v := range res.BuildArgs {
				rows = append(rows, []string{k, v})
			}
			if len(res.BuildArgs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Build args cleared.")
				return nil
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
	c.Flags().StringArray("arg", nil, "Build argument as KEY=VALUE (repeatable). Pass none to clear.")
	return c
}

func buildContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build-context <deployment-id> <context-path>",
		Short: "Set the build context root, relative to the repo (empty clears it)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			contextPath := ""
			if len(args) == 2 {
				contextPath = args[1]
			}
			res, err := cl.SetBuildContext(cmd.Context(), args[0], contextPath)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Build context set to %q.\n", res.ContextPath)
			return nil
		},
	}
}

func builderSizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "builder-size <deployment-id> <small|standard|large>",
		Short: "Set the build machine size",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			size := api.BuilderSizeBodyBuilderSize(args[1])
			if !size.Valid() {
				return fmt.Errorf("invalid builder size %q: use small, standard, or large", args[1])
			}
			res, err := cl.SetBuilderSize(cmd.Context(), args[0], size)
			if err != nil {
				return err
			}
			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Builder size set to %s.\n", string(res.BuilderSize))
			return nil
		},
	}
}
