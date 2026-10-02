package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// This file holds the datastore wrappers: the unified database/cache
// connection-info read, linking a datastore into an app, re-applying a
// deployment's current settings (resync), and the Compute box catalog.

// ConnectionInfo calls GET /v1/deployments/{id}/connection-info and returns a
// database's or cache's full connection info, including the password and the
// in-environment (uri) and public (publicUri) connection strings.
func (c *Client) ConnectionInfo(ctx context.Context, id string) (*api.ConnectionInfo, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdConnectionInfoWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromConnectionInfo(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// LinkDatastore calls POST /v1/deployments/{appId}/link, writing the
// datastore's connection string into the app's env as a secret variable.
// envVar overrides the default variable name when set.
func (c *Client) LinkDatastore(ctx context.Context, appID, datastoreID string, envVar *string) (*api.LinkDatastoreResult, error) {
	body := api.PostV1DeploymentsDeploymentIdLinkJSONRequestBody{DatastoreId: datastoreID, EnvVar: envVar}
	resp, err := c.inner.PostV1DeploymentsDeploymentIdLinkWithResponse(ctx, appID, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromLinkDatastore(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Resync calls POST /v1/deployments/{id}/resync, re-applying the deployment's
// current settings with no config change and no new revision.
func (c *Client) Resync(ctx context.Context, id string) (*api.ResyncResult, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdResyncWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromResync(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ListBoxes calls GET /v1/boxes and returns the Compute box catalog: the
// reserved sizes for web apps, databases and caches (pass a box id as skuId
// when creating a database or cache) plus the database storage rate.
func (c *Client) ListBoxes(ctx context.Context) (*api.ComputeBoxCatalog, error) {
	resp, err := c.inner.GetV1BoxesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListBoxes(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromConnectionInfo(r *api.GetV1DeploymentsDeploymentIdConnectionInfoResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromLinkDatastore(r *api.PostV1DeploymentsDeploymentIdLinkResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromResync(r *api.PostV1DeploymentsDeploymentIdResyncResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListBoxes(r *api.GetV1BoxesResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
