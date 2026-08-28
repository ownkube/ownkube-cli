package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file holds the remaining per-deployment action wrappers: lifecycle
// toggles (maintenance / rename / auto-deploy), build configuration, subdomain
// management, restart/rebuild/restore, and the read-only observability,
// telemetry, cache-connection and build-log surfaces. It follows the same
// pattern as deployment_write.go: call the generated WithResponse method, run
// checkError over the per-status error variants, then guard the payload.

// Restart calls POST /v1/deployments/{id}/restart, rolling the workload.
func (c *Client) Restart(ctx context.Context, id string) (*api.RestartResult, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdRestartWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromRestart(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Rebuild calls POST /v1/deployments/{id}/rebuild, queuing a fresh build of the
// current source. note is recorded on the queued revision when set.
func (c *Client) Rebuild(ctx context.Context, id string, note *string) (*api.RebuildResult, error) {
	params := &api.PostV1DeploymentsDeploymentIdRebuildParams{Note: note}
	resp, err := c.inner.PostV1DeploymentsDeploymentIdRebuildWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromRebuild(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// RestoreDatabase calls POST /v1/deployments/{id}/restore, replaying a managed
// database to targetTime (or the latest available point when nil).
func (c *Client) RestoreDatabase(ctx context.Context, id string, targetTime *string) (*api.RestoreDatabaseResult, error) {
	params := &api.PostV1DeploymentsDeploymentIdRestoreParams{TargetTime: targetTime}
	resp, err := c.inner.PostV1DeploymentsDeploymentIdRestoreWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromRestore(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetMaintenance calls PUT /v1/deployments/{id}/maintenance.
func (c *Client) SetMaintenance(ctx context.Context, id string, body api.SetMaintenanceBody) (*api.LifecycleDeploymentResult, error) {
	resp, err := c.inner.PutV1DeploymentsDeploymentIdMaintenanceWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromSetMaintenance(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// RenameDeployment calls PUT /v1/deployments/{id}/name, setting the cosmetic
// display label (empty clears it).
func (c *Client) RenameDeployment(ctx context.Context, id, displayName string) (*api.LifecycleDeploymentResult, error) {
	body := api.RenameDeploymentBody{DisplayName: displayName}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdNameWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromRename(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetAutoDeploy calls PUT /v1/deployments/{id}/auto-deploy.
func (c *Client) SetAutoDeploy(ctx context.Context, id string, enabled bool) (*api.LifecycleDeploymentResult, error) {
	body := api.SetAutoDeployBody{Enabled: enabled}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdAutoDeployWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromAutoDeploy(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetBuildArgs calls PUT /v1/deployments/{id}/build-args, replacing the build
// argument map wholesale.
func (c *Client) SetBuildArgs(ctx context.Context, id string, buildArgs map[string]string) (*api.BuildArgsResult, error) {
	body := api.BuildArgsBody{BuildArgs: buildArgs}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdBuildArgsWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromBuildArgs(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetBuildContext calls PUT /v1/deployments/{id}/build-context (empty clears it).
func (c *Client) SetBuildContext(ctx context.Context, id, contextPath string) (*api.BuildContextResult, error) {
	body := api.BuildContextBody{ContextPath: contextPath}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdBuildContextWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromBuildContext(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetBuilderSize calls PUT /v1/deployments/{id}/builder-size.
func (c *Client) SetBuilderSize(ctx context.Context, id string, size api.BuilderSizeBodyBuilderSize) (*api.BuilderSizeResult, error) {
	body := api.BuilderSizeBody{BuilderSize: size}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdBuilderSizeWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromBuilderSize(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ChangeSubdomain calls PUT /v1/deployments/{id}/subdomain, moving the
// deployment to a new address label under the platform domain.
func (c *Client) ChangeSubdomain(ctx context.Context, id, subdomain string) (*api.ChangeSubdomainResult, error) {
	body := api.ChangeSubdomainBody{Subdomain: subdomain}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdSubdomainWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromChangeSubdomain(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CheckSubdomain calls GET /v1/deployments/{id}/subdomain/check.
func (c *Client) CheckSubdomain(ctx context.Context, id, subdomain string) (*api.CheckSubdomainResult, error) {
	params := &api.GetV1DeploymentsDeploymentIdSubdomainCheckParams{Subdomain: subdomain}
	resp, err := c.inner.GetV1DeploymentsDeploymentIdSubdomainCheckWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCheckSubdomain(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SuggestSubdomains calls GET /v1/deployments/{id}/subdomain/suggest.
func (c *Client) SuggestSubdomains(ctx context.Context, id string, desired *string, limit *int) (*api.SuggestSubdomainsResult, error) {
	params := &api.GetV1DeploymentsDeploymentIdSubdomainSuggestParams{Desired: desired, Limit: limit}
	resp, err := c.inner.GetV1DeploymentsDeploymentIdSubdomainSuggestWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromSuggestSubdomains(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CacheConnection calls GET /v1/deployments/{id}/cache-connection.
func (c *Client) CacheConnection(ctx context.Context, id string) (*api.CacheConnectionResponse, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdCacheConnectionWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCacheConnection(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Observability calls GET /v1/deployments/{id}/observability. rangeSeconds and
// step narrow the time window (nil uses the server defaults).
func (c *Client) Observability(ctx context.Context, id string, rangeSeconds, step *int) (*api.DeploymentObservability, error) {
	params := &api.GetV1DeploymentsDeploymentIdObservabilityParams{RangeSeconds: rangeSeconds, Step: step}
	resp, err := c.inner.GetV1DeploymentsDeploymentIdObservabilityWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromObservability(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Telemetry calls GET /v1/deployments/{id}/telemetry.
func (c *Client) Telemetry(ctx context.Context, id string, rangeSeconds, step *int) (*api.DeploymentTelemetry, error) {
	params := &api.GetV1DeploymentsDeploymentIdTelemetryParams{RangeSeconds: rangeSeconds, Step: step}
	resp, err := c.inner.GetV1DeploymentsDeploymentIdTelemetryWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromTelemetry(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// BuildLogs calls GET /v1/deployments/{id}/revisions/{revisionId}/build-logs.
func (c *Client) BuildLogs(ctx context.Context, id, revisionID string, tailLines *int) (string, error) {
	params := &api.GetV1DeploymentsDeploymentIdRevisionsRevisionIdBuildLogsParams{TailLines: tailLines}
	resp, err := c.inner.GetV1DeploymentsDeploymentIdRevisionsRevisionIdBuildLogsWithResponse(ctx, id, revisionID, params)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromBuildLogs(resp), resp.Body); err != nil {
		return "", err
	}
	if resp.JSON200 == nil {
		return "", unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Logs, nil
}

// ---------------------------------------------------------------------------
// Error adapters (see client.go for the shared checkError contract)
// ---------------------------------------------------------------------------

func errorsFromRestart(r *api.PostV1DeploymentsDeploymentIdRestartResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromRebuild(r *api.PostV1DeploymentsDeploymentIdRebuildResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromRestore(r *api.PostV1DeploymentsDeploymentIdRestoreResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromSetMaintenance(r *api.PutV1DeploymentsDeploymentIdMaintenanceResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromRename(r *api.PutV1DeploymentsDeploymentIdNameResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromAutoDeploy(r *api.PutV1DeploymentsDeploymentIdAutoDeployResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromBuildArgs(r *api.PutV1DeploymentsDeploymentIdBuildArgsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromBuildContext(r *api.PutV1DeploymentsDeploymentIdBuildContextResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromBuilderSize(r *api.PutV1DeploymentsDeploymentIdBuilderSizeResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromChangeSubdomain(r *api.PutV1DeploymentsDeploymentIdSubdomainResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCheckSubdomain(r *api.GetV1DeploymentsDeploymentIdSubdomainCheckResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromSuggestSubdomains(r *api.GetV1DeploymentsDeploymentIdSubdomainSuggestResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCacheConnection(r *api.GetV1DeploymentsDeploymentIdCacheConnectionResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromObservability(r *api.GetV1DeploymentsDeploymentIdObservabilityResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromTelemetry(r *api.GetV1DeploymentsDeploymentIdTelemetryResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromBuildLogs(r *api.GetV1DeploymentsDeploymentIdRevisionsRevisionIdBuildLogsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
