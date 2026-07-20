package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file holds the cluster lifecycle write surface (create / destroy /
// cancel) plus the lightweight status poll. Read wrappers (list / get) live in
// client.go. Every method mirrors that file's pattern: call the generated
// WithResponse method, run checkError over the per-status error variants, then
// guard the success payload.

// CreateCluster calls POST /v1/clusters. Returns immediately with the new
// cluster's id and a `creating` status; poll ClusterStatus until it settles.
func (c *Client) CreateCluster(ctx context.Context, body api.PostV1ClustersJSONRequestBody) (*api.CreateClusterResponse, error) {
	resp, err := c.inner.PostV1ClustersWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCreateCluster(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DestroyCluster calls DELETE /v1/clusters/{id}. When force is true the
// teardown proceeds even if the cluster still has active deployments.
func (c *Client) DestroyCluster(ctx context.Context, id string, force bool) (*api.DestroyClusterResponse, error) {
	params := &api.DeleteV1ClustersClusterIdParams{}
	if force {
		f := api.True
		params.Force = &f
	}
	resp, err := c.inner.DeleteV1ClustersClusterIdWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDestroyCluster(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CancelCluster calls POST /v1/clusters/{id}/cancel. Valid only while the
// cluster is still `creating`.
func (c *Client) CancelCluster(ctx context.Context, id string) (*api.CancelClusterResponse, error) {
	resp, err := c.inner.PostV1ClustersClusterIdCancelWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCancelCluster(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ClusterStatus calls GET /v1/clusters/{id}/status — the lightweight poll
// target during create/destroy.
func (c *Client) ClusterStatus(ctx context.Context, id string) (*api.ClusterStatusResponse, error) {
	resp, err := c.inner.GetV1ClustersClusterIdStatusWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromClusterStatus(resp), resp.Body); err != nil {
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

func errorsFromCreateCluster(r *api.PostV1ClustersResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDestroyCluster(r *api.DeleteV1ClustersClusterIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCancelCluster(r *api.PostV1ClustersClusterIdCancelResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromClusterStatus(r *api.GetV1ClustersClusterIdStatusResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
