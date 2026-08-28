package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// ListCustomDomains calls GET /v1/deployments/{id}/custom-domains and returns
// the linked hosts plus the parent wildcards available to link under.
func (c *Client) ListCustomDomains(ctx context.Context, deploymentID string) (*api.ListCustomDomainsResponse, error) {
	resp, err := c.inner.GetV1DeploymentsDeploymentIdCustomDomainsWithResponse(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromListCustomDomains(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// LinkCustomDomain calls POST /v1/deployments/{id}/custom-domains, linking a
// host (subdomain or apex of a parent wildcard) to the deployment.
func (c *Client) LinkCustomDomain(ctx context.Context, deploymentID, hostname string) (*api.LinkedCustomDomain, error) {
	body := api.LinkCustomDomainBody{Hostname: hostname}
	resp, err := c.inner.PostV1DeploymentsDeploymentIdCustomDomainsWithResponse(ctx, deploymentID, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromLinkCustomDomain(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return &resp.JSON200.Domain, nil
}

// VerifyCustomDomain calls POST /v1/custom-domains/{domainId}/verify and
// returns the current certificate status.
func (c *Client) VerifyCustomDomain(ctx context.Context, domainID string) (string, error) {
	resp, err := c.inner.PostV1CustomDomainsDomainIdVerifyWithResponse(ctx, domainID)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromVerifyCustomDomain(resp), resp.Body); err != nil {
		return "", err
	}
	if resp.JSON200 == nil {
		return "", unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.CertStatus, nil
}

// UnlinkCustomDomain calls DELETE /v1/custom-domains/{domainId}.
func (c *Client) UnlinkCustomDomain(ctx context.Context, domainID string) error {
	resp, err := c.inner.DeleteV1CustomDomainsDomainIdWithResponse(ctx, domainID)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUnlinkCustomDomain(resp), resp.Body); err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return nil
}

func errorsFromListCustomDomains(r *api.GetV1DeploymentsDeploymentIdCustomDomainsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromLinkCustomDomain(r *api.PostV1DeploymentsDeploymentIdCustomDomainsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromVerifyCustomDomain(r *api.PostV1CustomDomainsDomainIdVerifyResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUnlinkCustomDomain(r *api.DeleteV1CustomDomainsDomainIdResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
