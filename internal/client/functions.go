package client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// Function surface (list / source / deploy) plus the deployment move-to-project
// action. Ember functions are cluster-less deployments (they attach to a cloud
// account + region), listed separately from cluster-hosted workloads.
//
// DeployFunction sends raw JSON bytes via the WithBody variant, mirroring
// UpdateFunction: the combined code+settings body is deeply nested and the
// server is the single source of truth for its validation.

// ListFunctions calls GET /v1/functions.
func (c *Client) ListFunctions(ctx context.Context) ([]api.Deployment, error) {
	resp, err := c.inner.GetV1FunctionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListFunctions(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Functions, nil
}

// GetFunctionSource calls GET /v1/deployments/{id}/function/source, returning
// the function's inline source file. A 404 means the deployment is not a
// function or has no recorded source.
func (c *Client) GetFunctionSource(ctx context.Context, id string) (*api.FunctionSourceResponse, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdFunctionSourceWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetFunctionSource(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeployFunction calls POST /v1/deployments/{id}/function/deploy with a raw
// JSON body (combined code + settings, one revision).
func (c *Client) DeployFunction(ctx context.Context, id string, jsonBody []byte) (*api.DeploymentActionResult, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdFunctionDeployWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeployFunction(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// MoveToProject calls PUT /v1/deployments/{id}/project, re-filing the deployment
// (and any managed database/cache in its group) under another project. This is
// a control-plane-only change: no revision, no re-sync.
func (c *Client) MoveToProject(ctx context.Context, id, targetProjectID string) (*api.MoveToProjectResult, error) {
	body := api.MoveToProjectBody{TargetProjectId: targetProjectID}
	resp, err := c.inner.PutV1DeploymentsDeploymentIdProjectWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromMoveToProject(resp), resp.Body); err != nil {
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

func errorsFromListFunctions(r *api.GetV1FunctionsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetFunctionSource(r *api.GetV1DeploymentsDeploymentIdFunctionSourceResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDeployFunction(r *api.PostV1DeploymentsDeploymentIdFunctionDeployResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromMoveToProject(r *api.PutV1DeploymentsDeploymentIdProjectResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
