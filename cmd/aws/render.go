package aws

import (
	"fmt"
	"io"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
)

// printAccount renders a single account as a field/value table, appending a
// failure block when the account is in a failed state.
func printAccount(w io.Writer, a *api.AwsAccount) error {
	rows := [][]string{
		{"FIELD", "VALUE"},
		{"ID", a.Id},
		{"Status", a.Status},
		{"Region", a.Region},
		{"AWS Account", a.AwsAccountId},
		{"Stack Status", a.CloudFormationStackStatus},
		{"Active Clusters", fmt.Sprintf("%d", len(a.ActiveClusters))},
	}
	if a.Failure.Kind != "" {
		rows = append(rows,
			[]string{"Failure", a.Failure.Title},
			[]string{"Remediation", string(a.Failure.Kind)},
			[]string{"Detail", a.Failure.Description},
		)
		if a.Failure.DocsUrl != "" {
			rows = append(rows, []string{"Docs", a.Failure.DocsUrl})
		}
	}
	return ux.Print(w, rows)
}

// printAccounts renders a list of accounts as a summary table.
func printAccounts(w io.Writer, accounts []api.AwsAccount) error {
	rows := [][]string{{"ID", "STATUS", "REGION", "AWS ACCOUNT", "CLUSTERS"}}
	for i := range accounts {
		a := &accounts[i]
		rows = append(rows, []string{
			a.Id, a.Status, a.Region, a.AwsAccountId,
			fmt.Sprintf("%d", len(a.ActiveClusters)),
		})
	}
	return ux.Print(w, rows)
}
