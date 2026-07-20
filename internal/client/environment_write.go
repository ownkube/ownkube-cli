package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// Environment write surface (create / update / set-env / delete). Read wrappers
// live in client.go. Unlike deployments, the environment bodies are plain
// structs (no OpenAPI oneOf), so these use the typed WithResponse variants
// directly instead of raw JSON bytes.

// CreateEnvironment calls POST /v1/environments.
func (c *Client) CreateEnvironment(ctx context.Context, body api.CreateEnvironmentBody) (*api.Environment, error) {
	resp, err := c.inner.PostV1EnvironmentsWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCreateEnvironment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateEnvironment calls PATCH /v1/environments/{id}.
func (c *Client) UpdateEnvironment(ctx context.Context, id string, body api.UpdateEnvironmentBody) (*api.Environment, error) {
	resp, err := c.inner.PatchV1EnvironmentsEnvironmentIdWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateEnvironment(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetEnvVars calls PUT /v1/environments/{id}/env. The body replaces the full
// shared env-var set and redeploys every cluster-bound app in the environment.
func (c *Client) SetEnvVars(ctx context.Context, id string, vars []api.EnvVarInput) (*api.SetEnvVarsResponse, error) {
	resp, err := c.inner.PutV1EnvironmentsEnvironmentIdEnvWithResponse(ctx, id, api.SetEnvVarsBody{Env: vars})
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromSetEnvVars(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteEnvironment calls DELETE /v1/environments/{id} (soft delete).
func (c *Client) DeleteEnvironment(ctx context.Context, id string) error {
	resp, err := c.inner.DeleteV1EnvironmentsEnvironmentIdWithResponse(ctx, id)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeleteEnvironment(resp), resp.Body); err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Error adapters (see client.go for the shared checkError contract)
// ---------------------------------------------------------------------------

func errorsFromCreateEnvironment(r *api.PostV1EnvironmentsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUpdateEnvironment(r *api.PatchV1EnvironmentsEnvironmentIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromSetEnvVars(r *api.PutV1EnvironmentsEnvironmentIdEnvResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDeleteEnvironment(r *api.DeleteV1EnvironmentsEnvironmentIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
