package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file wraps the repository-source surface: the read-only /v1/vcs/*
// listings (connected GitHub/GitLab accounts, their repositories and branches)
// and PUT/DELETE /v1/deployments/{id}/source, which connect, switch or clear an
// app's repository. Connecting a new account is dashboard-only; ConnectUrl on
// VcsConnections points there. Same wrapper pattern as deployment_write.go.

// ListVcsConnections calls GET /v1/vcs/connections.
func (c *Client) ListVcsConnections(ctx context.Context) (*api.VcsConnections, error) {
	resp, err := c.inner.GetV1VcsConnectionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromVcsConnections(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ListVcsRepositories calls GET /v1/vcs/repos. connectionID narrows the list
// to one connected account; "" lists across all of them.
func (c *Client) ListVcsRepositories(ctx context.Context, connectionID string) ([]api.VcsRepository, error) {
	params := &api.GetV1VcsReposParams{}
	if connectionID != "" {
		params.ConnectionId = &connectionID
	}
	resp, err := c.inner.GetV1VcsReposWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromVcsRepos(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Repositories, nil
}

// ListVcsBranches calls GET /v1/vcs/branches for repository (owner/repo).
// connectionID picks the account to read through; "" lets the server pick one
// that can see the repository.
func (c *Client) ListVcsBranches(ctx context.Context, repository, connectionID string) (*api.VcsBranches, error) {
	params := &api.GetV1VcsBranchesParams{Repository: repository}
	if connectionID != "" {
		params.ConnectionId = &connectionID
	}
	resp, err := c.inner.GetV1VcsBranchesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromVcsBranches(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// SetDeploymentSource calls PUT /v1/deployments/{id}/source, connecting or
// switching the app's repository and (unless body.Deploy is false) building it.
// When no connected account can see the repository the server answers
// PRECONDITION_FAILED with the dashboard link in the message.
func (c *Client) SetDeploymentSource(ctx context.Context, id string, body api.SetDeploymentSourceBody) (*api.SetDeploymentSourceResult, error) {
	resp, err := c.inner.PutV1DeploymentsDeploymentIdSourceWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromSetSource(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ClearDeploymentSource calls DELETE /v1/deployments/{id}/source, detaching
// the repository. The app keeps running its current version.
func (c *Client) ClearDeploymentSource(ctx context.Context, id string) (*api.ClearDeploymentSourceResult, error) {
	resp, err := c.inner.DeleteV1DeploymentsDeploymentIdSourceWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromClearSource(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromVcsConnections(r *api.GetV1VcsConnectionsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromVcsRepos(r *api.GetV1VcsReposResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromVcsBranches(r *api.GetV1VcsBranchesResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromSetSource(r *api.PutV1DeploymentsDeploymentIdSourceResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromClearSource(r *api.DeleteV1DeploymentsDeploymentIdSourceResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}
