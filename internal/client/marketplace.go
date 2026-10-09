package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file holds the marketplace wrappers: the catalog of ready-made
// open-source apps, a dry-run check, deploying one (every piece in one call),
// and removing a deployed app by its install id.

// ListMarketplaceApps calls GET /v1/marketplace/apps and returns the catalog.
func (c *Client) ListMarketplaceApps(ctx context.Context) ([]api.MarketplaceApp, error) {
	resp, err := c.inner.GetV1MarketplaceAppsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListMarketplaceApps(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Apps, nil
}

// GetMarketplaceApp calls GET /v1/marketplace/apps/{slug} and returns one app
// with the inputs to collect and the variables it sets for you.
func (c *Client) GetMarketplaceApp(ctx context.Context, slug string) (*api.MarketplaceAppDetail, error) {
	resp, err := c.inner.GetV1MarketplaceAppsSlugWithResponse(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetMarketplaceApp(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CheckMarketplaceApp calls POST /v1/marketplace/apps/{slug}/check: a dry run
// of the inputs and name, creating nothing.
func (c *Client) CheckMarketplaceApp(ctx context.Context, slug string, body api.MarketplaceCheckBody) (*api.MarketplaceCheckResponse, error) {
	resp, err := c.inner.PostV1MarketplaceAppsSlugCheckWithResponse(ctx, slug, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCheckMarketplaceApp(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeployMarketplaceApp calls POST /v1/marketplace/apps/{slug}/deploy, creating
// every piece of the app (app, database, cache) wired together.
func (c *Client) DeployMarketplaceApp(ctx context.Context, slug string, body api.MarketplaceDeployBody) (*api.MarketplaceDeployResponse, error) {
	resp, err := c.inner.PostV1MarketplaceAppsSlugDeployWithResponse(ctx, slug, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeployMarketplaceApp(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteMarketplaceApp calls DELETE /v1/marketplace/installs/{id}, removing
// every deployment one marketplace deploy created.
func (c *Client) DeleteMarketplaceApp(ctx context.Context, templateInstanceID string) (*api.MarketplaceDeleteResponse, error) {
	resp, err := c.inner.DeleteV1MarketplaceInstallsTemplateInstanceIdWithResponse(ctx, templateInstanceID)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeleteMarketplaceApp(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromListMarketplaceApps(r *api.GetV1MarketplaceAppsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromGetMarketplaceApp(r *api.GetV1MarketplaceAppsSlugResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromCheckMarketplaceApp(r *api.PostV1MarketplaceAppsSlugCheckResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromDeployMarketplaceApp(r *api.PostV1MarketplaceAppsSlugDeployResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}

func errorsFromDeleteMarketplaceApp(r *api.DeleteV1MarketplaceInstallsTemplateInstanceIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON429, r.JSON500}
}
