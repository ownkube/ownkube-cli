package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// ListRegions calls GET /v1/regions and returns the Ownkube Compute region
// catalog (the ids usable as the `region` field when creating a deployment
// with no clusterId).
func (c *Client) ListRegions(ctx context.Context) (*api.RegionListResponse, error) {
	resp, err := c.inner.GetV1RegionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListRegions(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromListRegions(r *api.GetV1RegionsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
