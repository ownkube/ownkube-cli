package client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file holds the deployment write surface (create / update / lifecycle
// actions). Read wrappers live in client.go. Every method mirrors that file's
// pattern: call the generated WithResponse method, run checkError over the
// per-status error variants, then guard the success payload.
//
// Create and update send raw JSON bytes via the WithBody variants: the create
// body is an OpenAPI `oneOf` union which oapi-codegen renders as an unexported
// field with no marshal helper, so the typed call would send an empty object.
// Passing the manifest bytes straight through sidesteps that gap and keeps the
// server the single source of truth for body validation.

// CreateDeployment calls POST /v1/deployments with a raw JSON manifest body.
func (c *Client) CreateDeployment(ctx context.Context, jsonBody []byte) (*api.CreateDeploymentResponse, error) {
	resp, err := c.inner.PostV1DeploymentsWithBodyWithResponse(ctx, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCreateDeployment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateDeployment calls PATCH /v1/deployments/{id} with a raw JSON body.
func (c *Client) UpdateDeployment(ctx context.Context, id string, jsonBody []byte) (*api.DeploymentActionResult, error) {
	resp, err := c.inner.PatchV1DeploymentsDeploymentIdWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateDeployment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateFunction calls PATCH /v1/deployments/{id}/function with a raw JSON body.
func (c *Client) UpdateFunction(ctx context.Context, id string, jsonBody []byte) (*api.DeploymentActionResult, error) {
	resp, err := c.inner.PatchV1DeploymentsDeploymentIdFunctionWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateFunction(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteDeployment calls DELETE /v1/deployments/{id} (soft delete).
func (c *Client) DeleteDeployment(ctx context.Context, id string) error {
	resp, err := c.inner.DeleteV1DeploymentsDeploymentIdWithResponse(ctx, id)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeleteDeployment(resp), resp.Body); err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return nil
}

// CopyDeployment calls POST /v1/deployments/{id}/copy, cloning the deployment
// into another environment/cluster.
func (c *Client) CopyDeployment(ctx context.Context, id string, body api.CopyDeploymentBody) (*api.DeploymentActionResult, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdCopyWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCopyDeployment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ResetDatabasePassword calls POST /v1/deployments/{id}/reset-password. The
// returned password is a live credential — callers must not log it.
func (c *Client) ResetDatabasePassword(ctx context.Context, id string) (*api.ResetPasswordResult, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdResetPasswordWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromResetPassword(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetJobRuns calls GET /v1/deployments/{id}/job-runs.
func (c *Client) GetJobRuns(ctx context.Context, id string) (*api.JobRunHistory, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdJobRunsWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetJobRuns(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// TriggerJobRun calls POST /v1/deployments/{id}/job-runs.
func (c *Client) TriggerJobRun(ctx context.Context, id string) (*api.TriggerJobRunResult, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdJobRunsWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromTriggerJobRun(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CancelJobRun calls DELETE /v1/deployments/{id}/job-runs/{jobName}.
func (c *Client) CancelJobRun(ctx context.Context, id, jobName string) (*api.CancelJobRunResult, error) {
	resp, err := c.inner.DeleteV1DeploymentsDeploymentIdJobRunsJobNameWithResponse(ctx, id, jobName)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCancelJobRun(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Rollback calls POST /v1/deployments/{id}/revisions/{revisionId}/rollback,
// re-deploying an earlier revision. Returns the new revision it creates.
func (c *Client) Rollback(ctx context.Context, id, revisionID string, note *string) (*api.Revision, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdRevisionsRevisionIdRollbackWithResponse(
		ctx, id, revisionID, api.RollbackBody{Note: note})
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromRollback(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Promote calls POST /v1/deployments/{id}/promote, shipping the source
// deployment's live revision to a target deployment. Returns the new revision.
func (c *Client) Promote(ctx context.Context, id, targetID string, note *string) (*api.Revision, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdPromoteWithResponse(
		ctx, id, api.PromoteBody{TargetDeploymentId: targetID, Note: note})
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromPromote(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ---------------------------------------------------------------------------
// Error adapters (see client.go for the shared checkError contract)
// ---------------------------------------------------------------------------

func errorsFromCreateDeployment(r *api.PostV1DeploymentsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUpdateDeployment(r *api.PatchV1DeploymentsDeploymentIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUpdateFunction(r *api.PatchV1DeploymentsDeploymentIdFunctionResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDeleteDeployment(r *api.DeleteV1DeploymentsDeploymentIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCopyDeployment(r *api.PostV1DeploymentsDeploymentIdCopyResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromResetPassword(r *api.PostV1DeploymentsDeploymentIdResetPasswordResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetJobRuns(r *api.GetV1DeploymentsDeploymentIdJobRunsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromTriggerJobRun(r *api.PostV1DeploymentsDeploymentIdJobRunsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCancelJobRun(r *api.DeleteV1DeploymentsDeploymentIdJobRunsJobNameResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromRollback(r *api.PostV1DeploymentsDeploymentIdRevisionsRevisionIdRollbackResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromPromote(r *api.PostV1DeploymentsDeploymentIdPromoteResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
