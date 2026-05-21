package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/version"
)

// Client wraps the generated OpenAPI client with API key injection and error handling.
type Client struct {
	inner  *api.ClientWithResponses
	apiURL string
}

// New creates a new Client targeting the given API base URL with the provided API key.
//
// If OKCTL_BASIC_AUTH is set ("user:pass"), it is sent as an HTTP Basic
// Authorization header on every request — useful for dev environments behind
// an HTTP gateway.
func New(apiURL, apiKey string) (*Client, error) {
	basicUser, basicPass, hasBasic := parseBasicAuthEnv(os.Getenv("OKCTL_BASIC_AUTH"))

	editor := api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("User-Agent", "okctl/"+version.Version)
		if hasBasic {
			req.SetBasicAuth(basicUser, basicPass)
		}
		return nil
	})

	inner, err := api.NewClientWithResponses(apiURL+"/api/cli", editor)
	if err != nil {
		return nil, fmt.Errorf("creating API client: %w", err)
	}

	return &Client{inner: inner, apiURL: apiURL}, nil
}

// parseBasicAuthEnv splits "user:pass" into its parts. Returns hasBasic=false
// when the env var is empty or malformed.
func parseBasicAuthEnv(v string) (user, pass string, hasBasic bool) {
	if v == "" {
		return "", "", false
	}
	idx := strings.IndexByte(v, ':')
	if idx < 0 {
		return v, "", true
	}
	return v[:idx], v[idx+1:], true
}

// UserInfo represents the authenticated user's profile.
type UserInfo struct {
	ID            string
	Name          string
	Email         string
	Image         *string
	EmailVerified bool
}

// GetUserInfo calls GET /v1/info and returns the user's profile.
func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	resp, err := c.inner.GetV1InfoWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetInfo(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	u := resp.JSON200.User
	return &UserInfo{
		ID:            u.Id,
		Name:          u.Name,
		Email:         string(u.Email),
		Image:         u.Image,
		EmailVerified: u.EmailVerified,
	}, nil
}

// ListDeploymentsFilter narrows the deployment list by cluster or environment.
type ListDeploymentsFilter struct {
	ClusterID     string
	EnvironmentID string
}

// ListDeployments calls GET /v1/deployments.
func (c *Client) ListDeployments(ctx context.Context, f ListDeploymentsFilter) ([]api.Deployment, error) {
	params := &api.GetV1DeploymentsParams{}
	if f.ClusterID != "" {
		params.ClusterId = &f.ClusterID
	}
	if f.EnvironmentID != "" {
		params.EnvironmentId = &f.EnvironmentID
	}

	resp, err := c.inner.GetV1DeploymentsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListDeployments(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Deployments, nil
}

// GetDeployment calls GET /v1/deployments/{id}.
func (c *Client) GetDeployment(ctx context.Context, id string) (*api.Deployment, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetDeployment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetDeploymentStatus calls GET /v1/deployments/{id}/status.
func (c *Client) GetDeploymentStatus(ctx context.Context, id string) (*api.DeploymentStatusResponse, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdStatusWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetStatus(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// LogsOptions configures a logs query.
type LogsOptions struct {
	RangeSeconds int
	Limit        int
	Filter       string
}

// GetDeploymentLogs calls GET /v1/deployments/{id}/logs.
func (c *Client) GetDeploymentLogs(ctx context.Context, id string, opts LogsOptions) ([]api.LogEntry, error) {
	params := &api.GetV1DeploymentsDeploymentIdLogsParams{}
	if opts.RangeSeconds > 0 {
		params.RangeSeconds = &opts.RangeSeconds
	}
	if opts.Limit > 0 {
		params.Limit = &opts.Limit
	}
	if opts.Filter != "" {
		params.Filter = &opts.Filter
	}

	resp, err := c.inner.GetV1DeploymentsDeploymentIdLogsWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetLogs(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Entries, nil
}

// GetDeploymentConnection calls GET /v1/deployments/{id}/connection.
func (c *Client) GetDeploymentConnection(ctx context.Context, id string) (*api.ConnectionDetailsResponse, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdConnectionWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetConnection(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ListRevisions calls GET /v1/deployments/{id}/revisions.
func (c *Client) ListRevisions(ctx context.Context, id string) ([]api.Revision, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdRevisionsWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListRevisions(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Revisions, nil
}

// ListClusters calls GET /v1/clusters.
func (c *Client) ListClusters(ctx context.Context) ([]api.Cluster, error) {
	resp, err := c.inner.GetV1ClustersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListClusters(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Clusters, nil
}

// GetCluster calls GET /v1/clusters/{id}.
func (c *Client) GetCluster(ctx context.Context, id string) (*api.Cluster, error) {
	resp, err := c.inner.GetV1ClustersClusterIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetCluster(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ListEnvironments calls GET /v1/environments.
func (c *Client) ListEnvironments(ctx context.Context) ([]api.Environment, error) {
	resp, err := c.inner.GetV1EnvironmentsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListEnvironments(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Environments, nil
}

// GetEnvironment calls GET /v1/environments/{id}.
func (c *Client) GetEnvironment(ctx context.Context, id string) (*api.Environment, error) {
	resp, err := c.inner.GetV1EnvironmentsEnvironmentIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetEnvironment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ListOrganizations calls GET /v1/organizations.
func (c *Client) ListOrganizations(ctx context.Context) ([]api.OrganizationSummary, error) {
	resp, err := c.inner.GetV1OrganizationsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListOrganizations(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Organizations, nil
}

// ListRegistries calls GET /v1/registries.
func (c *Client) ListRegistries(ctx context.Context) ([]api.Registry, error) {
	resp, err := c.inner.GetV1RegistriesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListRegistries(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Registries, nil
}

// GetRegistry calls GET /v1/registries/{id}.
func (c *Client) GetRegistry(ctx context.Context, id string) (*api.Registry, error) {
	resp, err := c.inner.GetV1RegistriesRegistryIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetRegistry(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ---------------------------------------------------------------------------
// Error handling helpers
// ---------------------------------------------------------------------------

func checkError(status int, candidates []*api.ErrorResponse, body []byte) error {
	for _, e := range candidates {
		if e != nil {
			return fmt.Errorf("API error %d (%s): %s", status, e.Code, e.Error)
		}
	}
	if status >= 400 {
		var generic api.ErrorResponse
		if err := json.Unmarshal(body, &generic); err == nil && generic.Error != "" {
			return fmt.Errorf("API error %d (%s): %s", status, generic.Code, generic.Error)
		}
		return fmt.Errorf("API error %d: %s", status, truncate(body))
	}
	return nil
}

func unexpectedStatus(status int, body []byte) error {
	return fmt.Errorf("unexpected response (status %d): %s", status, truncate(body))
}

func truncate(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}

// Per-endpoint adapters that collect every populated JSONxxx error variant.
// oapi-codegen generates one *ErrorResponse field per documented status code,
// so this is the cleanest way to share `checkError`.

func errorsFromGetInfo(r *api.GetV1InfoResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON401}
}

func errorsFromListDeployments(r *api.GetV1DeploymentsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetDeployment(r *api.GetV1DeploymentsDeploymentIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetStatus(r *api.GetV1DeploymentsDeploymentIdStatusResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetLogs(r *api.GetV1DeploymentsDeploymentIdLogsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetConnection(r *api.GetV1DeploymentsDeploymentIdConnectionResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListRevisions(r *api.GetV1DeploymentsDeploymentIdRevisionsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListClusters(r *api.GetV1ClustersResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetCluster(r *api.GetV1ClustersClusterIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListEnvironments(r *api.GetV1EnvironmentsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetEnvironment(r *api.GetV1EnvironmentsEnvironmentIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListOrganizations(r *api.GetV1OrganizationsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListRegistries(r *api.GetV1RegistriesResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetRegistry(r *api.GetV1RegistriesRegistryIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
