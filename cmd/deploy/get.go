package deploy

import (
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [app]",
		Short: "Get deployment details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}

			d, err := cl.GetDeployment(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), d)
			}
			rows := [][]string{
				{"FIELD", "VALUE"},
				{"ID", d.Id},
				{"Name", d.Name},
				{"Display Name", ux.Deref(d.DisplayName)},
				{"Type", string(d.ResourceType)},
				{"Status", d.Status},
				{"Status Message", d.StatusMessage},
				{"Cluster", d.ClusterId},
				{"Environment", d.EnvironmentId},
				{"Public Hostname", d.PublicHostname},
				{"Workflow PR", ux.Deref(d.WorkflowPullRequestUrl)},
			}
			rows = append(rows, sourceRows(d)...)
			// Best-effort: the details still print when revisions can't be read.
			if revs, err := cl.ListRevisions(cmd.Context(), d.Id); err == nil {
				if live := liveRevision(revs); live != nil {
					rows = append(rows, []string{"Live Revision", live.Id})
					if c := revisionCommit(*live); c != "" {
						rows = append(rows, []string{"Live Commit", c})
					}
				}
			}
			return ux.Print(cmd.OutOrStdout(), rows)
		},
	}
}

// sourceRows renders where the deployment's code comes from. Databases,
// caches and functions have no source block and get no rows.
func sourceRows(d *api.Deployment) [][]string {
	s := d.Source
	if s == nil {
		return nil
	}
	rows := [][]string{{"Source", string(s.Type)}}
	if s.Type == api.DeploymentSourceTypeGit {
		repo := s.Repository
		if s.Provider != "" {
			repo += " (" + string(s.Provider) + ")"
		}
		autoDeploy := "off"
		if s.AutoDeploy {
			autoDeploy = "on push to " + s.Branch
		}
		watch := "every push"
		if len(s.WatchPaths) > 0 {
			watch = strings.Join(s.WatchPaths, ", ")
		}
		waitForCI := "no"
		if s.WaitForChecks {
			waitForCI = "yes"
		}
		rows = append(rows,
			[]string{"Repository", repo},
			[]string{"Branch", s.Branch},
			[]string{"Auto Deploy", autoDeploy},
			[]string{"Watch Paths", watch},
			[]string{"Wait for CI", waitForCI},
			[]string{"Connection", s.ConnectionId},
		)
	}
	if s.Type != api.DeploymentSourceTypeImage {
		rows = append(rows,
			[]string{"Root Directory", orDefault(s.BuildContext, "(source root)")},
			[]string{"Dockerfile", orDefault(s.DockerfilePath, "(default)")},
		)
	}
	return rows
}

// liveRevision returns the revision currently serving, or nil.
func liveRevision(revs []api.Revision) *api.Revision {
	for i := range revs {
		if revs[i].Status == "live" {
			return &revs[i]
		}
	}
	return nil
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
