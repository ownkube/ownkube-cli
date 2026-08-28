package aws

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

const pollInterval = 5 * time.Second

func connectCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "connect",
		Short: "Connect a new AWS account to Ownkube",
		Long: `Start onboarding a new AWS account.

By default this opens the AWS console to a CloudFormation quick-create form.
Review it and click Create stack. The stack grants Ownkube a scoped
cross-account role and phones home when done; the CLI waits until access is
verified.

With --deploy, the stack is created for you using the AWS credentials already in
your environment (requires the AWS CLI). Ownkube never receives your AWS keys in
either mode.`,
		Args: cobra.NoArgs,
		RunE: runConnect,
	}
	c.Flags().Bool("deploy", false, "Deploy the stack now using local AWS credentials (requires the AWS CLI)")
	c.Flags().Bool("no-browser", false, "Print the quick-create URL but do not open a browser")
	c.Flags().Bool("no-wait", false, "Return once onboarding has started; do not poll for verification")
	c.Flags().Duration("timeout", 15*time.Minute, "How long to wait for verification")
	return c
}

func runConnect(cmd *cobra.Command, args []string) error {
	deploy, _ := cmd.Flags().GetBool("deploy")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	noWait, _ := cmd.Flags().GetBool("no-wait")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	api, err := ux.RequireClient()
	if err != nil {
		return err
	}

	res, err := api.ConnectAws(cmd.Context())
	if err != nil {
		return err
	}

	// Structured callers drive the flow themselves: hand back the full payload
	// (external ID, quick-create URL, and the machine-readable template) and let
	// them poll `okctl aws get -o json`.
	if ux.IsStructured() {
		return ux.Print(cmd.OutOrStdout(), res)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Account %s created in %s.\n\n", res.AccountId, res.Region)

	if deploy {
		if err := deployStack(cmd.Context(), out, res.Region, res.Template); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(out, "Open this URL and click Create stack:\n  %s\n\n", res.QuickCreateUrl)
		if !noBrowser {
			if err := ux.OpenBrowser(res.QuickCreateUrl); err != nil {
				fmt.Fprintf(out, "Could not open browser automatically: %v\n", err)
			}
		}
	}

	if noWait {
		fmt.Fprintf(out, "Run 'okctl aws get %s' to check status.\n", res.AccountId)
		return nil
	}

	return pollUntilVerified(cmd.Context(), out, api, res.AccountId, timeout)
}

// deployStack creates the onboarding CloudFormation stack via the local AWS CLI.
func deployStack(ctx context.Context, out io.Writer, region string, tpl api.AwsConnectTemplate) error {
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("--deploy needs the AWS CLI on your PATH; install it or omit --deploy to use the browser flow")
	}

	p := tpl.Parameters
	//nolint:gosec // args are server-provided ARNs/URLs/tokens, passed without a shell.
	c := exec.CommandContext(ctx, "aws", "cloudformation", "create-stack",
		"--stack-name", tpl.StackName,
		"--template-url", tpl.TemplateUrl,
		"--parameters",
		"ParameterKey=TrustArnParameter,ParameterValue="+p.TrustArnParameter,
		"ParameterKey=ExternalIdParameter,ParameterValue="+p.ExternalIdParameter,
		"ParameterKey=CallbackUrlParameter,ParameterValue="+p.CallbackUrlParameter,
		"ParameterKey=CallbackTokenParameter,ParameterValue="+p.CallbackTokenParameter,
		"--capabilities", "CAPABILITY_NAMED_IAM",
		"--region", region,
	)

	fmt.Fprintf(out, "Deploying stack %s with the AWS CLI...\n", tpl.StackName)
	combined, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws cloudformation create-stack failed: %w\n%s", err, combined)
	}
	fmt.Fprintf(out, "Stack creation started.\n\n")
	return nil
}

// pollUntilVerified polls the account until it reaches a terminal state or the
// timeout elapses, printing status transitions as they happen.
func pollUntilVerified(ctx context.Context, out io.Writer, api *client.Client, accountID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fmt.Fprintln(out, "Waiting for verification...")

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	last := ""
	for {
		a, err := api.GetAwsAccount(ctx, accountID)
		if err != nil {
			return err
		}
		if a.Status != last {
			fmt.Fprintf(out, "  status: %s\n", a.Status)
			last = a.Status
		}

		switch a.Status {
		case "verified":
			fmt.Fprintln(out, "\nConnected.")
			return printAccount(out, a)
		case "failed":
			fmt.Fprintln(out)
			_ = printAccount(out, a)
			if a.Failure.Kind != "" {
				return fmt.Errorf("connection failed: %s", a.Failure.Title)
			}
			return fmt.Errorf("connection failed")
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out after %s waiting for verification (last status: %s); run 'okctl aws get %s' to keep checking", timeout, last, accountID)
		case <-ticker.C:
		}
	}
}
