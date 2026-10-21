package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file wraps an app's variables: GET/PATCH /v1/deployments/{id}/variables
// (secret values are never returned there) and POST .../variables/resolve,
// which returns real values for a local `okctl run`. Same wrapper pattern as
// deployment_write.go.

// ListVariables calls GET /v1/deployments/{id}/variables: the app's own
// variables plus the ones its environment shares.
func (c *Client) ListVariables(ctx context.Context, id string) ([]api.AppVariable, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdVariablesWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListVariables(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Variables, nil
}

// UpdateVariables calls PATCH /v1/deployments/{id}/variables. One call is one
// revision; Changed is false when every value already matched.
func (c *Client) UpdateVariables(ctx context.Context, id string, body api.UpdateAppVariablesBody) (*api.UpdateAppVariablesResult, error) {
	resp, err := c.inner.PatchV1DeploymentsDeploymentIdVariablesWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateVariables(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ResolveVariables calls POST /v1/deployments/{id}/variables/resolve and
// returns every variable with its real value, secrets included. Callers hand
// the values to a child process only: never print them or write them out.
func (c *Client) ResolveVariables(ctx context.Context, id string, purpose api.PostV1DeploymentsDeploymentIdVariablesResolveParamsPurpose) (*api.ResolvedAppVariables, error) {
	params := &api.PostV1DeploymentsDeploymentIdVariablesResolveParams{Purpose: purpose}
	resp, err := c.inner.PostV1DeploymentsDeploymentIdVariablesResolveWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromResolveVariables(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromListVariables(r *api.GetV1DeploymentsDeploymentIdVariablesResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromUpdateVariables(r *api.PatchV1DeploymentsDeploymentIdVariablesResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromResolveVariables(r *api.PostV1DeploymentsDeploymentIdVariablesResolveResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}
