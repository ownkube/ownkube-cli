package up

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	linkstore "github.com/ownkube/okctl/internal/link"
	"github.com/spf13/cobra"
)

// createFlags are the overrides for a first `okctl up` in an unlinked
// directory. Every one is optional: the server resolves the rest (one shared
// resolver behind both this command and the MCP deploy_uploaded_source tool).
type createFlags struct {
	name, region, resourceType, project, environment string
	port                                             int
	public                                           bool
}

// createFlagNames are the flags that only apply when `up` creates the app.
var createFlagNames = []string{"name", "region", "port", "type", "public", "project", "environment"}

func (f *createFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.name, "name", "", "New app only: exact name (default: this directory's name, made unique)")
	cmd.Flags().StringVar(&f.region, "region", "", "New app only: region id from 'okctl regions list' (default: the region you already use)")
	cmd.Flags().IntVar(&f.port, "port", 0, "New app only: port the app listens on (default: Dockerfile EXPOSE, else the framework default, else 8080)")
	cmd.Flags().StringVar(&f.resourceType, "type", "", "New app only: web or worker (default: web)")
	cmd.Flags().BoolVar(&f.public, "public", true, "New app only: give a web app a public URL")
	cmd.Flags().StringVar(&f.project, "project", "", "New app only: project id (default: your default project)")
	cmd.Flags().StringVar(&f.environment, "environment", "", "New app only: environment id (default: the project's default environment)")
}

// rejectCreateFlags errors when a create-only flag is set on a redeploy, so a
// --port or --region never silently does nothing.
func rejectCreateFlags(cmd *cobra.Command) error {
	var set []string
	for _, name := range createFlagNames {
		if cmd.Flags().Changed(name) {
			set = append(set, "--"+name)
		}
	}
	if len(set) == 0 {
		return nil
	}
	return fmt.Errorf("%s only apply when creating a new app; this directory already deploys to an existing one",
		strings.Join(set, ", "))
}

// linkedDeploymentID returns the deployment bound to dir, or "" when the
// directory isn't linked.
func linkedDeploymentID(dir string) (string, error) {
	key, err := linkstore.ResolveKey(dir)
	if err != nil {
		return "", err
	}
	binding, ok, err := linkstore.NewManager(ux.Config().Dir()).Get(key)
	if err != nil || !ok {
		return "", err
	}
	return binding.DeploymentID, nil
}

// buildCreateBody turns the flags into the source-deploy request, sending only
// what the user set. nameHint is the linked root's directory name.
func buildCreateBody(cmd *cobra.Command, f *createFlags, uploadID, nameHint string, note *string) (api.SourceDeployBody, error) {
	parsed, err := client.ParseUploadID(uploadID)
	if err != nil {
		return api.SourceDeployBody{}, err
	}
	body := api.SourceDeployBody{UploadId: parsed, NameHint: &nameHint, Note: note}
	changed := cmd.Flags().Changed
	if changed("name") {
		body.Name = &f.name
	}
	if changed("region") {
		body.Region = &f.region
	}
	if changed("port") {
		body.Port = &f.port
	}
	if changed("type") {
		if f.resourceType != "web" && f.resourceType != "worker" {
			return body, fmt.Errorf("--type must be web or worker, got %q", f.resourceType)
		}
		t := api.SourceDeployBodyResourceType(f.resourceType)
		body.ResourceType = &t
	}
	if changed("public") {
		body.Public = &f.public
	}
	if changed("project") {
		body.ProjectId = &f.project
	}
	if changed("environment") {
		body.EnvironmentId = &f.environment
	}
	return body, nil
}

// createApp creates the app from an uploaded tarball, reports the defaults the
// server chose, and links dir to it so the next `up` redeploys. Returns the
// result for the caller to follow.
func createApp(ctx context.Context, cmd *cobra.Command, cl *client.Client, f *createFlags, dir, uploadID string, note *string) (*api.SourceDeployResult, error) {
	key, err := linkstore.ResolveKey(dir)
	if err != nil {
		return nil, err
	}
	body, err := buildCreateBody(cmd, f, uploadID, filepath.Base(key), note)
	if err != nil {
		return nil, err
	}
	result, err := cl.DeployUploadedSource(ctx, body)
	if err != nil {
		return nil, err
	}

	binding := linkstore.Binding{
		OrganizationID: organizationID(ctx, cl),
		ClusterID:      result.ClusterId,
		EnvironmentID:  result.EnvironmentId,
		DeploymentID:   result.DeploymentId,
	}
	linkErr := linkstore.NewManager(ux.Config().Dir()).Set(key, binding)

	if !ux.IsStructured() {
		printCreated(cmd.OutOrStdout(), result, key, linkErr)
	}
	if result.RevisionId == "" {
		return nil, fmt.Errorf("the app was created but its build didn't start; run 'okctl up' again to deploy it")
	}
	return result, nil
}

// organizationID is the org to record on the new link: the configured default,
// else the account's only org. Best-effort: "" leaves it to the next lookup.
func organizationID(ctx context.Context, cl *client.Client) string {
	if o := ux.Organization(); o != "" {
		return o
	}
	if orgs, err := cl.ListOrganizations(ctx); err == nil && len(orgs) == 1 {
		return orgs[0].Id
	}
	return ""
}

func printCreated(out io.Writer, r *api.SourceDeployResult, linkKey string, linkErr error) {
	// Required+nullable renders as a value: a null `defaults` is the zero struct.
	d := r.Defaults
	if d.Name == "" {
		fmt.Fprintf(out, "Created app %s.\n", r.DeploymentId)
	} else {
		fmt.Fprintf(out, "Created %s app %q in %s.\n", d.ResourceType, d.Name, d.RegionLabel)
		fmt.Fprintf(out, "  Port:   %d (%s)\n", d.Port, portSourceLabel(string(d.PortSource)))
		switch {
		case r.PublicHostname != "":
			fmt.Fprintf(out, "  URL:    https://%s\n", r.PublicHostname)
		case !d.Public:
			fmt.Fprintln(out, "  URL:    none (private)")
		}
	}
	if linkErr != nil {
		fmt.Fprintf(out, "Could not link this directory (%v); pass --service %s to redeploy it.\n", linkErr, r.DeploymentId)
		return
	}
	fmt.Fprintf(out, "Linked %s to it; the next 'okctl up' redeploys this app.\n", linkKey)
}

func portSourceLabel(source string) string {
	switch source {
	case "dockerfile":
		return "from the Dockerfile's EXPOSE"
	case "language":
		return "detected framework default"
	case "fallback":
		return "default; pass --port if your app listens elsewhere"
	default:
		return "--port"
	}
}
