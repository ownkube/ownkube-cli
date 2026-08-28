package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// ListAlertRules calls GET /v1/deployments/{id}/alerts.
func (c *Client) ListAlertRules(ctx context.Context, deploymentID string) ([]api.AlertRule, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdAlertsWithResponse(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListAlertRules(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Rules, nil
}

// CreateAlertRule calls POST /v1/deployments/{id}/alerts.
func (c *Client) CreateAlertRule(ctx context.Context, deploymentID string, body api.PostV1DeploymentsDeploymentIdAlertsJSONRequestBody) (*api.AlertRule, error) {
	resp, err := c.inner.PostV1DeploymentsDeploymentIdAlertsWithResponse(ctx, deploymentID, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCreateAlertRule(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateAlertRule calls PATCH /v1/alert-rules/{ruleId}.
func (c *Client) UpdateAlertRule(ctx context.Context, ruleID string, body api.PatchV1AlertRulesRuleIdJSONRequestBody) (*api.AlertRule, error) {
	resp, err := c.inner.PatchV1AlertRulesRuleIdWithResponse(ctx, ruleID, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateAlertRule(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteAlertRule calls DELETE /v1/alert-rules/{ruleId}.
func (c *Client) DeleteAlertRule(ctx context.Context, ruleID string) error {
	resp, err := c.inner.DeleteV1AlertRulesRuleIdWithResponse(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeleteAlertRule(resp), resp.Body); err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return nil
}

// AlertFiringsFilter narrows the firing list by deployment and count.
type AlertFiringsFilter struct {
	DeploymentID string
	Limit        int
}

// ListAlertFirings calls GET /v1/alert-firings.
func (c *Client) ListAlertFirings(ctx context.Context, f AlertFiringsFilter) ([]api.AlertFiring, error) {
	params := &api.GetV1AlertFiringsParams{}
	if f.DeploymentID != "" {
		params.DeploymentId = &f.DeploymentID
	}
	if f.Limit > 0 {
		params.Limit = &f.Limit
	}
	resp, err := c.inner.GetV1AlertFiringsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListAlertFirings(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Firings, nil
}

func errorsFromListAlertRules(r *api.GetV1DeploymentsDeploymentIdAlertsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCreateAlertRule(r *api.PostV1DeploymentsDeploymentIdAlertsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUpdateAlertRule(r *api.PatchV1AlertRulesRuleIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDeleteAlertRule(r *api.DeleteV1AlertRulesRuleIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromListAlertFirings(r *api.GetV1AlertFiringsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
