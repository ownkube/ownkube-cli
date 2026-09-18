package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// Project surface (list / get / create / update / delete). Projects are the
// top-level workspace grouping (Org -> Project -> Environment -> App). The
// bodies are plain structs (no OpenAPI oneOf), so these use the typed
// WithResponse variants directly instead of raw JSON bytes.

// ListProjects calls GET /v1/projects.
func (c *Client) ListProjects(ctx context.Context) ([]api.Project, error) {
	resp, err := c.inner.GetV1ProjectsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListProjects(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Projects, nil
}

// GetProject calls GET /v1/projects/{id}.
func (c *Client) GetProject(ctx context.Context, id string) (*api.Project, error) {
	resp, err := c.inner.GetV1ProjectsProjectIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetProject(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CreateProject calls POST /v1/projects.
func (c *Client) CreateProject(ctx context.Context, body api.CreateProjectBody) (*api.Project, error) {
	resp, err := c.inner.PostV1ProjectsWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCreateProject(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateProject calls PATCH /v1/projects/{id}.
func (c *Client) UpdateProject(ctx context.Context, id string, body api.UpdateProjectBody) (*api.Project, error) {
	resp, err := c.inner.PatchV1ProjectsProjectIdWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateProject(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteProject calls DELETE /v1/projects/{id} (soft delete, gated server-side:
// never the Default, never with a live deployment).
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	resp, err := c.inner.DeleteV1ProjectsProjectIdWithResponse(ctx, id)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeleteProject(resp), resp.Body); err != nil {
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

func errorsFromListProjects(r *api.GetV1ProjectsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetProject(r *api.GetV1ProjectsProjectIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCreateProject(r *api.PostV1ProjectsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUpdateProject(r *api.PatchV1ProjectsProjectIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDeleteProject(r *api.DeleteV1ProjectsProjectIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
