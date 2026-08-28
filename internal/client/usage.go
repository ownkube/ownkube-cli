package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// GetUsageCurrent calls GET /v1/usage/current — the live burn rate.
func (c *Client) GetUsageCurrent(ctx context.Context) (*api.CurrentUsageResponse, error) {
	resp, err := c.inner.GetV1UsageCurrentWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUsageCurrent(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetUsageMonthToDate calls GET /v1/usage/month-to-date.
func (c *Client) GetUsageMonthToDate(ctx context.Context) (*api.MonthToDateUsageResponse, error) {
	resp, err := c.inner.GetV1UsageMonthToDateWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUsageMonthToDate(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetUsageProjected calls GET /v1/usage/projected — the straight-line month
// projection.
func (c *Client) GetUsageProjected(ctx context.Context) (*api.ProjectedMonthCostResponse, error) {
	resp, err := c.inner.GetV1UsageProjectedWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUsageProjected(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetUsageHistory calls GET /v1/usage/history. The points carry union-typed
// period boundaries, so callers typically render this with -o json.
func (c *Client) GetUsageHistory(ctx context.Context) (*api.UsageHistoryResponse, error) {
	resp, err := c.inner.GetV1UsageHistoryWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUsageHistory(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromUsageCurrent(r *api.GetV1UsageCurrentResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUsageMonthToDate(r *api.GetV1UsageMonthToDateResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUsageProjected(r *api.GetV1UsageProjectedResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUsageHistory(r *api.GetV1UsageHistoryResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
